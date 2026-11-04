"""Arranque do checkout pelo Compose, sem alterar dados persistentes."""

import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import time
from urllib.error import HTTPError, URLError
from urllib.request import ProxyHandler, build_opener


ROOT = Path(__file__).resolve().parents[2]
ADMIN_URL = "https://admin.bindnet.local.com"


def check_tools():
    if os.uname().sysname != "Linux":
        raise RuntimeError("O launcher requer Linux com sessão gráfica.")
    if os.geteuid() == 0:
        raise RuntimeError("Execute como utilizador normal, sem sudo.")
    for tool in ("docker", "xdg-open"):
        if not shutil.which(tool):
            raise RuntimeError(f"Instale {tool} antes de continuar.")
    result = subprocess.run(
        ["docker", "compose", "version"], capture_output=True, check=False,
        timeout=15,
    )
    if result.returncode:
        raise RuntimeError("Instale o plugin Docker Compose v2.")


class Runtime:
    def __init__(self, root=ROOT, report=print, output=None):
        self.root = Path(root)
        self.report = report
        self.output = output
        self.environment = os.environ.copy()
        # As tags devem coincidir com as usadas pelo worker ao ler o mesmo env.
        for name in ("STAGE", "VERSION", "DOCKERHUB"):
            self.environment.pop(name, None)
        # Os caminhos do worker e os nomes dos volumes pertencem a este projeto.
        self.environment["DOCKERFILE"] = "Dockerfile"
        self.command = []
        self.frontend_port = "9090"

    def run(self, arguments, capture=False):
        result = subprocess.run(
            arguments, cwd=self.root, env=self.environment, check=False,
            stdout=subprocess.PIPE if capture else self.output,
            stderr=subprocess.PIPE if capture else subprocess.STDOUT,
            text=True, timeout=30 if capture else None,
        )
        if result.returncode:
            # Config e inspect podem conter segredos: nunca imprimir a captura.
            raise RuntimeError(f"Falha ao executar {' '.join(arguments[:3])}.")
        return result.stdout if capture else ""

    def compose(self, *arguments, capture=False):
        return self.run(self.command + list(arguments), capture)

    def validate(self):
        self.report("A validar Docker e a configuração do projeto…")
        check_tools()
        try:
            self.run(["docker", "info"], capture=True)
        except (RuntimeError, subprocess.TimeoutExpired) as error:
            raise RuntimeError(
                "Docker indisponível. Inicie o Docker e confirme que o seu utilizador "
                "consegue executar docker info sem sudo."
            ) from error
        promote = (self.root / "promote.yml").read_text()
        match = re.search(r"^current_stage:\s*([\w-]+)\s*$", promote, re.M)
        if not match:
            raise RuntimeError("current_stage inválido em promote.yml.")
        candidates = (f".env.{match[1]}", ".env")
        env_file = next((name for name in candidates if (self.root / name).is_file()), None)
        if not env_file:
            raise RuntimeError(
                "Falta o ambiente. Execute scripts/gen-secrets.sh --out "
                f"{candidates[0]} e reveja as configurações antes de iniciar."
            )
        self.environment["BINDNET_ENV_FILE"] = env_file
        self.command = [
            "docker", "compose", "--project-name", "bindnet",
            "--project-directory", str(self.root), "--env-file", env_file,
            "-f", "docker-compose.yml", "-f", "docker-compose.assets.build.yml",
        ]
        self.compose("config", "--quiet", capture=True)
        config = json.loads(self.compose("config", "--format", "json", capture=True))
        services = config["services"]
        for service, keys in {
            "backend": ("POSTGRES_PASSWORD", "MONGO_PASSWORD"),
            "minio": ("MINIO_ROOT_USER", "MINIO_ROOT_PASSWORD"),
            "redis": ("REDIS_PASSWORD",),
        }.items():
            values = services[service].get("environment", {})
            for key in keys:
                value = values.get(key, "")
                if not value:
                    raise RuntimeError(f"Configure {key} no arquivo de ambiente.")
        admin = services["backend"]["environment"]
        username, password = admin.get("ADMIN_USERNAME"), admin.get("ADMIN_PASSWORD")
        if bool(username) != bool(password):
            raise RuntimeError("Configure ADMIN_USERNAME e ADMIN_PASSWORD em conjunto.")
        # Ambos vazios permitem reutilizar o administrador já persistido pelo backend.
        self.frontend_port = services["frontend"]["environment"]["FRONTEND_PORT"]
        # O worker usa o mesmo env_file ao aplicar hotspot/DNS posteriormente.
        self.report("Docker disponível; configuração validada.")

    def start(self):
        self.report("A compilar e validar as imagens do Bindnet…")
        # Build antes de qualquer recriação; o cache do Docker evita trabalho repetido.
        self.compose("build")
        self.report("A preparar as redes Docker externas…")
        for network in ("wnet", "dbnet"):
            result = subprocess.run(
                ["docker", "network", "inspect", network], cwd=self.root,
                env=self.environment, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
            )
            if result.returncode:
                self.run(["docker", "network", "create", network])
        self.report("A iniciar os serviços do Bindnet…")
        self.compose("up", "--detach", "--no-build")
        self.report("A aguardar o painel, o DNS e o HTTPS…")
        self.wait_ready()

    def wait_ready(self, timeout=180):
        # A URL final valida também resolução local, gateway e confiança na CA.
        opener = build_opener(ProxyHandler({}))
        deadline = time.monotonic() + timeout
        reason = "O painel ainda não está disponível."
        while time.monotonic() < deadline:
            try:
                with opener.open(ADMIN_URL + "/", timeout=3) as response:
                    page = response.read(65536)
                    if response.geturl() != ADMIN_URL + "/" or b"<title>Bindnet</title>" not in page:
                        raise ValueError("O domínio administrativo não está a servir o painel Bindnet.")
                try:
                    opener.open(ADMIN_URL + "/api/auth/me", timeout=3).close()
                except HTTPError as error:
                    with error:
                        if error.code != 401 or error.read(1024).strip() != b"nao autenticado":
                            raise ValueError("A API administrativa ainda não está disponível.") from error
                    return
                raise ValueError("Resposta inesperada da API administrativa.")
            except (OSError, URLError, ValueError) as error:
                reason = str(error)
            time.sleep(2)
        raise RuntimeError(
            f"O endereço {ADMIN_URL} não ficou pronto: {reason}\n"
            "Confirme DNS local, site/certificado no nginx-ui e confiança na CA. "
            f"Para configurar, aceda a http://127.0.0.1:{self.frontend_port}. "
            "Os serviços iniciados permanecem em execução."
        )

    def open_browser(self):
        self.report(f"A abrir {ADMIN_URL} no navegador…")
        self.run(["xdg-open", ADMIN_URL])
