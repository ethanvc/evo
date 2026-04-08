#!/usr/bin/env python3
"""
Copy Go's encoding/json/v2 (and dependencies) into the project,
rewriting import paths and removing goexperiment build tags.

Usage:
    python3 copy_jsonv2.py --go-src /path/to/go/src
"""

import argparse
import os
import shutil
import sys

MODULE = "github.com/ethanvc/evo"
GOSTD_PREFIX = f"{MODULE}/logjson/internal/gostd"

# Directories to copy (relative to encoding/json/).
# "." means the root encoding/json/ directory itself.
COPY_DIRS = [
    (".", "."),
    ("v2", "v2"),
    ("jsontext", "jsontext"),
    ("internal", "internal"),
    ("internal/jsonflags", "internal/jsonflags"),
    ("internal/jsonopts", "internal/jsonopts"),
    ("internal/jsonwire", "internal/jsonwire"),
    ("internal/jsontest", "internal/jsontest"),
]

# Import path replacements (longest first to avoid partial matches).
# The last entry handles self-referencing imports in root-level test files.
IMPORT_REPLACEMENTS = [
    ("encoding/json/internal/jsonflags", f"{GOSTD_PREFIX}/encoding/json/internal/jsonflags"),
    ("encoding/json/internal/jsonopts", f"{GOSTD_PREFIX}/encoding/json/internal/jsonopts"),
    ("encoding/json/internal/jsonwire", f"{GOSTD_PREFIX}/encoding/json/internal/jsonwire"),
    ("encoding/json/internal/jsontest", f"{GOSTD_PREFIX}/encoding/json/internal/jsontest"),
    ("encoding/json/internal", f"{GOSTD_PREFIX}/encoding/json/internal"),
    ("encoding/json/jsontext", f"{GOSTD_PREFIX}/encoding/json/jsontext"),
    ("encoding/json/v2", f"{GOSTD_PREFIX}/encoding/json/v2"),
    ("encoding/json", f"{GOSTD_PREFIX}/encoding/json"),
]

ZSTD_OLD_IMPORT = '"internal/zstd"'
ZSTD_NEW_IMPORT = '"github.com/klauspost/compress/zstd"'

BUILD_TAG_V2 = "//go:build goexperiment.jsonv2"
BUILD_TAG_NOT_V2 = "//go:build !goexperiment.jsonv2"


# ---------------------------------------------------------------------------
# Go source transformations
# ---------------------------------------------------------------------------

def should_skip_file(content: str) -> bool:
    """Skip v1-only files that have //go:build !goexperiment.jsonv2."""
    for line in content.split("\n")[:15]:
        if line.strip() == BUILD_TAG_NOT_V2:
            return True
    return False


def rewrite_imports(content: str) -> str:
    for old, new in IMPORT_REPLACEMENTS:
        content = content.replace(f'"{old}"', f'"{new}"')
    return content


def remove_build_tag(content: str) -> str:
    lines = content.split("\n")
    out = []
    skip_next_blank = False
    for line in lines:
        if line.strip() == BUILD_TAG_V2:
            skip_next_blank = True
            continue
        if skip_next_blank and line.strip() == "":
            skip_next_blank = False
            continue
        skip_next_blank = False
        out.append(line)
    return "\n".join(out)


def fix_zstd_usage(content: str) -> str:
    """
    Replace internal/zstd with klauspost/compress/zstd and adapt the call site.
    internal/zstd.NewReader returns io.Reader; klauspost returns (*Decoder, error).
    """
    content = content.replace(ZSTD_OLD_IMPORT, ZSTD_NEW_IMPORT)
    old_call = (
        "zr := zstd.NewReader(bytes.NewReader(b))\n"
        "\t\t\treturn mustGet(io.ReadAll(zr))"
    )
    new_call = (
        "zr, err := zstd.NewReader(bytes.NewReader(b))\n"
        "\t\t\tif err != nil { panic(err) }\n"
        "\t\t\tdefer zr.Close()\n"
        "\t\t\treturn mustGet(io.ReadAll(zr))"
    )
    content = content.replace(old_call, new_call)
    return content


def process_go_file(src_path: str, dst_path: str, is_jsontest_testdata: bool) -> bool:
    """Read, transform, and write a .go file. Returns False if skipped."""
    with open(src_path, "r", encoding="utf-8") as f:
        content = f.read()

    if should_skip_file(content):
        return False

    content = remove_build_tag(content)
    content = rewrite_imports(content)
    if is_jsontest_testdata:
        content = fix_zstd_usage(content)

    os.makedirs(os.path.dirname(dst_path), exist_ok=True)
    with open(dst_path, "w", encoding="utf-8") as f:
        f.write(content)
    return True


