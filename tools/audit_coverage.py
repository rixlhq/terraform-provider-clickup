#!/usr/bin/env python3
"""Compare the ClickUp OpenAPI spec against the provider implementation.

Outputs a list of unimplemented paths/methods grouped by API area.
"""

import json
import re
import sys
from collections import defaultdict
from pathlib import Path


def load_json(p: str) -> dict:
    return json.loads(Path(p).read_text())


def area(path: str, operation_id: str | None) -> str:
    """Categorize an endpoint by its URL prefix / operation name."""
    if "/webhook" in path or "Webhook" in (operation_id or ""):
        return "Webhooks"
    if "/time" in path or "time" in (operation_id or "").lower():
        return "Time tracking"
    if "/goal" in path or "/key_result" in path or "Goal" in (operation_id or "") or "KeyResult" in (operation_id or ""):
        return "Goals / key results"
    if "/task" in path:
        return "Tasks"
    if "/list" in path:
        return "Lists"
    if "/folder" in path:
        return "Folders"
    if "/space" in path:
        return "Spaces"
    if "/view" in path:
        return "Views"
    if "/comment" in path:
        return "Comments"
    if "/team" in path or "/group" in path or "/user" in path or "/guest" in path:
        return "Users, teams, guests"
    if "/custom" in path or "/field" in path:
        return "Custom fields / items"
    if "/attachment" in path:
        return "Attachments"
    if "/tag" in path:
        return "Tags"
    if "/checklist" in path:
        return "Checklists"
    return "Other"


def main() -> None:
    if len(sys.argv) < 2:
        print(f"usage: {sys.argv[0]} <ClickUp_PUBLIC_API_V2.prepared.json>", file=sys.stderr)
        sys.exit(1)

    spec_path = Path(sys.argv[1])
    spec = load_json(spec_path)

    cfg = load_json("generator_config.yml")
    ds_paths = {
        (cfg["data_sources"][name]["read"]["path"], cfg["data_sources"][name]["read"]["method"])
        for name in cfg["data_sources"]
    }
    res_create = {
        (cfg["resources"][name]["create"]["path"], cfg["resources"][name]["create"]["method"])
        for name in cfg["resources"]
    }
    res_read = {
        (cfg["resources"][name]["read"]["path"], cfg["resources"][name]["read"]["method"])
        for name in cfg["resources"]
    }
    res_update = {
        (cfg["resources"][name]["update"]["path"], cfg["resources"][name]["update"]["method"])
        for name in cfg["resources"]
    }
    res_delete = {
        (cfg["resources"][name]["delete"]["path"], cfg["resources"][name]["delete"]["method"])
        for name in cfg["resources"]
    }

    # Hand-written resources discovered by provider.go and provider package.
    import re as _re
    provider_src = Path("internal/provider").glob("*.go")
    manual_resources = set()
    for f in provider_src:
        text = f.read_text()
        for m in _re.finditer(r'func new([A-Za-z]+)Resource\(\)', text):
            manual_resources.add(m.group(1).lower())

    # data sources present in datasources.go
    ds_file = Path("internal/provider/datasources.go").read_text()
    manual_datasources = set(_re.findall(r'func new([A-Za-z]+)DataSource\(\)', ds_file))

    # View resources are hand-written and not reflected in generator_config.yml.
    view_resources = {
        ("/v2/team/{team_id}/view", "post"),
        ("/v2/folder/{folder_id}/view", "post"),
        ("/v2/list/{list_id}/view", "post"),
        ("/v2/space/{space_id}/view", "post"),
        ("/v2/view/{view_id}", "get"),
        ("/v2/view/{view_id}", "put"),
        ("/v2/view/{view_id}", "delete"),
    }

    implemented_paths = ds_paths | res_create | res_read | res_update | res_delete | view_resources

    # Hand-written raw data sources in datasources_manual.go.
    manual_ds_src = Path("internal/provider/datasources_manual.go").read_text()
    for m in _re.finditer(r'newRawJSONDataSource\("([^"]+)",\s*"([^"]+)"', manual_ds_src):
        implemented_paths.add((m.group(2), "get"))

    # Hand-written resources in the provider package.  Detect simple path
    # constants and genericResource initializers.
    for f in Path("internal/provider").glob("*_resource.go"):
        text = f.read_text()
        for m in _re.finditer(r'"(/v2/[^"]+)"', text):
            # Heuristic: a resource file declaring a path likely implements it
            # for at least one of create/read/update/delete.  This avoids
            # double-counting without parsing Go code.
            for method in ("post", "get", "put", "delete"):
                implemented_paths.add((m.group(1), method))

    missing: dict[str, list[tuple[str, str, str]]] = defaultdict(list)
    total = 0
    implemented = 0
    for path, ops in spec["paths"].items():
        for method, op in ops.items():
            if not isinstance(op, dict):
                continue
            oid = op.get("operationId", "")
            # skip OAuth token endpoint
            if path == "/v2/oauth/token":
                continue
            total += 1
            key = (path, method)
            if key in implemented_paths:
                implemented += 1
                continue
            missing[area(path, oid)].append((method.upper(), path, oid))

    print(f"Total endpoints: {total}")
    print(f"Implemented via generator or hand-written: {implemented}")
    print(f"Missing: {total - implemented}")
    print()
    for a in sorted(missing):
        print(f"## {a} ({len(missing[a])} missing)")
        for m, p, oid in sorted(missing[a]):
            print(f"  {m} {p}  [{oid}]")
        print()


if __name__ == "__main__":
    main()
