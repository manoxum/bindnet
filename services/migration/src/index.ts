// Comando migration: unico job deste servico e rodar
// `prisma migrate deploy` contra o Postgres do stack e encerrar (nunca
// fica de pe como servidor). O services/backend (Go) so usa o driver
// Postgres puro - o Prisma Client (Node-only) nunca roda em producao,
// so a CLI de migrations aqui.
import { spawnSync } from "node:child_process";

console.log("[migration] aplicando migrations do Postgres (prisma migrate deploy)");

const result = spawnSync("npx", ["prisma", "migrate", "deploy"], {
  stdio: "inherit",
});

if (result.status !== 0) {
  console.error("[migration] falha ao aplicar migrations, encerrando com erro");
  process.exit(result.status ?? 1);
}

console.log("[migration] migrations aplicadas com sucesso, encerrando");
process.exit(0);
