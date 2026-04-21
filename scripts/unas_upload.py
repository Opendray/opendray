#!/usr/bin/env python3
"""UNAS SMB uploader — a smbclient-free replacement for deploy_release.sh.

Uses smbprotocol (pure Python, no native libsmbclient needed). We picked
this path because the syz LXC runs inside a sandbox that forbids
sudo-installing samba-client; smbprotocol installs via `pip install
--user smbprotocol` with no compile step.

Subcommands (all exit 0/non-zero cleanly for shell integration):

  ping   <server> <share> <user>
    Auth-check only. Prints "ok" or an error. Exit 0 / 1.

  mkdir  <server> <share> <user> <remote_dir>
    Creates the remote directory tree idempotently. Safe to call on
    existing dirs — each ensure_directory call no-ops when the dir is
    already present.

  put    <server> <share> <user> <remote_dir> <local_file>
    Uploads local_file into remote_dir, keeping the basename. Prints
    the remote size in bytes to stdout on success so the caller can
    verify against the local size.

  size   <server> <share> <user> <remote_dir> <filename>
    Prints the size of <remote_dir>/<filename> in bytes (for
    post-upload verification). Exit 0 if found, 1 if not.

Password is read from stdin — NEVER accept it as an argv argument,
that leaks into `ps` output and shell history.

Usage from deploy_release.sh:
    UNAS_PASSWORD=$(cat ~/.config/opendray/unas.pw)
    echo "$UNAS_PASSWORD" | python3 scripts/unas_upload.py ping \\
        192.168.9.8 Claude_Workspace linivek
"""

from __future__ import annotations

import os
import sys
from contextlib import contextmanager
from pathlib import Path
from uuid import uuid4

from smbprotocol.connection import Connection
from smbprotocol.exceptions import SMBException
from smbprotocol.open import (
    CreateDisposition,
    CreateOptions,
    FileAttributes,
    FilePipePrinterAccessMask,
    ImpersonationLevel,
    Open,
    ShareAccess,
)
from smbprotocol.session import Session
from smbprotocol.tree import TreeConnect


CHUNK = 64 * 1024  # 64 KiB per SMB write — well under the default max


def _read_password() -> str:
    pw = sys.stdin.read().rstrip("\n\r")
    if not pw:
        print("empty password on stdin", file=sys.stderr)
        sys.exit(1)
    return pw


@contextmanager
def _tree(server: str, share: str, user: str, password: str):
    """Context manager that yields a connected TreeConnect and tears
    down every layer cleanly even on error."""
    conn = Connection(uuid4(), server, 445)
    conn.connect(timeout=10)
    try:
        sess = Session(conn, user, password)
        sess.connect()
        try:
            tree = TreeConnect(sess, rf"\\{server}\{share}")
            tree.connect()
            try:
                yield tree
            finally:
                tree.disconnect()
        finally:
            sess.disconnect()
    finally:
        conn.disconnect()


def _ensure_dir(tree: TreeConnect, remote_dir: str) -> None:
    """Create `remote_dir` one segment at a time; existing segments are
    harmless no-ops."""
    parts = [p for p in remote_dir.replace("\\", "/").split("/") if p]
    running = ""
    for part in parts:
        running = f"{running}/{part}" if running else part
        dir_open = Open(tree, running)
        try:
            dir_open.create(
                ImpersonationLevel.Impersonation,
                FilePipePrinterAccessMask.FILE_READ_ATTRIBUTES,
                FileAttributes.FILE_ATTRIBUTE_DIRECTORY,
                ShareAccess.FILE_SHARE_READ | ShareAccess.FILE_SHARE_WRITE,
                CreateDisposition.FILE_OPEN_IF,
                CreateOptions.FILE_DIRECTORY_FILE,
            )
            dir_open.close()
        except SMBException as e:
            # File-already-exists as a non-dir, or perms — surface.
            raise RuntimeError(f"mkdir {running}: {e}") from e


def cmd_ping(server: str, share: str, user: str) -> int:
    password = _read_password()
    try:
        with _tree(server, share, user, password):
            print("ok")
        return 0
    except Exception as e:
        print(f"ping failed: {e}", file=sys.stderr)
        return 1


