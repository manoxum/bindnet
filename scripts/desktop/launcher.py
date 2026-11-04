"""Launcher do sistema: validação, confirmação de arranque e abertura do navegador."""

import fcntl
import os
from pathlib import Path
import queue
import sys
import threading

from runtime import ROOT, Runtime


def acquire_lock(path):
    lock = Path(path).open("a")
    try:
        fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
    except OSError:
        lock.close()
        raise RuntimeError("Já existe um arranque do Bindnet em curso.") from None
    return lock


def launch():
    from splash import Splash

    os.umask(0o077)
    state = ROOT / ".harness"
    (state / "logs").mkdir(parents=True, exist_ok=True)
    (state / "run").mkdir(parents=True, exist_ok=True)
    with acquire_lock(state / "run/launcher.lock"):
        log_path = state / "logs/launcher.log"
        with log_path.open("w", buffering=1) as output:
            events = queue.Queue()

            def report(message):
                print(f"[bindnet-launcher] {message}", file=output, flush=True)
                events.put(("status", message))

            runtime = Runtime(report=report, output=output)

            def work(action, success):
                try:
                    action()
                    events.put((success, "Pronto para iniciar o Bindnet."))
                except Exception as error:
                    report(f"Erro: {error}")
                    events.put(("failed", str(error)))

            def start():
                def startup():
                    runtime.start()
                    runtime.open_browser()

                threading.Thread(target=work, args=(startup, "done"), daemon=True).start()

            splash = Splash(ROOT, log_path, events, start)
            # O loop gráfico apresenta a tela antes de iniciar qualquer trabalho.
            splash.after(150, lambda: threading.Thread(
                target=work, args=(runtime.validate, "validated"), daemon=True,
            ).start())
            splash.mainloop()
            return 1 if splash.failed else 0


def main():
    try:
        return launch()
    except Exception as error:
        message = f"[bindnet-launcher] {error}"
        print(message, file=sys.stderr)
        # Falhas anteriores à janela também precisam aparecer pelo atalho sem terminal.
        try:
            from tkinter import Tk, messagebox

            window = Tk()
            window.withdraw()
            messagebox.showerror("Bindnet", message, parent=window)
            window.destroy()
        except Exception:
            pass
        return 1


if __name__ == "__main__":
    sys.exit(main())
