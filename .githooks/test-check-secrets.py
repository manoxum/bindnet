#!/usr/bin/env python3
import pathlib
import shutil
import subprocess
import tempfile
import unittest

class SecretCommitTest(unittest.TestCase):
    def test_blocks_index_secret_without_printing_value(self):
        source = pathlib.Path(__file__).resolve().parent
        with tempfile.TemporaryDirectory() as directory:
            root = pathlib.Path(directory)
            def git(*args):
                return subprocess.run(["git", *args], cwd=root, capture_output=True)
            self.assertEqual(git("init", "-q").returncode, 0)
            git("config", "user.name", "Test")
            git("config", "user.email", "test@example.invalid")
            shutil.copytree(source, root / ".githooks", ignore=shutil.ignore_patterns("__pycache__"))
            git("config", "core.hooksPath", ".githooks")
            git("update-index", "--add", "--cacheinfo", "160000," + "a" * 40 + ",external")
            for prefix in ["bnet_", "kxf_"]:
                secret = prefix + "A" * 43
                (root / "settings.txt").write_text(secret)
                git("add", "settings.txt")
                (root / "settings.txt").write_text("clean working copy")
                result = git("commit", "-m", "test")
                self.assertNotEqual(result.returncode, 0)
                self.assertNotIn(secret.encode(), result.stdout + result.stderr)
                self.assertIn(b"settings.txt", result.stderr)
            git("add", "settings.txt")
            self.assertEqual(git("commit", "-m", "safe").returncode, 0)

if __name__ == "__main__":
    unittest.main()
