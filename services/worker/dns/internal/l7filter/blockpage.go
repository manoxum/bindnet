package l7filter

// blockPageResponse e a resposta HTTP completa servida no proxy HTTP
// quando o host esta bloqueado - status 200 com HTML de aviso (em vez de
// erro), para o usuario entender que foi a politica da rede. Em HTTPS
// nao ha pagina (a conexao e resetada), limitacao aceita por desenho.
const blockPageResponse = "HTTP/1.1 200 OK\r\n" +
	"Content-Type: text/html; charset=utf-8\r\n" +
	"Cache-Control: no-store\r\n" +
	"Connection: close\r\n" +
	"\r\n" + blockPageHTML

const blockPageHTML = `<!doctype html>
<html lang="pt">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Conteúdo bloqueado</title>
<style>
  body{margin:0;font-family:system-ui,Segoe UI,Roboto,Helvetica,Arial,sans-serif;background:#0f172a;color:#e2e8f0;display:flex;min-height:100vh;align-items:center;justify-content:center}
  .card{max-width:34rem;margin:1.5rem;padding:2rem;background:#1e293b;border:1px solid #334155;border-radius:1rem;text-align:center}
  .icon{font-size:3rem;line-height:1}
  h1{font-size:1.35rem;margin:1rem 0 .5rem}
  p{color:#94a3b8;line-height:1.5;margin:.5rem 0}
  .tag{display:inline-block;margin-top:1rem;font-size:.8rem;color:#64748b}
</style>
</head>
<body>
  <div class="card">
    <div class="icon">🛡️</div>
    <h1>Conteúdo bloqueado</h1>
    <p>O acesso a este conteúdo foi bloqueado pelas políticas desta rede.</p>
    <p>Se você acredita que isto é um engano, fale com o responsável pela rede.</p>
    <div class="tag">bindnet · filtro de conteúdo</div>
  </div>
</body>
</html>`