# ---------------------------------------------------------------------------
# Directory copy logic
# ---------------------------------------------------------------------------

def copy_directory(src_dir: str, dst_dir: str, recurse_subdirs: bool, jsontest_dir: str):
    copied, skipped = 0, 0
    for entry in sorted(os.listdir(src_dir)):
        src_path = os.path.join(src_dir, entry)

        if os.path.isdir(src_path):
            if entry == "testdata":
                dst_td = os.path.join(dst_dir, entry)
                shutil.copytree(src_path, dst_td, dirs_exist_ok=True)
                print(f"  copied testdata/ -> {dst_td}")
                copied += 1
            elif recurse_subdirs:
                c, s = copy_directory(
                    src_path, os.path.join(dst_dir, entry),
                    recurse_subdirs=True, jsontest_dir=jsontest_dir,
                )
                copied += c
                skipped += s
            continue

        if not os.path.isfile(src_path):
            continue

        dst_path = os.path.join(dst_dir, entry)
        if entry.endswith(".go"):
            is_jsontest_testdata = (
                os.path.normpath(src_dir) == os.path.normpath(jsontest_dir)
                and entry == "testdata.go"
            )
            if process_go_file(src_path, dst_path, is_jsontest_testdata):
                copied += 1
            else:
                skipped += 1
        else:
            os.makedirs(os.path.dirname(dst_path), exist_ok=True)
            shutil.copy2(src_path, dst_path)
            copied += 1

    return copied, skipped


# ---------------------------------------------------------------------------
# Orchestration
# ---------------------------------------------------------------------------

def parse_args():
    parser = argparse.ArgumentParser(
        description="Copy Go's encoding/json/v2 into the project with import rewriting.",
    )
    parser.add_argument(
        "--go-src", required=True,
        help="Path to Go source root (e.g. /path/to/go1.26.1/src)",
    )
    return parser.parse_args()


def resolve_paths(go_src: str):
    """Return (json_src, output_base, jsontest_dir)."""
    json_src = os.path.join(os.path.abspath(go_src), "encoding", "json")
    if not os.path.isdir(json_src):
        print(f"ERROR: {json_src} does not exist", file=sys.stderr)
        sys.exit(1)

    script_dir = os.path.dirname(os.path.abspath(__file__))
    output_base = os.path.join(script_dir, "encoding", "json")
    jsontest_dir = os.path.join(json_src, "internal", "jsontest")
    return json_src, output_base, jsontest_dir


def clean_output(output_base: str):
    if os.path.exists(output_base):
        print(f"Cleaning existing output directory: {output_base}")
        shutil.rmtree(output_base)


def copy_all(json_src: str, output_base: str, jsontest_dir: str):
    total_copied, total_skipped = 0, 0
    for src_rel, dst_rel in COPY_DIRS:
        if src_rel == ".":
            src_dir = json_src
        else:
            src_dir = os.path.join(json_src, src_rel)
        if dst_rel == ".":
            dst_dir = output_base
        else:
            dst_dir = os.path.join(output_base, dst_rel)

        if not os.path.isdir(src_dir):
            print(f"WARNING: {src_dir} not found, skipping", file=sys.stderr)
            continue

        # "." and "internal" don't recurse (subdirs listed separately).
        recurse = src_rel not in (".", "internal")
        label = "encoding/json/" if src_rel == "." else f"{src_rel}/"
        print(f"Copying {label}")
        c, s = copy_directory(src_dir, dst_dir, recurse_subdirs=recurse, jsontest_dir=jsontest_dir)
        total_copied += c
        total_skipped += s

    return total_copied, total_skipped


def main():
    args = parse_args()
    json_src, output_base, jsontest_dir = resolve_paths(args.go_src)

    print(f"Source: {json_src}")
    print(f"Output: {output_base}\n")

    clean_output(output_base)
    copied, skipped = copy_all(json_src, output_base, jsontest_dir)

    print(f"\nDone. Copied {copied} files, skipped {skipped} v1-only files.")
    print("\nNext steps:")
    print("  cd golangproj && go mod tidy")
    print("  cd golangproj && go build ./logjson/...")
    print("  cd golangproj && go test ./logjson/internal/gostd/encoding/json/...")


if __name__ == "__main__":
    main()
