#!/usr/bin/env python3
"""Generate generator_config.yml and Go wrappers for ClickUp GET/CRUD endpoints."""

import json
import re
from pathlib import Path

SPEC_PATH = Path("ClickUp_PUBLIC_API_V2.prepared.json")
CONFIG_PATH = Path("generator_config.yml")

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


def normalize_path_params(path: str) -> str:
    """Lower-case path placeholders so they match Terraform attribute names."""
    return re.sub(r"\{(\w+)\}", lambda m: "{" + m.group(1).lower() + "}", path)


def camel_to_snake(name: str) -> str:
    name = re.sub(r"(.)([A-Z][a-z]+)", r"\1_\2", name)
    return re.sub(r"([a-z0-9])([A-Z])", r"\1_\2", name).lower()


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


def operation_method(method: str) -> str:
    return method.lower()


def build_resources(spec: dict) -> dict:
    """Find endpoints with full create/read/update/delete cycles."""
    # Group operations by the entity they act on.
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

    resources = {}
    for base, methods in groups.items():
        # A full Terraform resource needs create, read, update, and delete.
        if "post" not in methods or "get" not in methods:
            continue
        if "patch" not in methods and "put" not in methods:
            continue
        if "delete" not in methods:
            continue

        name = data_source_name(resource_base(next(iter(methods["get"]))["op_id"]))
        if not name:
            name = camel_to_snake(base).strip("_")

        if name in hand_written_resources:
            continue

        def pick(methods_for_kind):
            # Prefer the shortest path for a given method (avoids sub-action variants).
            return min(methods_for_kind, key=lambda x: len(x["path"]))

        create = pick(methods["post"])
        read = pick(methods["get"])
        update = pick(methods.get("patch") or methods["put"])
        delete_op = pick(methods["delete"])

        resources[name] = {
            "create": {
                "path": normalize_path_params(create["path"]),
                "method": operation_method(create["method"]),
            },
            "read": {
                "path": normalize_path_params(read["path"]),
                "method": operation_method(read["method"]),
            },
            "update": {
                "path": normalize_path_params(update["path"]),
                "method": operation_method(update["method"]),
            },
            "delete": {
                "path": normalize_path_params(delete_op["path"]),
                "method": operation_method(delete_op["method"]),
            },
            "schema": {},
        }

    return resources


def main() -> None:
    spec = json.loads(SPEC_PATH.read_text())

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

    # Write as JSON because the YAML parser accepts JSON and PyYAML may not be installed.
    CONFIG_PATH.write_text(json.dumps(config, indent=2))
    print(f"wrote {CONFIG_PATH}")


if __name__ == "__main__":
    main()
