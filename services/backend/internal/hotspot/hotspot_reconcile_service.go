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

// Saude do SERVICO do hotspot (esta ele no ar? deve subir sozinho?),
// separada da reconciliacao por dispositivo em hotspot_reconcile.go -
// dominios distintos e aquele arquivo ja passava do limite de ~200
// linhas (ver CLAUDE.md).

// hotspotStartingGrace e quanto tempo o hotspot pode ficar em
// "starting" (script runner vivo, mas sem nenhuma instancia de
// create_ap no ar) antes da reconciliacao forcar um restart.
//
// Esse estado e legitimo por alguns segundos em todo start normal, e e
// por isso que o worker o reporta separado de "running"/"stopped" (ver
// status_service em services/worker/hotspot/entrypoint.sh). O problema
// e ficar PRESO nele: quando o AP cai por uma causa que o retry local
// nao resolve, o runner continua vivo tentando, o worker continua
// dizendo running=true, e a auto-recuperacao daqui - que so age com
// running=false - nunca disparava. Era exatamente o "o sistema nao
// reconhece que foi parado": o unico jeito de sair era Parar e Iniciar
// na mao pelo painel.
const hotspotStartingGrace = 60 * time.Second

// reconcileHotspotService resolve o estado do servico e trata os dois
// casos em que nao ha o que reconciliar por dispositivo. Devolve true
// so quando o AP esta realmente no ar.
func reconcileHotspotService(ctx context.Context, db *sql.DB, worker *workerapi.Client, audit *audit.Client, startingSince *time.Time) bool {
	var status struct {
		Running bool   `json:"running"`
		Status  string `json:"status"`
	}
	if err := worker.Call(ctx, http.MethodGet, "/hotspot/status", nil, &status); err != nil {
		return false
	}

	if status.Running && status.Status == "starting" {
		forceRestartIfStuckStarting(ctx, db, worker, startingSince)
		return false
	}
	*startingSince = time.Time{}

	if !status.Running {
		// Hotspot parado: ninguem pode estar conectado, fecha toda
		// sessao em aberto (ver closeStaleSessions em
		// hotspot_sessions.go).
		if err := closeStaleSessions(db, nil); err != nil {
			log.Printf("[backend] reconciliacao: falha ao fechar sessoes com hotspot parado: %v", err)
		}
		recoverHotspotIfDesired(ctx, db, worker, audit)
		return false
	}
	return true
}

// forceRestartIfStuckStarting marca o inicio do estado "starting" e,
// se ele passar de hotspotStartingGrace, reinicia o servico.
func forceRestartIfStuckStarting(ctx context.Context, db *sql.DB, worker *workerapi.Client, startingSince *time.Time) {
	if startingSince.IsZero() {
		*startingSince = time.Now()
		return
	}
	if time.Since(*startingSince) < hotspotStartingGrace {
		return
	}
	*startingSince = time.Time{}
	log.Printf("[backend] reconciliacao: hotspot preso em \"starting\" ha mais de %s - forcando restart", hotspotStartingGrace)
	// Reinicia pelo mesmo caminho que o operador usaria na mao (Parar +
	// Iniciar): o "restart" do entrypoint mata o runner preso antes de
	// subir outro. Sem matar o runner, um start novo so responderia
	// "Servico do hotspot ja esta em execucao".
	if err := applyHotspotRuntimeConfig(ctx, db, worker.Slow(), StartReasonAuto); err != nil {
		log.Printf("[backend] reconciliacao: falha ao forcar restart do hotspot preso: %v", err)
	}
}

// recoverHotspotIfDesired religa o hotspot sozinho quando ele cai sem
// que o admin tenha pedido (ex.: watchdog de falha de beacon em
// services/worker/hotspot/watchdog.sh derrubando o create_ap travado)
// - so age se a ultima intencao registrada foi ligar
// (store.HotspotDesiredStateRunning, a mesma usada por
// AutoStartHotspotOnBoot), senao um "parar" deliberado pelo painel
// seria desfeito no proximo ciclo deste loop.
func recoverHotspotIfDesired(ctx context.Context, db *sql.DB, worker *workerapi.Client, audit *audit.Client) {
	desired, err := store.HotspotDesiredStateRunning(ctx, db)
	if err != nil {
		log.Printf("[backend] reconciliacao: falha ao ler estado desejado do hotspot: %v", err)
		return
	}
	if !desired {
		return
	}

	iface, err := currentHotspotInterface(ctx, db)
	if err != nil {
		log.Printf("[backend] reconciliacao: falha ao ler WIFI_INTERFACE para religar o hotspot: %v", err)
		return
	}
	if err := startHotspotAndReapply(ctx, db, worker, audit, iface, "sistema (auto-recuperacao)"); err != nil {
		log.Printf("[backend] reconciliacao: falha ao religar hotspot automaticamente: %v", err)
		return
	}
	log.Println("[backend] hotspot religado automaticamente apos queda detectada pela reconciliacao")
}
