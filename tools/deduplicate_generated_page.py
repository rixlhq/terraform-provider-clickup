#!/usr/bin/env python3
"""Rename duplicate generated types in datasource_page so the package compiles."""

import re
from pathlib import Path

FILE = Path("internal/provider/generated/datasource_page/page_data_source_gen.go")


def root_of(name: str) -> str:
    if name.endswith("Type"):
        return name.removesuffix("Type")
    if name.endswith("Value"):
        return name.removesuffix("Value")
    return name


def main() -> None:
    text = FILE.read_text()

    # Find all top-level type declarations in order.
    decls = [(i, m.group(1)) for i, line in enumerate(text.split("\n"), 1) if (m := re.match(r"^type (\w+) struct", line))]

    # The second group for a root starts at the second occurrence of its *Type* declaration.
    root_first = {}
    root_second = {}
    for line_no, name in decls:
        if not name.endswith("Type"):
            continue
        root = root_of(name)
        if root in root_first:
            root_second.setdefault(root, line_no)
        else:
            root_first[root] = line_no

    # Process from the end of the file backwards so earlier line numbers stay stable.
    for root, start_line in sorted(root_second.items(), key=lambda kv: kv[1], reverse=True):
        type_name = root + "Type"
        value_name = root + "Value"

        tokens = [type_name, value_name]
        for suffix in ["", "Null", "Unknown", "Must"]:
            tokens.append(f"New{value_name}{suffix}")
        tokens.sort(key=len, reverse=True)

        lines = text.split("\n")
        start_pos = sum(len(line) + 1 for line in lines[: start_line - 1])

        prefix = text[:start_pos]
        region = text[start_pos:]

        for tok in tokens:
            region = re.sub(rf"\b{re.escape(tok)}\b", tok + "2", region)

        text = prefix + region

    FILE.write_text(text)
    print(f"deduplicated {FILE}")


if __name__ == "__main__":
    main()
