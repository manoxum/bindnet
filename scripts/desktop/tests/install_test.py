"""Verifica integração Linux num HOME temporário, sem instalar no host."""

import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from install import exec_quote, install


class InstallTest(unittest.TestCase):
    def test_install_is_repeatable_and_handles_shell_characters(self):
        with tempfile.TemporaryDirectory() as directory:
            base = Path(directory)
            root = base / ("repo ' " + chr(34) + chr(92) + " $() ` %")
            home = base / ("home ' " + chr(34) + chr(92) + " $() ` %")
            icon = root / "services/frontend/public/09-nucleo.svg"
            icon.parent.mkdir(parents=True)
            icon.write_text("<svg/>")
            entrypoint = root / "scripts/desktop/launcher.py"
            entrypoint.parent.mkdir(parents=True)
            entrypoint.write_text("from pathlib import Path\nPath('started').touch()\n")
            with patch("install.subprocess.run") as run, patch.dict(
                os.environ, {"XDG_DATA_HOME": str(base / "snap-private")},
            ):
                launcher, desktop = install(root, home)
                first = desktop.read_bytes()
                install(root, home)
                self.assertEqual(first, desktop.read_bytes())
                self.assertFalse((root / "started").exists())
                self.assertFalse((base / "snap-private").exists())
                self.assertNotIn("docker", str(run.call_args_list))
            self.assertTrue(os.access(launcher, os.X_OK))
            subprocess.run(["bash", "-n", str(launcher)], check=True)
            subprocess.run([str(launcher)], check=True)
            self.assertTrue((root / "started").is_file())
            self.assertIn(f"Exec={exec_quote(launcher)}\n", desktop.read_text())
            self.assertIn("Terminal=false", desktop.read_text())
            self.assertIn("StartupWMClass=bindnet-dev", desktop.read_text())
            self.assertEqual((home / ".local/share/icons/hicolor/scalable/apps/bindnet-dev.svg").read_text(), "<svg/>")
            if shutil.which("desktop-file-validate"):
                subprocess.run(["desktop-file-validate", str(desktop)], check=True)

    def test_desktop_exec_escapes_reserved_characters(self):
        self.assertEqual(exec_quote('/tmp/a b%"$`\\c'), '"/tmp/a b%%\\\\"\\\\$\\\\`\\\\\\\\c"')


if __name__ == "__main__":
    unittest.main()
