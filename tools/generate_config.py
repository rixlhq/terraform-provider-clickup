#!/usr/bin/env python3
"""Discover ClickUp endpoints and build the tfplugingen-openapi inputs.

Replaces the former generate_data_sources.py + merge_update_schemas.py:
  1. Scan the prepared OpenAPI spec for GET endpoints (data sources) and
     create/read/update/delete groups (resources).
  2. Write generator_config.yml (the tfplugingen-openapi config).
  3. Merge each resource's update request body into its create request body
     so the generated Terraform schema includes update-only fields, then
     write the merged spec that tfplugingen-openapi consumes.

Usage: generate_config.py <prepared_spec.json> <output_config.yml> <output_merged.json>
"""

import json
import re
import sys
from copy import deepcopy
from pathlib import Path


# --- shared helpers -----------------------------------------------------------

def normalize_path_params(path: str) -> str:
    """Lower-case path placeholders so they match Terraform attribute names."""
    return re.sub(r"\{(\w+)\}", lambda m: "{" + m.group(1).lower() + "}", path)


def camel_to_snake(name: str) -> str:
    name = re.sub(r"(.)([A-Z][a-z]+)", r"\1_\2", name)
    return re.sub(r"([a-z0-9])([A-Z])", r"\1_\2", name).lower()


def snake_to_camel(s: str) -> str:
    return "".join(part[:1].upper() + part[1:] for part in s.split("_") if part)


# --- endpoint discovery (was generate_data_sources.py) -----------------------

# Data sources whose generated schemas are unusable or intentionally raw JSON.
# They are implemented by hand in internal/provider/datasources_manual.go.
HAND_WRITTEN_DATA_SOURCES = {
    "folder",
    "goals",
    "task",
    "view",
    "view_tasks",
    "task_time_in_status",
    "bulk_time_in_status",
    "time_entries",
}

# OperationId-to-provider-name overrides. The data_source_name() derivation
# from OpenAPI operationId cannot produce clean names for apostrophes,
# camelCase concatenations, and ClickUp's internal numbering (e.g. Teams1).
DATA_SOURCE_NAME_OVERRIDES = {
    "GetTask'sTimeinStatus": "task_time_in_status",
    "GetBulkTasks'TimeinStatus": "bulk_time_in_status",
    "GetWorkspaceplan": "workspace_plan",
    "GetWorkspaceseats": "workspace_seats",
    "GetTeams1": "user_groups",
    "Gettrackedtime": "tracked_time",
    "Getsingulartimeentry": "time_entry",
    "Gettimeentryhistory": "time_entry_history",
    "Getrunningtimeentry": "running_time_entry",
    "Getalltagsfromtimeentries": "time_entry_tags",
}

# Extra resources that build_resources does not discover automatically.
# These have create/read/update/delete (or list-read) and can use the
# genericResource scaffolding.
MANUAL_RESOURCES = {
    "folderless_list": {
        "create": {"path": "/v2/space/{space_id}/list", "method": "post"},
        "read": {"path": "/v2/list/{list_id}", "method": "get"},
        "update": {"path": "/v2/list/{list_id}", "method": "put"},
        "delete": {"path": "/v2/list/{list_id}", "method": "delete"},
        "schema": {},
    },
}


def data_source_name(operation_id: str) -> str:
    if operation_id in DATA_SOURCE_NAME_OVERRIDES:
        return DATA_SOURCE_NAME_OVERRIDES[operation_id]

    # Strip common read prefixes, then snake-case.
    cleaned = re.sub(r"^(Get|get|Search|search)", "", operation_id)
    cleaned = camel_to_snake(cleaned)
    cleaned = re.sub(r"[^a-z0-9_]", "_", cleaned)
    cleaned = re.sub(r"_+", "_", cleaned).strip("_")
    # Remove trailing "public" to keep names short.
    cleaned = re.sub(r"_public$", "", cleaned)
    return cleaned


def resource_base(operation_id: str) -> str:
    # Strip common mutating/reading prefixes to find the entity name.
    return re.sub(r"^(Create|create|Get|get|Update|update|Patch|patch|Delete|delete|Remove|remove)", "", operation_id)


