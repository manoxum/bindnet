#!/usr/bin/env python3
"""Bloqueia tokens Bindnet/KaxaFirme em blobs preparados para commit."""
import argparse
import re
import subprocess
import sys

PATTERN = re.compile(rb"(?<![A-Za-z0-9_-])(?:bnet_|kxf_)[A-Za-z0-9_-]{43}(?![A-Za-z0-9_-])")


def git(*args):
    return subprocess.check_output(["git", *args])


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--all", action="store_true", help="Verificar todos os ficheiros do indice")
    options = parser.parse_args()
    command = ("ls-files", "-z") if options.all else ("diff", "--cached", "--name-only", "--diff-filter=ACMR", "-z")
    paths = git(*command).split(b"\0")
    modes = {}
    for entry in git("ls-files", "--stage", "-z").split(b"\0"):
        if entry:
            metadata, name = entry.split(b"\t", 1)
            modes[name] = metadata.split()[0]
    found = False
    for path in paths:
        if not path or modes.get(path) == b"160000":
            continue
        name = path.decode("utf-8", "surrogateescape")
        ref = ":" + name
        if git("cat-file", "-t", ref).strip() != b"blob":
            continue
        data = git("cat-file", "blob", ref)
        for match in PATTERN.finditer(data):
            line = data.count(b"\n", 0, match.start()) + 1
            print(f"Token de integracao detetado em {name!r}:{line}. Remova-o e renove o token.", file=sys.stderr)
            found = True
    return 1 if found else 0


if __name__ == "__main__":
    try:
        sys.exit(main())
    except subprocess.CalledProcessError:
        print("Nao foi possivel verificar os ficheiros preparados. Commit bloqueado.", file=sys.stderr)
        sys.exit(2)
