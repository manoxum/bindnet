# Segurança e dados persistentes

Arquivos `.env` e a pasta `.promote/` podem conter segredos. Eles não podem
ser versionados, incluídos em exemplos reais ou impressos em logs. O fluxo
`bin/promote auth copy` garante a regra `/.promote/` em `.gitignore` antes de
copiar credenciais Docker.

Volumes Compose são dados de produção/localidade. Em especial, o PostgreSQL
guarda a CA local e a sua chave privada; não apague nem recrie volumes sem uma
recuperação aprovada e verificável. A lista completa de cuidados e as regras
funcionais relacionadas permanecem em [data.md](data.md),
[certificates.md](certificates.md) e [cross-cutting.md](cross-cutting.md).

Somente `services/worker` tem capacidades equivalentes a root no host. Novas
operações privilegiadas devem ser rotas específicas, validadas e mínimas;
nunca disponibilize execução arbitrária por API.
