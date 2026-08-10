package hotspot

import (
	"bindnet/backend/internal/audit"
	"bindnet/backend/internal/auth"
	"bindnet/backend/internal/hotspot/store"
	"bindnet/backend/internal/workerapi"
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
)

// currentHotspotInterface busca WIFI_INTERFACE configurado pelo painel -
// usada tanto para ligar/desligar o hotspot quanto para listar clientes.
func currentHotspotInterface(ctx context.Context, db *sql.DB) (string, error) {
	return store.HotspotWifiInterface(ctx, db)
}

// detachedHotspotContext e o contexto das acoes de ciclo de vida do
// hotspot (start/stop/apply/recover-wifi), no lugar de r.Context().
//
// Motivo: essas acoes derrubam a rede de quem as pediu. O painel e
// operado tambem do celular conectado PELO PROPRIO hotspot, entao
// parar/reiniciar o AP mata o socket do pedido no meio, o Go cancela
// r.Context(), e tudo que viesse depois (gravar a intencao, devolver a
// placa ao NetworkManager, desmontar chains) falhava com "context
// canceled" - deixando estado pela metade justamente no caminho em que
// ninguem esta olhando. O prazo generoso cobre o pior caso de
// stop_service em services/worker/hotspot/entrypoint.sh (30s de espera
// pelo SIGTERM + force_stop_create_ap) com folga, ver slowTimeout em
// internal/workerapi/client.go.
func detachedHotspotContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 2*time.Minute)
}

// reapplyHotspotRules reinstala todas as regras que vivem por cima do
// AP e se perdem quando ele sobe de novo (bloqueios, shaping,
// isolamento, zonas do firewall). Todas idempotentes no worker e todas
// best-effort - por isso nao devolvem erro. Um unico lugar porque as
// quatro tem que andar sempre juntas: /hotspot/start, /hotspot/apply e
// startHotspotAndReapply (autostart e auto-recuperacao) chamavam a
// mesma sequencia copiada.
func reapplyHotspotRules(ctx context.Context, db *sql.DB, worker *workerapi.Client, iface string) {
	reapplyHotspotBlocklist(ctx, db, worker, iface)
	reapplyHotspotShaping(ctx, worker, iface)
	reapplyHotspotIsolation(ctx, db, worker)
	reapplyHotspotFirewall(ctx, db, worker)
}

func stopHotspotService(ctx context.Context, worker *workerapi.Client) error {
	return worker.Call(ctx, http.MethodPost, "/hotspot/stop", nil, nil)
}

// stopHotspotAndTeardown derruba o servico do hotspot e desmonta tudo
// que so faz sentido com ele no ar, devolvendo a lista de falhas em vez
// de abortar na primeira. Devolver slice vazio = teardown completo.
//
// O best-effort e deliberado: a versao anterior abortava assim que
// /hotspot/stop devolvia erro, e com isso um simples estouro de timeout
// (ver slowTimeout em internal/workerapi/client.go) deixava a placa
// Wi-Fi presa FORA do NetworkManager - ela sumia do menu de rede do
// sistema e so voltava com "Recuperar Wi-Fi". Nenhum erro do worker
// deve conseguir produzir esse estado: o pior caso aceitavel e um
// hotspot que nao morreu, nunca uma placa sequestrada.
//
// iface vazia pula as etapas que dependem de WIFI_INTERFACE (o hotspot
// ainda e parado): sem o nome da placa nao ha o que devolver ao
// NetworkManager nem chains por interface pra remover.
func stopHotspotAndTeardown(ctx context.Context, worker *workerapi.Client, iface string) []string {
	var problems []string
	record := func(what string, err error) {
		if err != nil {
			log.Printf("[backend] parada do hotspot: %s falhou: %v", what, err)
			problems = append(problems, what+": "+err.Error())
		}
	}

	slow := worker.Slow()
	record("parar o servico do hotspot", stopHotspotService(ctx, slow))
	if iface == "" {
		return problems
	}

	// wifi-manage e idempotente (ver handleWifiManage no worker) mesmo
	// quando o /start anterior nao chegou a desgerenciar a placa
	// (cenario AP+STA, ver unmanageWifiInterfaceIfIdle em
	// services/worker/controller/internal/hotspot/service.go) - chamar
	// sempre garante que a placa nunca fica presa "unmanaged".
	record("devolver "+iface+" ao NetworkManager",
		slow.Call(ctx, http.MethodPost, "/network/wifi-manage", map[string]string{"interface": iface}, nil))
	// Higiene: nada disso deve sobreviver com o AP parado.
	record("desmontar isolamento de clientes", teardownIsolationWorker(ctx, worker, iface))
	record("desmontar firewall (wan/local)", teardownFirewallWorker(ctx, worker, iface))
	record("desmontar reforco de DNS", teardownDNSForceWorker(ctx, worker, iface))
	return problems
}

// recoverWifiAdapter e a acao de emergencia do botao "Recuperar Wi-Fi":
// mesmo teardown completo de stopHotspotAndTeardown, so que aqui uma
// falha ao parar o hotspot nunca pode impedir a devolucao da placa ao
// NetworkManager - destravar a placa e justamente o proposito do botao.
// Desgerenciar a placa antes do hotspot subir (quando ela nao esta
// associada como cliente) e feito pelo proprio worker, bem em cima do
// "docker exec ... start" - ver unmanageWifiInterfaceIfIdle em
// services/worker/controller/internal/hotspot/service.go.
func recoverWifiAdapter(ctx context.Context, worker *workerapi.Client, iface string) []string {
	return stopHotspotAndTeardown(ctx, worker, iface)
}

// RegisterHotspotUplinkRoute troca SO a fonte de internet
// (INTERNET_INTERFACE) sem reiniciar o hotspot: grava a chave no banco
// e deixa o monitor de uplink do runner
// (services/worker/hotspot/uplink.sh) detectar a mudanca e alternar o
// NAT ao vivo em ate UPLINK_MONITOR_INTERVAL segundos - clientes
// conectados nao caem. Com o hotspot parado, o valor simplesmente vale
// no proximo start. Usada pelo quick-switch do card de resumo do
// painel; o formulario completo ("Salvar e aplicar") continua
// reiniciando, porque as demais chaves (SSID, senha, canal...) exigem
// subir o hostapd de novo.
func RegisterHotspotUplinkRoute(mux *http.ServeMux, admin *auth.Administrator, audit *audit.Client, db *sql.DB) {
	mux.HandleFunc("POST /api/hotspot/uplink", auth.RequireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Interface string `json:"interface"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Interface) == "" {
			http.Error(w, "campo 'interface' obrigatorio", http.StatusBadRequest)
			return
		}
		if err := store.SaveHotspotConfig(r.Context(), db, map[string]string{"INTERNET_INTERFACE": req.Interface}); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		username, _ := auth.SessionUser(r, admin)
		audit.Record(r.Context(), "hotspot_uplink_switched", username, map[string]any{"interface": req.Interface})
		w.WriteHeader(http.StatusNoContent)
	}))
}
