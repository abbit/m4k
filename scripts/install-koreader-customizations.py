#!/usr/bin/env python3
"""
Installs customizations into a remote KOReader instance, using SCP to copy the plugin files.
"""

from pathlib import Path
import subprocess

CUSTOMIZATIONS_DIR = Path(__file__).parent.parent / "koreader-customizations"
DEVICE_KOREADER_DIR = "/mnt/us/koreader"


def main() -> None:
    # copy plugins and settings
    subprocess.check_call(
        [
            "scp",
            "-r",
            f"{CUSTOMIZATIONS_DIR}/plugins",
            f"{CUSTOMIZATIONS_DIR}/settings",
            f"kindle:{DEVICE_KOREADER_DIR}/",
        ]
    )

    print("Customizations successfully installed! Restart KOReader.")


if __name__ == "__main__":
    main()
