#!/usr/bin/env python3
"""Create SecureTar v2 and v3 encrypted test archives for hassio-tar testing."""

import os
import tarfile
import tempfile
from pathlib import Path

from securetar import SecureTarArchive

PASSWORD = "test-password-123"
OUTPUT_DIR = Path(os.environ.get("TEST_OUTPUT_DIR", "/tmp/test-fixtures"))


def create_test_content(tmp_dir: Path) -> Path:
    """Create sample files to encrypt."""
    content_dir = tmp_dir / "sample"
    content_dir.mkdir()
    (content_dir / "hello.txt").write_text("Hello from hassio-tar test!\n")
    (content_dir / "data.bin").write_bytes(os.urandom(4096))
    sub = content_dir / "subdir"
    sub.mkdir()
    (sub / "nested.txt").write_text("Nested file content\n")
    return content_dir


def create_encrypted_archive(
    version: int, content_dir: Path, output_path: Path
) -> None:
    """Create a SecureTar encrypted archive with the given version."""
    with SecureTarArchive(
        output_path,
        "w",
        password=PASSWORD,
        create_version=version,
    ) as archive:
        with archive.create_tar("core.tar.gz", gzip=True) as inner:
            inner.add(str(content_dir), arcname=".")


def create_plain_tar(content_dir: Path, output_path: Path) -> None:
    """Create a plain (unencrypted) tar for passthrough testing."""
    with tarfile.open(output_path, "w") as tf:
        tf.add(str(content_dir), arcname=".")


def main() -> None:
    OUTPUT_DIR.mkdir(parents=True, exist_ok=True)

    with tempfile.TemporaryDirectory() as tmp:
        content_dir = create_test_content(Path(tmp))

        create_encrypted_archive(2, content_dir, OUTPUT_DIR / "backup_v2.tar")
        create_encrypted_archive(3, content_dir, OUTPUT_DIR / "backup_v3.tar")
        create_plain_tar(content_dir, OUTPUT_DIR / "backup_plain.tar")

    (OUTPUT_DIR / "password.txt").write_text(PASSWORD)
    print(f"Test fixtures created in {OUTPUT_DIR}")


if __name__ == "__main__":
    main()
