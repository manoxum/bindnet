"""Tela nativa de progresso; todas as alterações de UI ficam na thread principal."""

from pathlib import Path
import queue
import tkinter as tk
from tkinter import messagebox, ttk


BG = "#102b25"
FG = "#f4f8f5"
ACCENT = "#b8f24a"


class Splash(tk.Tk):
    def __init__(self, root, log_path, events, start):
        super().__init__(className="bindnet-dev")
        self.title("Bindnet · Desenvolvimento")
        self.configure(bg=BG)
        x = max(0, (self.winfo_screenwidth() - 620) // 2)
        y = max(0, (self.winfo_screenheight() - 400) // 2)
        self.geometry(f"620x400+{x}+{y}")
        self.minsize(620, 400)
        style = ttk.Style(self)
        style.theme_use("clam")
        style.configure("Bindnet.Horizontal.TProgressbar", background=ACCENT,
                        troughcolor="#24483d", borderwidth=0, lightcolor=ACCENT,
                        darkcolor=ACCENT)
        self.events = events
        self.start = start
        self.busy = True
        self.failed = False
        self.expanded = False
        self.log_path = Path(log_path)
        self.reader = self.log_path.open(encoding="utf-8", errors="replace")
        self.protocol("WM_DELETE_WINDOW", self.close)
        self.create_header(root)
        self.status = tk.Label(self, text="A verificar o ambiente…", bg=BG, fg=FG,
                               font=("Sans", 12), wraplength=540, justify="left")
        self.status.pack(anchor="w", padx=32, pady=(16, 8))
        self.progress = ttk.Progressbar(self, mode="indeterminate",
                                       style="Bindnet.Horizontal.TProgressbar")
        self.progress.pack(fill="x", padx=32)
        self.progress.start(12)
        self.notice = tk.Label(
            self, text="Aguarde a validação de Docker e da configuração.",
            bg=BG, fg="#b4c9bf", justify="left", wraplength=550, font=("Sans", 10),
        )
        self.notice.pack(anchor="w", padx=32, pady=12)
        actions = tk.Frame(self, bg=BG)
        actions.pack(fill="x", padx=32, pady=(0, 12))
        self.action = tk.Button(actions, text="Iniciar Bindnet", command=self.confirm,
                                state="disabled", bg=ACCENT, fg=BG, padx=16, pady=7)
        self.action.pack(side="right")
        tk.Button(actions, text="Ver registos", command=self.toggle_logs,
                  bg=BG, fg=FG, padx=10, pady=7).pack(side="left")
        self.details = tk.Frame(self, bg=BG)
        self.text = tk.Text(self.details, bg="#091c17", fg=FG, wrap="word",
                            height=10, state="disabled", font=("Monospace", 9))
        scrollbar = ttk.Scrollbar(self.details, command=self.text.yview)
        self.text.configure(yscrollcommand=scrollbar.set)
        scrollbar.pack(side="right", fill="y")
        self.text.pack(fill="both", expand=True)
        self.poll_id = self.after(100, self.poll)

    def create_header(self, root):
        header = tk.Frame(self, bg=BG)
        header.pack(fill="x", padx=32, pady=(26, 0))
        mark = tk.Canvas(header, width=76, height=76, bg=BG, highlightthickness=0)
        mark.pack(side="left", padx=(0, 18))
        mark.create_oval(19, 19, 57, 57, fill=FG, outline="")
        mark.create_line(38, 7, 38, 69, fill=FG, width=10)
        mark.create_line(7, 38, 69, 38, fill=FG, width=10)
        for x, y in ((38, 7), (69, 38), (38, 38)):
            mark.create_oval(x - 6, y - 6, x + 6, y + 6, fill=ACCENT, outline="")
        labels = tk.Frame(header, bg=BG)
        labels.pack(side="left")
        tk.Label(labels, text="Bindnet", bg=BG, fg=FG,
                 font=("Sans", 27, "bold")).pack(anchor="w")
        version = (Path(root) / "VERSION").read_text().strip()
        tk.Label(labels, text=f"{version}  /  Ambiente de desenvolvimento", bg=BG,
                 fg="#b4c9bf", font=("Sans", 10)).pack(anchor="w")

    def confirm(self):
        self.busy = True
        self.action.configure(state="disabled")
        self.notice.configure(text="Os serviços permanecerão ativos depois de fechar esta janela.")
        self.progress.start(12)
        self.start()

    def toggle_logs(self):
        self.expanded = not self.expanded
        if self.expanded:
            self.geometry("620x640")
            self.details.pack(fill="both", expand=True, padx=32, pady=(0, 20))
        else:
            self.details.pack_forget()
            self.geometry("620x400")

    def poll(self):
        self.after_cancel(self.poll_id)
        chunk = self.reader.read(16384)
        if chunk:
            follow = self.text.yview()[1] >= 0.99
            self.text.configure(state="normal")
            self.text.insert("end", chunk)
            self.text.configure(state="disabled")
            if follow:
                self.text.see("end")
        while True:
            try:
                kind, message = self.events.get_nowait()
            except queue.Empty:
                break
            self.handle_event(kind, message)
            if kind == "done":
                return
        self.poll_id = self.after(100, self.poll)

    def handle_event(self, kind, message):
        self.status.configure(text=message)
        if kind == "validated":
            self.busy = False
            self.progress.stop()
            self.action.configure(state="normal")
            self.notice.configure(text=(
                "Iniciar compila e sobe o stack Docker. Pode interromper DNS ou Wi-Fi "
                "em uso e reativar o hotspot conforme a configuração salva. "
                "Clique em Iniciar Bindnet para autorizar."
            ))
        elif kind == "failed":
            self.busy = False
            self.failed = True
            self.progress.stop()
            self.action.configure(text="Fechar", command=self.close, state="normal")
            self.notice.configure(text=f"Consulte os detalhes abaixo ou {self.log_path}.")
            self.status.configure(text="Não foi possível iniciar o Bindnet.")
            if not self.expanded:
                self.toggle_logs()
        elif kind == "done":
            self.busy = False
            self.close()

    def close(self):
        if self.busy:
            messagebox.showinfo("Bindnet", "Operação em curso. Aguarde a conclusão; pode minimizar a janela.")
            return
        self.reader.close()
        self.progress.stop()
        self.after_cancel(self.poll_id)
        self.destroy()