def cmd_mkdir(server: str, share: str, user: str, remote_dir: str) -> int:
    password = _read_password()
    try:
        with _tree(server, share, user, password) as tree:
            _ensure_dir(tree, remote_dir)
        print(f"mkdir {remote_dir}: ok")
        return 0
    except Exception as e:
        print(f"mkdir failed: {e}", file=sys.stderr)
        return 1


def cmd_put(
    server: str, share: str, user: str, remote_dir: str, local_file: str
) -> int:
    password = _read_password()
    local = Path(local_file)
    if not local.is_file():
        print(f"local file not found: {local_file}", file=sys.stderr)
        return 1

    remote_path = f"{remote_dir.rstrip('/')}/{local.name}".lstrip("/")
    size_local = local.stat().st_size

    try:
        with _tree(server, share, user, password) as tree:
            _ensure_dir(tree, remote_dir)
            f = Open(tree, remote_path)
            f.create(
                ImpersonationLevel.Impersonation,
                FilePipePrinterAccessMask.FILE_WRITE_DATA
                | FilePipePrinterAccessMask.FILE_READ_ATTRIBUTES,
                FileAttributes.FILE_ATTRIBUTE_NORMAL,
                ShareAccess.FILE_SHARE_READ,
                CreateDisposition.FILE_OVERWRITE_IF,
                CreateOptions.FILE_NON_DIRECTORY_FILE,
            )
            try:
                offset = 0
                with local.open("rb") as fh:
                    while True:
                        buf = fh.read(CHUNK)
                        if not buf:
                            break
                        f.write(buf, offset)
                        offset += len(buf)
            finally:
                f.close()

        # Verify by re-opening + stating the remote file.
        with _tree(server, share, user, password) as tree:
            f = Open(tree, remote_path)
            f.create(
                ImpersonationLevel.Impersonation,
                FilePipePrinterAccessMask.FILE_READ_ATTRIBUTES,
                FileAttributes.FILE_ATTRIBUTE_NORMAL,
                ShareAccess.FILE_SHARE_READ,
                CreateDisposition.FILE_OPEN,
                CreateOptions.FILE_NON_DIRECTORY_FILE,
            )
            try:
                remote_size = f.end_of_file
            finally:
                f.close()

        if remote_size != size_local:
            print(
                f"size mismatch: local={size_local} remote={remote_size}",
                file=sys.stderr,
            )
            return 1
        # Machine-readable on stdout, one line, just the bytes count.
        print(remote_size)
        return 0
    except Exception as e:
        print(f"put failed: {e}", file=sys.stderr)
        return 1


def cmd_size(
    server: str, share: str, user: str, remote_dir: str, filename: str
) -> int:
    password = _read_password()
    remote_path = f"{remote_dir.rstrip('/')}/{filename}".lstrip("/")
    try:
        with _tree(server, share, user, password) as tree:
            f = Open(tree, remote_path)
            f.create(
                ImpersonationLevel.Impersonation,
                FilePipePrinterAccessMask.FILE_READ_ATTRIBUTES,
                FileAttributes.FILE_ATTRIBUTE_NORMAL,
                ShareAccess.FILE_SHARE_READ,
                CreateDisposition.FILE_OPEN,
                CreateOptions.FILE_NON_DIRECTORY_FILE,
            )
            try:
                print(f.end_of_file)
            finally:
                f.close()
        return 0
    except Exception as e:
        print(f"size failed: {e}", file=sys.stderr)
        return 1


def main(argv: list[str]) -> int:
    if len(argv) < 2:
        print(__doc__, file=sys.stderr)
        return 2
    cmd = argv[1]
    try:
        if cmd == "ping" and len(argv) == 5:
            return cmd_ping(argv[2], argv[3], argv[4])
        if cmd == "mkdir" and len(argv) == 6:
            return cmd_mkdir(argv[2], argv[3], argv[4], argv[5])
        if cmd == "put" and len(argv) == 7:
            return cmd_put(argv[2], argv[3], argv[4], argv[5], argv[6])
        if cmd == "size" and len(argv) == 7:
            return cmd_size(argv[2], argv[3], argv[4], argv[5], argv[6])
    except KeyboardInterrupt:
        return 130
    print(f"bad usage: {' '.join(argv)}", file=sys.stderr)
    print(__doc__, file=sys.stderr)
    return 2


if __name__ == "__main__":
    sys.exit(main(sys.argv))
