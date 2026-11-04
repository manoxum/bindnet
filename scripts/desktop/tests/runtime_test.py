"""Contratos do Compose e do arranque, sempre com subprocessos simulados."""

from io import BytesIO
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import MagicMock, patch
from urllib.error import HTTPError, URLError

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from launcher import acquire_lock
from runtime import ADMIN_URL, Runtime


class RuntimeTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        (self.root / "promote.yml").write_text("current_stage: main\n")
        (self.root / ".env.main").touch()
        self.runtime = Runtime(self.root, report=lambda message: None)

    def config(self):
        return {"services": {
            "backend": {"environment": {key: "configured" for key in (
                "ADMIN_USERNAME", "ADMIN_PASSWORD", "POSTGRES_PASSWORD", "MONGO_PASSWORD",
            )}},
            "minio": {"environment": {"MINIO_ROOT_USER": "configured", "MINIO_ROOT_PASSWORD": "configured"}},
            "redis": {"environment": {"REDIS_PASSWORD": "configured"}},
            "frontend": {"environment": {"FRONTEND_PORT": "9191"}},
        }}

    def validate(self, config=None):
        with patch("runtime.check_tools"), patch.object(
            self.runtime, "run", return_value=json.dumps(config or self.config()),
        ) as run:
            self.runtime.validate()
            return run

    def test_validation_is_read_only_and_uses_existing_project_and_env(self):
        run = self.validate()
        self.assertIn("bindnet", self.runtime.command)
        self.assertIn(".env.main", self.runtime.command)
        self.assertEqual(self.runtime.environment["BINDNET_ENV_FILE"], ".env.main")
        self.assertEqual(self.runtime.frontend_port, "9191")
        self.assertNotIn("up", str(run.call_args_list))
        self.assertNotIn("build", [str(call.args[0][-1]) for call in run.call_args_list])
        self.assertEqual((self.root / ".env.main").read_text(), "")

    def test_missing_env_fails_and_legacy_env_is_supported(self):
        (self.root / ".env.main").unlink()
        with self.assertRaisesRegex(RuntimeError, "Falta o ambiente"):
            self.validate()
        (self.root / ".env").touch()
        self.validate()
        self.assertEqual(self.runtime.environment["BINDNET_ENV_FILE"], ".env")

    def test_missing_database_password_is_rejected(self):
        config = self.config()
        config["services"]["backend"]["environment"]["POSTGRES_PASSWORD"] = ""
        with self.assertRaisesRegex(RuntimeError, "Configure POSTGRES_PASSWORD"):
            self.validate(config)

    def test_existing_passwords_are_preserved_without_a_new_strength_policy(self):
        config = self.config()
        config["services"]["backend"]["environment"]["POSTGRES_PASSWORD"] = "change-existing"
        self.validate(config)

    def test_persisted_admin_can_be_reused_with_both_bootstrap_values_empty(self):
        config = self.config()
        config["services"]["backend"]["environment"].update(ADMIN_USERNAME="", ADMIN_PASSWORD="")
        self.validate(config)
        config["services"]["backend"]["environment"]["ADMIN_USERNAME"] = "admin"
        with self.assertRaisesRegex(RuntimeError, "em conjunto"):
            self.validate(config)

    def test_inherited_image_tags_cannot_diverge_from_worker_environment(self):
        with patch.dict("os.environ", {"VERSION": "different", "STAGE": "other"}):
            runtime = Runtime(self.root)
        self.assertNotIn("VERSION", runtime.environment)
        self.assertNotIn("STAGE", runtime.environment)

    def test_daemon_failure_is_actionable(self):
        with patch("runtime.check_tools"), patch.object(self.runtime, "run", side_effect=RuntimeError):
            with self.assertRaisesRegex(RuntimeError, "docker info sem sudo"):
                self.runtime.validate()

    def test_failed_build_never_starts_or_touches_networks(self):
        with patch.object(self.runtime, "compose", side_effect=RuntimeError("build")) as compose:
            with patch("runtime.subprocess.run") as run, self.assertRaises(RuntimeError):
                self.runtime.start()
            compose.assert_called_once_with("build")
            run.assert_not_called()

    def test_startup_creates_only_missing_networks_and_never_removes_volumes(self):
        with patch.object(self.runtime, "compose") as compose, patch.object(self.runtime, "run") as run:
            with patch("runtime.subprocess.run", side_effect=[
                subprocess.CompletedProcess([], 0), subprocess.CompletedProcess([], 1),
            ]), patch.object(self.runtime, "wait_ready") as ready:
                self.runtime.start()
            self.assertEqual([call.args for call in compose.call_args_list], [
                ("build",), ("up", "--detach", "--no-build"),
            ])
            run.assert_called_once_with(["docker", "network", "create", "dbnet"])
            ready.assert_called_once()

    def test_failure_does_not_log_captured_configuration_secrets(self):
        with patch("runtime.subprocess.run", return_value=subprocess.CompletedProcess([], 1, "secret", "secret")):
            with self.assertRaises(RuntimeError) as error:
                self.runtime.run(["docker", "compose", "config"], capture=True)
            self.assertNotIn("secret", str(error.exception))

    def test_duplicate_launch_is_blocked(self):
        with acquire_lock(self.root / "launcher.lock"):
            with self.assertRaisesRegex(RuntimeError, "arranque.*em curso"):
                acquire_lock(self.root / "launcher.lock")

    def test_browser_uses_exact_admin_domain(self):
        with patch.object(self.runtime, "run") as run:
            self.runtime.open_browser()
            run.assert_called_once_with(["xdg-open", ADMIN_URL])

    def test_readiness_requires_page_and_live_api(self):
        page = MagicMock()
        page.__enter__.return_value = page
        page.geturl.return_value = ADMIN_URL + "/"
        page.read.return_value = b"<title>Bindnet</title>"
        unauthorized = HTTPError(ADMIN_URL + "/api/auth/me", 401, "", {}, BytesIO(b"nao autenticado\n"))
        with patch("runtime.build_opener") as opener:
            opener.return_value.open.side_effect = [page, unauthorized]
            self.runtime.wait_ready()
            self.assertEqual(opener.return_value.open.call_count, 2)

    def test_dns_or_tls_failure_has_bounded_wait_and_configuration_guidance(self):
        with patch("runtime.build_opener") as opener, patch("runtime.time.sleep"):
            opener.return_value.open.side_effect = URLError("certificado ou DNS inválido")
            with patch("runtime.time.monotonic", side_effect=[0, 1, 200]):
                with self.assertRaisesRegex(RuntimeError, "Confirme DNS local.*confiança na CA"):
                    self.runtime.wait_ready()


if __name__ == "__main__":
    unittest.main()
