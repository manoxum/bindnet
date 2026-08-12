package hotspot

import (
	"bindnet/backend/internal/audit"
	"bindnet/backend/internal/auth"
	"bindnet/backend/internal/hotspot/store"
	"bindnet/backend/internal/workerapi"
	"database/sql"
	"net/http"
	"strings"
)

// RegisterHotspotLifecycleRoutes agrupa as quatro rotas que mexem no
// servico AP de verdade (ligar, parar, reaplicar configuracao e
// destravar a placa), separadas das rotas de leitura/configuracao em
// hotspot.go porque compartilham tres decisoes que nenhuma outra rota
// do painel tem:
//
//  1. contexto proprio (detachedHotspotContext) - elas derrubam a rede
//     de quem as pediu;
//  2. cliente lento do worker (workerapi.Client.Slow) - passam pelo
//     stop_service do entrypoint.sh, que estoura os 15s padrao;
//  3. a intencao do admin (_DESIRED_STATE) e gravada ANTES da acao, pra
//     o loop de reconciliacao nunca desfazer o que foi pedido.
//
// Chamada por RegisterHotspotRoutes (hotspot.go).
func RegisterHotspotLifecycleRoutes(mux *http.ServeMux, worker *workerapi.Client, admin *auth.Administrator, audit *audit.Client, db *sql.DB) {
	mux.HandleFunc("POST /api/hotspot/apply", auth.RequireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := detachedHotspotContext()
		defer cancel()
		if err := applyHotspotRuntimeConfig(ctx, db, worker.Slow(), StartReasonManual); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		if iface, err := currentHotspotInterface(ctx, db); err == nil {
			reapplyHotspotRules(ctx, db, worker, iface)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	mux.HandleFunc("POST /api/hotspot/start", auth.RequireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := detachedHotspotContext()
		defer cancel()

		iface, err := currentHotspotInterface(ctx, db)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		// O worker desgerencia a placa fisica no NetworkManager (quando
		// nao ha STA associada) internamente, bem em cima do "docker exec
		// ... start" - ver unmanageWifiInterfaceIfIdle em
		// services/worker/controller/internal/hotspot/service.go. Fazer
		// essa checagem aqui tambem, bem mais cedo, so alargava a janela
		// entre a checagem e a tentativa real do create_ap (o hotspot
		// ainda leva alguns segundos pra rodar de fato:
		// EnsureHotspotContainer, restart do dns-provider, espera do
		// banco), dando tempo de sobra pra uma associacao Wi-Fi marginal
		// cair entre as duas. O container do hotspot fica vivo;
		// ligar/desligar controla apenas o servico AP interno.
		//
		// Intencao ANTES da acao, mesmo motivo do /stop abaixo: se o
		// pedido morrer no meio, o loop de reconciliacao
		// (recoverHotspotIfDesired) e quem termina o servico - mas so se
		// a intencao ja estiver gravada.
		if err := store.SetHotspotDesiredState(ctx, db, true); err != nil {
			http.Error(w, "falha ao gravar a intencao de ligar o hotspot: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if err := startHotspotRuntimeConfig(ctx, db, worker.Slow(), StartReasonManual); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		reapplyHotspotRules(ctx, db, worker, iface)
		username, _ := auth.SessionUser(r, admin)
		audit.Record(ctx, "hotspot_started", username, nil)
		w.WriteHeader(http.StatusNoContent)
	}))

	mux.HandleFunc("POST /api/hotspot/stop", auth.RequireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := detachedHotspotContext()
		defer cancel()

		// Erro aqui nao aborta: sem WIFI_INTERFACE ainda queremos parar o
		// servico, so nao ha placa nomeada pra devolver ao NetworkManager
		// (stopHotspotAndTeardown trata iface vazia).
		iface, _ := currentHotspotInterface(ctx, db)

		// INTENCAO PRIMEIRO, antes de qualquer chamada ao worker. O
		// hotspot volta sozinho em ate 15s se esta gravacao nao
		// acontecer: recoverHotspotIfDesired (hotspot_reconcile.go) ve o
		// AP parado com _DESIRED_STATE ainda em "running" e religa. Era
		// exatamente esse o sintoma "parei e ele voltou" - o handler
		// antigo gravava a intencao no FIM, depois de um teardown que
		// abortava por timeout antes de chegar la.
		if err := store.SetHotspotDesiredState(ctx, db, false); err != nil {
			http.Error(w, "falha ao gravar a intencao de parar o hotspot: "+err.Error(), http.StatusInternalServerError)
			return
		}

		problems := stopHotspotAndTeardown(ctx, worker, iface)
		username, _ := auth.SessionUser(r, admin)
		audit.Record(ctx, "hotspot_stopped", username, nil)
		if len(problems) > 0 {
			http.Error(w, "hotspot marcado como parado, mas com falhas: "+strings.Join(problems, "; "), http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	mux.HandleFunc("POST /api/hotspot/recover-wifi", auth.RequireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := detachedHotspotContext()
		defer cancel()

		iface, err := currentHotspotInterface(ctx, db)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		// Destravar a placa e sempre uma parada: sem gravar a intencao, o
		// loop de reconciliacao religaria o hotspot em cima da placa que
		// acabamos de devolver ao NetworkManager.
		if err := store.SetHotspotDesiredState(ctx, db, false); err != nil {
			http.Error(w, "falha ao gravar a intencao de parar o hotspot: "+err.Error(), http.StatusInternalServerError)
			return
		}
		problems := recoverWifiAdapter(ctx, worker, iface)
		username, _ := auth.SessionUser(r, admin)
		audit.Record(ctx, "wifi_adapter_recovered", username, map[string]any{"interface": iface})
		if len(problems) > 0 {
			http.Error(w, "placa recuperada, mas com falhas: "+strings.Join(problems, "; "), http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
}
