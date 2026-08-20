// hotspot_messages_push.go faz o "push" de avisos urgentes reusando o
// portal cativo por-MAC ja existente do bloqueio de credito/cota
// (applyCaptivePortalRedirect em hotspot_credit_recharge.go) - o
// dispositivo recebe o balao "Entrar na rede" do proprio SO e cai no
// bindnet.local.com, onde ve o aviso. NAO ha alteracao de DNS nem interceptacao L7.
// O redirect e "possuido" originalmente pelos reconciles de credito/cota;
// por isso, ao desligar o push de um aviso, so removemos o redirect se
// nenhum desses bloqueios ainda o quiser ligado.
package hotspot

import (
	"bindnet/backend/internal/hotspot/store"
	"bindnet/backend/internal/workerapi"
	"context"
	"database/sql"
	"log"
)

// reconcileMessageCaptivePush ajusta o push via portal cativo conforme os
// avisos urgentes pendentes. targetMAC vazio (broadcast) reconcilia todos
// os dispositivos conectados agora; preenchido, so aquele MAC.
// Best-effort: so loga em falha, ja que a entrega base do aviso e sempre
// o portal em bindnet.local.com.
func reconcileMessageCaptivePush(ctx context.Context, db *sql.DB, worker *workerapi.Client, targetMAC string) {
	macs, err := pushReconcileTargets(ctx, db, worker, targetMAC)
	if err != nil {
		log.Printf("[backend] avisos: falha ao listar alvos do push do portal cativo: %v", err)
		return
	}
	if len(macs) == 0 {
		return
	}
	credit, err := hotspotCreditBlockedSet(db)
	if err != nil {
		log.Printf("[backend] avisos: falha ao ler bloqueios de credito para o push: %v", err)
		return
	}
	quota, err := store.HotspotQuotaBlockedSet(db)
	if err != nil {
		log.Printf("[backend] avisos: falha ao ler bloqueios de cota para o push: %v", err)
		return
	}
	for _, mac := range macs {
		want, err := store.HasActiveUrgentForMAC(db, mac)
		if err != nil {
			log.Printf("[backend] avisos: falha ao avaliar aviso urgente de %s: %v", mac, err)
			continue
		}
		if want {
			applyCaptivePortalRedirect(ctx, db, worker, mac, true)
			continue
		}
		// Sem aviso urgente pendente: so desliga se o portal cativo nao
		// estiver sendo mantido pelo bloqueio de credito/cota (donos
		// originais desse redirect).
		if credit[mac] || quota[mac] {
			continue
		}
		applyCaptivePortalRedirect(ctx, db, worker, mac, false)
	}
}

// pushReconcileTargets resolve para quais MACs reconciliar o push: um so
// (aviso direcionado) ou todos os conectados agora (broadcast).
func pushReconcileTargets(ctx context.Context, db *sql.DB, worker *workerapi.Client, targetMAC string) ([]string, error) {
	if targetMAC != "" {
		return []string{targetMAC}, nil
	}
	iface, err := store.HotspotWifiInterface(ctx, db)
	if err != nil {
		return nil, err
	}
	clients, err := liveHotspotClients(ctx, worker, iface)
	if err != nil {
		return nil, err
	}
	macs := make([]string, 0, len(clients))
	for _, c := range clients {
		macs = append(macs, c.MAC)
	}
	return macs, nil
}
