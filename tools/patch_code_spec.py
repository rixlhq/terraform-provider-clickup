#!/usr/bin/env python3
"""Post-process provider_code_spec.json to fix generator limitations.

The ClickUp OpenAPI spec declares task custom field values as a polymorphic
oneOf/anyOf. The generator cannot represent that, so it falls back to bool and
skips the value. We force the value to a JSON-encoded string and remove the
button-only `value_options` wrapper, then transform the string back to the real
value at request time and stringify the response value when reading state.
"""

import json
import sys
from pathlib import Path


def patch_task_custom_fields(spec: dict) -> None:
    for resource in spec.get("resources", []):
        if resource.get("name") != "task":
            continue
        for attr in resource.get("schema", {}).get("attributes", []):
            if attr.get("name") != "custom_fields":
                continue
            list_nested = attr.get("list_nested", {})
            nested = list_nested.get("nested_object", {})
            attrs = nested.get("attributes", [])
            new_attrs = []
            for a in attrs:
                name = a.get("name")
                if name == "value":
                    new_attrs.append(
                        {
                            "name": "value",
                            "string": {
                                "computed_optional_required": "required",
                                "description": "JSON-encoded custom field value. Use jsonencode() for non-string values.",
                            },
                        }
                    )
                    continue
                if name == "value_options":
                    # Drop the button-only value_options wrapper; it is not useful
                    # through the generic resource and complicates the schema.
                    continue
                new_attrs.append(a)
            nested["attributes"] = new_attrs


def main() -> None:
    if len(sys.argv) < 2:
        print(f"usage: {sys.argv[0]} <provider_code_spec.json>", file=sys.stderr)
        sys.exit(1)
    path = Path(sys.argv[1])
    spec = json.loads(path.read_text())
    patch_task_custom_fields(spec)
    path.write_text(json.dumps(spec, indent=2))
    print(f"patched {path}")


if __name__ == "__main__":
    main()
