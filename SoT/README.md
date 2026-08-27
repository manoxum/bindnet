# Fonte de verdade — Bindnet

Esta pasta reúne a documentação canónica do Bindnet. Ela descreve o que o
produto faz, os seus limites operacionais e os contratos que não podem ser
alterados acidentalmente. A implementação deve concordar com estes documentos;
quando o comportamento muda intencionalmente, atualize o documento temático
correspondente na mesma alteração.

| Documento | Fonte de verdade para |
| --- | --- |
| [overview.md](overview.md) | propósito e capacidades do produto |
| [hotspot-radio.md](hotspot-radio.md) | criação do AP, rádio, bandas e seleção de canais |
| [hotspot-uplink.md](hotspot-uplink.md) | fonte de internet e modo AP |
| [hotspot-concurrency.md](hotspot-concurrency.md) | coexistência de AP e estação Wi-Fi |
| [hotspot-recovery.md](hotspot-recovery.md) | encerramento, watchdog e auto-recuperação |
| [hotspot-control.md](hotspot-control.md) | comandos do painel para iniciar, parar e recuperar rede |
| [hotspot-clients.md](hotspot-clients.md) | dispositivos, vouchers, portal cativo e avisos |
| [content-and-isolation.md](content-and-isolation.md) | isolamento de clientes e políticas de conteúdo |
| [dns.md](dns.md) | DNS split-horizon, TLDs locais e descoberta |
| [certificates.md](certificates.md) | CA local e gestão de certificados |
| [edge-and-ca-page.md](edge-and-ca-page.md) | nginx-ui, gateway e página pública da CA |
| [data.md](data.md) | PostgreSQL, MongoDB, Redis, MinIO e migrations |
| [admin-panel.md](admin-panel.md) | responsabilidades do painel, backend e worker |
| [cross-cutting.md](cross-cutting.md) | regras que atravessam todo o stack |
| [architecture.md](architecture.md) | composição do sistema, responsabilidades e fronteiras entre serviços |
| [operations.md](operations.md) | operação segura, ambientes e comandos de manutenção |
| [security.md](security.md) | segredos, privilégios, volumes e dados persistentes |
| [bin/](../bin/SoT/README.md) | contrato operacional reutilizável do submódulo `bin` |

`RULE.md` na raiz é somente um ponteiro de compatibilidade; não adicione regras
novas nele. Escolha o documento temático acima ao alterar comportamento.
