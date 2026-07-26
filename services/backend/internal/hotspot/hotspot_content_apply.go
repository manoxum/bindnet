// hotspot_content_apply.go liga os planos de conteudo ao runtime: publica
// o mapa IP->plano dos clientes ao vivo (lido pelo dns-provider para o
// bloqueio de dominio) e fornece as regras de IP/CIDR do plano para a
// zona WAN do firewall (bloqueio por IP). Bloqueio de dominio e feito no
// dns-provider; bloqueio de IP no worker - aqui so o backend compila e
// distribui o estado desejado, mesma filosofia de isolamento/firewall.
package hotspot

import (
	"bindnet/backend/internal/hotspot/store"
	"bindnet/backend/internal/workerapi"
	"context"
	"database/sql"
	"log"
)

// applyContentLive reaplica o conteudo agora (best-effort): republica o
// mapa IP->plano e reaplica as zonas do firewall (que passam a incluir as
// regras de IP do plano). Chamado em toda mutacao de plano/regra/vinculo.
func applyContentLive(ctx context.Context, db *sql.DB, worker *workerapi.Client) {
	if err := publishContentBindings(ctx, db, worker); err != nil {
		log.Printf("[backend] publicacao do mapa IP->plano de conteudo falhou: %v", err)
	}
	applyFirewallLive(ctx, db, worker)
	applyDNSForceLive(ctx, db, worker)
}

// publishContentBindings monta o mapa IP->plano dos clientes conectados
// agora (plano efetivo device->perfil) e grava em
// hotspot_content_client_bindings, de onde o dns-provider le. Clientes
// sem plano efetivo simplesmente nao entram (o dns-provider trata
// ausencia como "sem bloqueio").
func publishContentBindings(ctx context.Context, db *sql.DB, worker *workerapi.Client) error {
	iface, err := store.HotspotWifiInterface(ctx, db)
	if err != nil {
		return err
	}
	clients, err := liveHotspotClients(ctx, worker, iface)
	if err != nil {
		return err
	}
	bindings := make([]store.ContentClientBinding, 0, len(clients))
	for _, client := range clients {
		if client.IP == "" {
			continue
		}
		planID, err := effectiveContentPlanID(db, client.MAC)
		if err != nil {
			return err
		}
		if planID == nil {
			continue
		}
		bindings = append(bindings, store.ContentClientBinding{IP: client.IP, MAC: client.MAC, PlanID: *planID})
	}
	return store.ReplaceContentClientBindings(db, bindings)
}

// contentWANRules compila as regras de IP/CIDR dos planos de conteudo dos
// clientes conectados em entradas da zona WAN (casadas por MAC de
// origem + host de destino) - para a zona WAN aplicar via iptables no
// worker, reusando o mesmo motor do firewall por zonas. So carrega as
// regras de cada plano uma vez, mesmo com varios dispositivos no mesmo
// plano.
func contentWANRules(db *sql.DB, clients []isolationClient) ([]firewallZoneRule, error) {
	planByMAC := make(map[string]string, len(clients))
	planIDs := map[string]bool{}
	for _, client := range clients {
		planID, err := effectiveContentPlanID(db, client.MAC)
		if err != nil {
			return nil, err
		}
		if planID != nil {
			planByMAC[client.MAC] = *planID
			planIDs[*planID] = true
		}
	}
	ipRulesByPlan := make(map[string][]store.ContentRule, len(planIDs))
	for planID := range planIDs {
		rules, err := store.ListContentRules(db, planID)
		if err != nil {
			return nil, err
		}
		for _, rule := range rules {
			if rule.Kind == "ip" {
				ipRulesByPlan[planID] = append(ipRulesByPlan[planID], rule)
			}
		}
	}
	var out []firewallZoneRule
	for mac, planID := range planByMAC {
		for _, rule := range ipRulesByPlan[planID] {
			action := "deny"
			if rule.Action == "allow" {
				action = "allow"
			}
			// Protocol "any" (nao ""): o worker (sanitizeZoneRules) rejeita
			// protocolo vazio, o que faria o apply inteiro do firewall
			// falhar. DstPorts vazio = todas as portas.
			out = append(out, firewallZoneRule{MAC: mac, Protocol: "any", DstHost: rule.Value, Action: action})
		}
	}
	return out, nil
}
