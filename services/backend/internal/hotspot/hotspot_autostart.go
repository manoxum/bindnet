package hotspot

import (
	"bindnet/backend/internal/audit"
	"bindnet/backend/internal/hotspot/store"
	"bindnet/backend/internal/workerapi"
	"context"
	"database/sql"
	"log"
	"net/http"
	"time"
)

// AutoStartHotspotOnBoot sobe o hotspot sozinho quando o backend
// arranca (container recriado, ou reboot da maquina), mas so quando o
// operador ligou o interruptor "iniciar automaticamente no arranque"
// no painel - ver store.HotspotAutostartEnabled em
// hotspot_config_state.go e as rotas em hotspot_autostart_routes.go.
// Sem isso, o container do hotspot sobe em modo "manager" e fica
// ocioso ate alguem clicar em "Iniciar" no painel de novo.
//
// Interruptor LIGADO significa subir SEMPRE, sem consultar a ultima
// intencao do admin (store.HotspotDesiredStateRunning): "auto-start
// sim" tem que ser previsivel, e um hotspot que nao volta depois de
// uma falta de luz so porque estava parado naquele instante seria
// exatamente a surpresa que o interruptor existe pra eliminar. Quem
// nao quer isso poe o interruptor em "nao" (o padrao).
//
// Roda em goroutine de fundo (chamada assim em main.go) com retry
// curto: o worker/container do hotspot podem demorar alguns segundos
// para ficar prontos logo apos o backend subir.
func AutoStartHotspotOnBoot(db *sql.DB, worker *workerapi.Client, audit *audit.Client) {
	ctx := context.Background()

	enabled, err := store.HotspotAutostartEnabled(ctx, db)
	if err != nil {
		log.Printf("[backend] autostart do hotspot: falha ao ler o interruptor de arranque automatico: %v", err)
		return
	}
	if !enabled {
		return
	}

	// Grava a intencao antes de tentar: e ela que faz a reconciliacao
	// (recoverHotspotIfDesired) manter o hotspot de pe se ele cair
	// depois, e tambem o que faz as tentativas abaixo continuarem
	// valendo caso o backend reinicie no meio delas.
	if err := store.SetHotspotDesiredState(ctx, db, true); err != nil {
		log.Printf("[backend] autostart do hotspot: falha ao gravar estado desejado (ligado): %v", err)
	}

	var lastErr error
	for attempt := 0; attempt < 10; attempt++ {
		if attempt > 0 {
			time.Sleep(3 * time.Second)
		}

		var status struct {
			Running bool `json:"running"`
		}
		if err := worker.Call(ctx, http.MethodGet, "/hotspot/status", nil, &status); err != nil {
			lastErr = err
			continue
		}
		if status.Running {
			return
		}

		iface, err := currentHotspotInterface(ctx, db)
		if err != nil {
			lastErr = err
			continue
		}
		if err := startHotspotAndReapply(ctx, db, worker, audit, iface, "sistema (autostart)"); err != nil {
			lastErr = err
			continue
		}
		log.Println("[backend] hotspot religado automaticamente apos reinicio do backend")
		return
	}
	log.Printf("[backend] autostart do hotspot desistiu apos varias tentativas: %v", lastErr)
}

// startHotspotAndReapply religa o servico do hotspot e reaplica
// bloqueios/shaping por cima - mesma sequencia usada tanto aqui
// (boot do backend) quanto pela recuperacao automatica em
// reconcileHotspotOnce (hotspot_reconcile.go) quando o hotspot cai
// sozinho (ex.: watchdog de falha de beacon em
// services/worker/hotspot/watchdog.sh) com o admin ainda querendo ele
// ligado.
func startHotspotAndReapply(ctx context.Context, db *sql.DB, worker *workerapi.Client, audit *audit.Client, iface, username string) error {
	// Cliente lento: subir o AP passa pelo mesmo caminho demorado do
	// painel (ver slowTimeout em internal/workerapi/client.go).
	if err := startHotspotRuntimeConfig(ctx, db, worker.Slow(), StartReasonAuto); err != nil {
		return err
	}
	reapplyHotspotRules(ctx, db, worker, iface)
	audit.Record(ctx, "hotspot_started", username, nil)
	return nil
}
