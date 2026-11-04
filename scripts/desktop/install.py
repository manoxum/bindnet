"""Instala o atalho de desenvolvimento no menu do utilizador Linux."""

import os
from pathlib import Path
import shlex
import shutil
import subprocess
import sys

from runtime import ROOT, check_tools


def desktop_escape(value):
    return str(value).replace("\\", "\\\\").replace("\n", "\\n").replace("\r", "\\r")


def exec_quote(value):
    # Exec tem uma camada de escapes própria, além das regras gerais do desktop.
    value = str(value).replace("%", "%%")
    for character in ("\\", '"', "`", "$"):
        value = value.replace(character, "\\" + character)
    return desktop_escape('"' + value + '"')


def install(root=ROOT, home=None, python=None):
    root = Path(root).resolve()
    home = Path(home) if home else Path.home()
    python = python or sys.executable
    # Local canónico do host, mesmo num terminal de IDE instalado por Snap.
    data = home / ".local/share"
    launcher = home / ".local/bin/bindnet-dev"
    application = data / "applications/bindnet-dev.desktop"
    icon = data / "icons/hicolor/scalable/apps/bindnet-dev.svg"
    for path in (launcher, application, icon):
        path.parent.mkdir(parents=True, exist_ok=True)
    shutil.copyfile(root / "services/frontend/public/09-nucleo.svg", icon)
    launcher.write_text(
        "#!/usr/bin/env bash\nset -euo pipefail\n"
        f"cd {shlex.quote(str(root))}\n"
        f"exec {shlex.quote(str(python))} "
        f"{shlex.quote(str(root / 'scripts/desktop/launcher.py'))}\n"
    )
    launcher.chmod(0o755)
    application.write_text(
        "[Desktop Entry]\nVersion=1.0\nType=Application\n"
        "Name=Bindnet (desenvolvimento)\n"
        "Comment=Validar, compilar e abrir o painel de administração Bindnet\n"
        f"Exec={exec_quote(launcher)}\nIcon={desktop_escape(icon)}\n"
        "Terminal=false\nCategories=Development;\n"
        "StartupWMClass=bindnet-dev\nStartupNotify=true\n"
    )
    application.chmod(0o755)
    for tool, arguments in (
        ("update-desktop-database", [str(application.parent)]),
        ("gtk-update-icon-cache", ["-f", "-t", str(data / "icons/hicolor")]),
    ):
        if shutil.which(tool):
            subprocess.run([tool, *arguments], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    return launcher, application


def main():
    try:
        import tkinter  # noqa: F401

        check_tools()
        launcher, application = install()
        print(f"[install-dev] Launcher instalado: {launcher}")
        print(f"[install-dev] Atalho no menu: {application}")
        print("[install-dev] Abra ‘Bindnet (desenvolvimento)’ ou execute make start-dev.")
    except (ImportError, OSError, RuntimeError) as error:
        print(f"[install-dev] Erro: {error}", file=sys.stderr)
        print("[install-dev] A tela de abertura requer Python 3 com tkinter (python3-tk).", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
