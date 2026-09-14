"""Regression coverage for the actual release artifact deployment gate."""

import runpy
import unittest
from pathlib import Path

check_target = runpy.run_path(str(Path(__file__).with_name("check-macos-target")))["check_target"]


def load_command(version="13.0", platform="MACOS"):
    return f"Load command 1\n      cmd LC_BUILD_VERSION\n platform {platform}\n    minos {version}\n"


class MacOSTargetTest(unittest.TestCase):
    def test_accepts_macos_13(self):
        self.assertEqual(check_target(load_command()), ["13.0"])
        self.assertEqual(check_target(load_command("13.0.0", "1")), ["13.0.0"])

    def test_rejects_mismatched_or_missing_metadata(self):
        for commands in ("", load_command("14.0"), load_command("12.0"),
                         load_command(platform="IOS"), load_command("invalid"),
                         load_command() + load_command("15.0")):
            with self.subTest(commands=commands), self.assertRaises(ValueError):
                check_target(commands)


if __name__ == "__main__":
    unittest.main()
