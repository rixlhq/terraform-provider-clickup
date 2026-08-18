#!/usr/bin/env python3
"""Merge update request body schemas into create request bodies.

The tfplugingen-openapi resource mapper only uses the create and read operation
schemas when building the Terraform resource schema. Update-only fields (such as
"color" or "admin_can_manage") are therefore missing from the generated
resource, which makes Update requests fail because buildBody cannot send them.

This script takes the prepared OpenAPI spec and the generated
generator_config.yml, finds each resource's create and update operations, and
merges the update request body properties into the create request body as
optional properties. The resulting spec is fed to tfplugingen-openapi so the
resource schema contains the update-only attributes, while tools/generate*.py
continue to read the original prepared spec and therefore emit the original
create/update body field lists.
"""

import json
import sys
from copy import deepcopy
from pathlib import Path


def schema_body_properties(schema: dict) -> dict:
    """Return the request body JSON schema for an operation, or {}."""
    return (
        schema.get("requestBody", {})
        .get("content", {})
        .get("application/json", {})
        .get("schema", {})
    )


def is_add_rem_object(schema: dict) -> bool:
    """Detect ClickUp's { add: [...], rem: [...] } update shape."""
    if schema.get("type") != "object":
        return False
    props = schema.get("properties", {})
    return "add" in props and "rem" in props


def merge_props(
    create_props: dict, update_props: dict, create_required: list | None
) -> dict:
    """Deep merge update properties into create properties."""
    if create_required is None:
        create_required = []

    out = deepcopy(create_props)
    for key, update_schema in update_props.items():
        if key not in out:
            # Update-only field: add as optional by making sure it is not in
            # the create required list.
            if key in create_required:
                create_required.remove(key)
            out[key] = deepcopy(update_schema)
            if is_add_rem_object(out[key]):
                out[key].pop("required", None)
            continue

        existing = out[key]
        if not isinstance(existing, dict) or not isinstance(update_schema, dict):
            continue

        existing_type = existing.get("type")
        update_type = update_schema.get("type")
        if existing_type != update_type:
            # ClickUp uses an array on create but an { add, rem } object on
            # update for fields like assignees and group_assignees. Promote the
            # richer update shape into the create schema so the generated
            # Terraform resource can represent both operations.
            if existing_type == "array" and is_add_rem_object(update_schema):
                if key in create_required:
                    create_required.remove(key)
                out[key] = deepcopy(update_schema)
                # Promoting the { add, rem } shape to the create schema lets the
                # generated resource represent both create and update, but the
                # individual add/rem sub-fields should not be required.
                out[key].pop("required", None)
                continue

            # ClickUp sometimes accepts a numeric value on create but a string
            # (including the sentinel "none") on update. Use the update string
            # type for the Terraform schema and transform back to a number at
            # create request time.
            if existing_type in ("integer", "number") and update_type == "string":
                if key in create_required:
                    create_required.remove(key)
                out[key] = deepcopy(update_schema)
                continue

        if existing_type == "object" and isinstance(
            update_schema.get("properties"), dict
        ):
            existing.setdefault("properties", {})
            existing.setdefault("required", [])
            merge_props(
                existing["properties"],
                update_schema["properties"],
                existing.get("required", []),
            )

        if existing_type == "array" and isinstance(update_schema.get("items"), dict):
            existing_items = existing.get("items", {})
            if isinstance(existing_items, dict):
                if existing_items.get("type") == update_schema["items"].get("type") == "object":
                    existing_items.setdefault("properties", {})
                    existing_items.setdefault("required", [])
                    merge_props(
                        existing_items["properties"],
                        update_schema["items"].get("properties", {}),
                        existing_items.get("required", []),
                    )

    return out


def merge_update_into_create(spec: dict, resource: dict) -> None:
    """Merge a single resource's update request body into its create body."""
    create_path = resource["create"]["path"]
    create_method = resource["create"]["method"]
    update_path = resource["update"]["path"]
    update_method = resource["update"]["method"]

    create_op = spec["paths"].get(create_path, {}).get(create_method)
    update_op = spec["paths"].get(update_path, {}).get(update_method)
    if not create_op or not update_op:
        return

    create_body = schema_body_properties(create_op)
    update_body = schema_body_properties(update_op)
    if not create_body or not update_body:
        return

    if "properties" not in create_body and "properties" not in update_body:
        return

    create_body.setdefault("properties", {})
    create_body.setdefault("required", [])
    create_body["properties"] = merge_props(
        create_body["properties"],
        update_body.get("properties", {}),
        create_body.get("required", []),
    )


def main() -> None:
    if len(sys.argv) < 4:
        print(
            f"usage: {sys.argv[0]} <prepared_spec.json> <generator_config.yml> <output.json>",
            file=sys.stderr,
        )
        sys.exit(1)

    spec_path = Path(sys.argv[1])
    config_path = Path(sys.argv[2])
    output_path = Path(sys.argv[3])

    spec = json.loads(spec_path.read_text())
    config = json.loads(config_path.read_text())

    for resource in config.get("resources", {}).values():
        merge_update_into_create(spec, resource)

    output_path.write_text(json.dumps(spec, indent=2))
    print(f"wrote {output_path}")


if __name__ == "__main__":
    main()
