# Arquitetura do Bindnet

Bindnet é um stack Docker Compose operado numa máquina física e integra rede
Wi-Fi, DNS local e serviços de administração. A arquitetura detalhada e os
comportamentos de cada componente estão nos documentos temáticos do
[índice SoT](README.md).

## Limites de responsabilidade

- `services/frontend` apresenta o painel administrativo e chama a API.
- `services/backend` mantém o domínio, os dados PostgreSQL e as APIs públicas
  do painel.
- `services/worker/controller` executa operações privilegiadas e específicas
  do host; não deve expor execução arbitrária.
- `services/worker/dns` fornece DNS split-horizon e persiste/cacheia os
  registros locais conforme as regras de domínio.
- `services/migration` aplica o schema Prisma antes dos consumidores de dados.
- Os serviços de infraestrutura (PostgreSQL, MongoDB, Redis e MinIO) mantêm
  estado persistente e não podem ser tratados como descartáveis.

O submódulo [bin/](../bin/README.md) é transversal e reutilizável. Ele não é
uma camada de domínio do Bindnet; o seu contrato reside em `bin/SoT/`.
