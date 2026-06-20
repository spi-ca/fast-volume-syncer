#!/usr/bin/env python3
"""Check that Go packages, declarations, and meaningful fields have comments."""

from __future__ import annotations

import pathlib
import re
import sys

FUNC_OR_TYPE = re.compile(
    r"func\s+(\([^)]*\)\s*)?[A-Za-z_]"
    r"|type\s+[A-Za-z_]\w*\s+struct\b"
    r"|type\s+[A-Za-z_]\w*\s+interface\b"
)
STRUCT_START = re.compile(r"type\s+\w+\s+struct\b")
INTERFACE_START = re.compile(r"type\s+\w+\s+interface\b")
FIELD_LINE = re.compile(
    r"(\*?[A-Za-z_]\w*(\.[A-Za-z_]\w*)?|[A-Za-z_]\w*(\s*,\s*[A-Za-z_]\w*)*)\s+"
)
INTERFACE_METHOD = re.compile(r"[A-Za-z_]\w*\(")


def previous_comment(lines: list[str], index: int) -> bool:
    """Return true when the nearest previous non-blank line is a line comment."""
    cursor = index - 1
    while cursor >= 0 and lines[cursor].strip() == "":
        cursor -= 1
    return cursor >= 0 and lines[cursor].lstrip().startswith("//")


def check_file(path: pathlib.Path) -> list[tuple[int, str, str]]:
    """Return comment coverage issues for one Go file."""
    lines = path.read_text().splitlines()
    issues: list[tuple[int, str, str]] = []
    in_struct = False
    in_interface = False

    for index, line in enumerate(lines):
        stripped = line.strip()
        line_no = index + 1

        if stripped.startswith("package ") and not previous_comment(lines, index):
            issues.append((line_no, "package", stripped))

        if FUNC_OR_TYPE.match(stripped) and not previous_comment(lines, index):
            issues.append((line_no, "declaration", stripped))

        if STRUCT_START.match(stripped) and "{" in stripped and "}" not in stripped:
            in_struct = True
            continue
        if INTERFACE_START.match(stripped) and "{" in stripped and "}" not in stripped:
            in_interface = True
            continue

        if in_struct:
            if stripped == "}":
                in_struct = False
                continue
            if stripped and not stripped.startswith("//") and FIELD_LINE.match(stripped):
                if not previous_comment(lines, index):
                    issues.append((line_no, "struct field", stripped))

        if in_interface:
            if stripped == "}":
                in_interface = False
                continue
            if stripped and not stripped.startswith("//") and INTERFACE_METHOD.match(stripped):
                if not previous_comment(lines, index):
                    issues.append((line_no, "interface method", stripped))

    return issues


def main() -> int:
    """Check all Go files under the repository root or supplied paths."""
    roots = [pathlib.Path(arg) for arg in sys.argv[1:]] or [pathlib.Path(".")]
    files: list[pathlib.Path] = []
    for root in roots:
        if root.is_file() and root.suffix == ".go":
            files.append(root)
        elif root.is_dir():
            files.extend(root.glob("**/*.go"))

    issues: list[tuple[pathlib.Path, int, str, str]] = []
    for path in sorted(set(files)):
        if ".git" in path.parts:
            continue
        for line_no, kind, text in check_file(path):
            issues.append((path, line_no, kind, text))

    print(f"go files checked: {len(set(files))}")
    print(f"missing comments: {len(issues)}")
    for path, line_no, kind, text in issues:
        print(f"{path}:{line_no}: missing {kind} comment: {text}")
    return 1 if issues else 0


if __name__ == "__main__":
    raise SystemExit(main())