def build_resources(spec: dict) -> dict:
    """Find endpoints with full create/read/update/delete cycles."""
    groups: dict[str, dict[str, dict]] = {}

    for path, item in spec["paths"].items():
        for method, op in item.items():
            if method not in ("get", "post", "put", "patch", "delete"):
                continue
            op_id = op.get("operationId")
            if not op_id:
                continue

            base = resource_base(op_id)
            if not base:
                continue

            groups.setdefault(base, {}).setdefault(method, []).append(
                {
                    "path": path,
                    "method": method,
                    "op": op,
                    "op_id": op_id,
                }
            )

    # Resources that are implemented by hand-written Go files instead of the
    # tfplugingen-framework generator.
    hand_written_resources = {"folder", "user_group", "view"}

    def pick(methods_for_kind):
        # Prefer the shortest path for a given method (avoids sub-action variants).
        return min(methods_for_kind, key=lambda x: len(x["path"]))

    resources = {}
    for base, methods in groups.items():
        # A full Terraform resource needs create, read, update, and delete.
        if "post" not in methods or "get" not in methods:
            continue
        if "patch" not in methods and "put" not in methods:
            continue
        if "delete" not in methods:
            continue

        create = pick(methods["post"])
        read = pick(methods["get"])
        update = pick(methods.get("patch") or methods["put"])
        delete_op = pick(methods["delete"])

        # Derive the name from the selected read operation, not the first GET in
        # the group, and avoid double-stripping read prefixes.
        name = data_source_name(read["op_id"])
        if not name:
            name = camel_to_snake(base).strip("_")

        if name in hand_written_resources:
            continue

        resources[name] = {
            "create": {
                "path": normalize_path_params(create["path"]),
                "method": create["method"],
            },
            "read": {
                "path": normalize_path_params(read["path"]),
                "method": read["method"],
            },
            "update": {
                "path": normalize_path_params(update["path"]),
                "method": update["method"],
            },
            "delete": {
                "path": normalize_path_params(delete_op["path"]),
                "method": delete_op["method"],
            },
            "schema": {},
        }

    return resources


def discover(spec: dict) -> dict:
    """Build the generator_config dict from the prepared spec."""
    config = {"provider": {"name": "clickup"}, "data_sources": {}}

    for path, item in spec["paths"].items():
        for method, op in item.items():
            if method != "get":
                continue
            op_id = op.get("operationId")
            if not op_id:
                continue

            # Skip endpoints with no response schema (e.g. comments/types/subtypes).
            resp = op.get("responses", {}).get("200", {}).get("content", {}).get("application/json", {})
            if not resp.get("schema"):
                continue

            all_params = op.get("parameters", [])
            path_names = {p["name"].lower() for p in all_params if p.get("in") == "path"}
            query_names = {p["name"].lower() for p in all_params if p.get("in") == "query"}
            if path_names & query_names:
                # Path and query params normalize to the same Terraform attribute; skip to avoid schema conflicts.
                continue

            name = data_source_name(op_id)
            if name in HAND_WRITTEN_DATA_SOURCES:
                continue

            path_param_names = {par["name"] for par in all_params if par.get("in") == "path"}

            query_ignores = []
            for par in all_params:
                if par.get("in") != "query":
                    continue
                # Ignore query params that share a name with a path param to avoid duplicate attributes.
                if par["name"] in path_param_names or par["name"] in ("next_cursor", "channel_types"):
                    query_ignores.append(par["name"])

            schema = {}
            if query_ignores:
                schema["ignores"] = query_ignores

            config["data_sources"][name] = {
                "read": {"path": normalize_path_params(path), "method": "get"},
                "schema": schema if schema else {},
            }

    config["resources"] = build_resources(spec)
    for name, res in MANUAL_RESOURCES.items():
        if name not in config["resources"]:
            config["resources"][name] = res

    return config


# --- update-schema merge (was merge_update_schemas.py) -----------------------

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


# --- entrypoint ---------------------------------------------------------------

def main() -> None:
    if len(sys.argv) < 4:
        print(
            f"usage: {sys.argv[0]} <prepared_spec.json> <output_config.yml> <output_merged.json>",
            file=sys.stderr,
        )
        sys.exit(1)

    spec_path = Path(sys.argv[1])
    config_path = Path(sys.argv[2])
    merged_path = Path(sys.argv[3])

    spec = json.loads(spec_path.read_text())
    config = discover(spec)

    # Write as JSON because the YAML parser accepts JSON and PyYAML may not be installed.
    config_path.write_text(json.dumps(config, indent=2))
    print(f"wrote {config_path}")

    # Merge update request bodies into create request bodies using the freshly
    # discovered config, then write the merged spec for tfplugingen-openapi.
    for resource in config.get("resources", {}).values():
        merge_update_into_create(spec, resource)

    merged_path.write_text(json.dumps(spec, indent=2))
    print(f"wrote {merged_path}")


if __name__ == "__main__":
    main()
