"""Verifica a confirmação e a apresentação nativa com DISPLAY real ou Xvfb."""

import os
from pathlib import Path
import queue
import sys
import tempfile
import unittest
from unittest.mock import Mock, patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from splash import Splash


@unittest.skipUnless(os.environ.get("DISPLAY"), "Requer DISPLAY; execute com xvfb-run -a make check-launcher.")
class SplashTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        (self.root / "VERSION").write_text("v1.2.3")
        self.log = self.root / "launcher.log"
        self.log.write_text("[bindnet-launcher] A validar…\n")
        self.events = queue.Queue()
        self.start = Mock()
        self.window = Splash(self.root, self.log, self.events, self.start)
        self.window.update()
        self.addCleanup(self.destroy)

    def destroy(self):
        if self.window.winfo_exists():
            self.window.busy = False
            self.window.close()

    def test_start_requires_validation_and_explicit_click(self):
        self.window.action.invoke()
        self.start.assert_not_called()
        self.window.handle_event("validated", "Pronto")
        self.start.assert_not_called()
        self.assertIn("Wi-Fi", self.window.notice.cget("text"))
        self.window.action.invoke()
        self.start.assert_called_once()
        self.assertTrue(self.window.busy)
        self.assertEqual(self.window.action.cget("state"), "disabled")

    def test_failure_keeps_window_and_log_visible(self):
        self.events.put(("failed", "Falha de build"))
        self.window.poll()
        self.window.update()
        self.assertTrue(self.window.failed)
        self.assertFalse(self.window.busy)
        self.assertTrue(self.window.expanded)
        self.assertTrue(self.window.winfo_exists())
        self.assertIn("A validar", self.window.text.get("1.0", "end"))

    def test_close_during_startup_preserves_operation(self):
        with patch("splash.messagebox.showinfo") as message:
            self.window.close()
            message.assert_called_once()
        self.assertTrue(self.window.winfo_exists())

    def test_actions_and_notice_fit_in_compact_window(self):
        self.window.handle_event("validated", "Pronto para iniciar o Bindnet.")
        self.window.update()
        action_bottom = self.window.action.winfo_rooty() + self.window.action.winfo_height()
        self.assertLessEqual(action_bottom, self.window.winfo_rooty() + self.window.winfo_height())
        self.assertGreater(self.window.action.winfo_height(), 20)


if __name__ == "__main__":
    unittest.main()
