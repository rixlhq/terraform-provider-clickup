#!/usr/bin/env python3
"""Flatten unsupported OpenAPI schema composition for terraform-plugin-codegen-openapi."""

import json
import sys
from pathlib import Path


def load_spec(path: Path) -> dict:
    text = path.read_text()
    return json.loads(text)


def _decode_json_pointer(part: str) -> str:
    """Decode a JSON pointer segment (~1 -> /, ~0 -> ~)."""
    return part.replace("~1", "/").replace("~0", "~")


def resolve(ref_or_schema: any, spec: dict, visited: set) -> any:
    if not isinstance(ref_or_schema, dict):
        return ref_or_schema

    if "$ref" in ref_or_schema:
        ref = ref_or_schema["$ref"]
        if not ref.startswith("#/"):
            return ref_or_schema

        if ref in visited:
            return {"type": "object"}
        visited.add(ref)

        if ref.startswith("#/components/schemas/"):
            name = ref.split("/")[-1]
            component = spec["components"]["schemas"].get(name, {})
            return resolve(component, spec, visited)

        # Resolve other internal JSON pointers (e.g. into paths).
        parts = [_decode_json_pointer(p) for p in ref.lstrip("#/").split("/") if p != ""]
        current = spec
        for part in parts:
            if not isinstance(current, dict) or part not in current:
                return {"type": "object"}
            current = current[part]
        return resolve(current, spec, visited)

    return ref_or_schema


def normalize_type(t: any) -> str | None:
    """Return a single string type, handling OpenAPI 3.1 type arrays."""
    if isinstance(t, list):
        for candidate in t:
            if candidate != "null":
                return candidate
        return None
    return t


def merge_schemas(schemas: list, spec: dict, visited: set) -> dict:
    merged: dict = {
        "type": "object",
        "properties": {},
        "required": [],
    }

    for s in schemas:
        s = resolve(s, spec, visited)
        if not isinstance(s, dict):
            continue

        if "description" in s:
            merged["description"] = s["description"]
        if "type" in s:
            merged["type"] = normalize_type(s["type"])

        for prop, prop_schema in s.get("properties", {}).items():
            merged["properties"][prop] = transform(prop_schema, spec, visited)

        for req in s.get("required", []):
            if req not in merged["required"]:
                merged["required"].append(req)

        if "enum" in s:
            merged.setdefault("enum", [])
            for v in s["enum"]:
                if v not in merged["enum"]:
                    merged["enum"].append(v)

        if s.get("additionalProperties") is False:
            merged["additionalProperties"] = False
        elif "additionalProperties" in s and s["additionalProperties"] is not False:
            merged["additionalProperties"] = transform(s["additionalProperties"], spec, visited)

    if not merged["required"]:
        del merged["required"]
    if not merged["properties"]:
        del merged["properties"]
    return merged


def flatten_one_of(schemas: list, spec: dict, visited: set) -> dict:
    resolved = [resolve(s, spec, set(visited)) for s in schemas]
    types = set()
    for s in resolved:
        if isinstance(s, dict) and "type" in s:
            t = normalize_type(s["type"])
            if t:
                types.add(t)

    if "object" in types or all(isinstance(s, dict) and normalize_type(s.get("type")) == "object" for s in resolved if isinstance(s, dict)):
        return merge_schemas(schemas, spec, visited)

    if types == {"integer", "string"} or types == {"number", "string"}:
        merged = {"type": "string"}
        enums = []
        for s in schemas:
            s = resolve(s, spec, set(visited))
            if isinstance(s, dict) and "enum" in s:
                for v in s["enum"]:
                    sv = str(v)
                    if sv not in enums:
                        enums.append(sv)
        if enums:
            merged["enum"] = enums
        return merged

    if len(resolved) > 0:
        first = resolved[0]
        if isinstance(first, dict):
            return transform(first, spec, visited)

    return {"type": "string"}


def transform(schema: any, spec: dict, visited: set | None = None) -> any:
    if visited is None:
        visited = set()

    if not isinstance(schema, dict):
        return schema

    schema = resolve(schema, spec, visited)
    if not isinstance(schema, dict):
        return schema

    if "allOf" in schema:
        merged = merge_schemas(schema["allOf"], spec, visited)
        for k, v in schema.items():
            if k not in ("allOf",):
                merged[k] = v
        return transform(merged, spec, visited)

    if "anyOf" in schema or "oneOf" in schema:
        alternatives = schema.get("anyOf") or schema.get("oneOf") or []
        merged = flatten_one_of(alternatives, spec, visited)
        for k, v in schema.items():
            if k not in ("anyOf", "oneOf"):
                merged[k] = v
        return transform(merged, spec, visited)

    out = {}
    for k, v in schema.items():
        if k in ("allOf", "anyOf", "oneOf"):
            continue
        if k == "type":
            t = normalize_type(v)
            if t:
                out[k] = t
            continue
        if k == "properties" and isinstance(v, dict):
            out[k] = {prop: transform(prop_schema, spec, visited) for prop, prop_schema in v.items()}
        elif k in ("items", "additionalProperties") and isinstance(v, (dict, bool)):
            out[k] = v if isinstance(v, bool) else transform(v, spec, visited)
        elif isinstance(v, dict):
            out[k] = transform(v, spec, visited)
        elif isinstance(v, list):
            out[k] = [transform(e, spec, visited) if isinstance(e, dict) else e for e in v]
        else:
            out[k] = v
    return out


def prepare(spec: dict) -> dict:
    spec = json.loads(json.dumps(spec))
    spec["components"]["schemas"] = {
        name: transform(schema, spec)
        for name, schema in spec.get("components", {}).get("schemas", {}).items()
    }
    for path, path_item in spec.get("paths", {}).items():
        # Merge path-level parameters into each operation.
        path_params = [transform(p, spec) for p in path_item.pop("parameters", []) if isinstance(p, dict)]
        for method, operation in path_item.items():
            if not isinstance(operation, dict):
                continue
            if "parameters" in operation:
                op_params = [transform(p, spec) for p in operation["parameters"] if isinstance(p, dict)]
                operation["parameters"] = path_params + op_params
            elif path_params:
                operation["parameters"] = path_params
            if "requestBody" in operation:
                operation["requestBody"] = transform(operation["requestBody"], spec)
            if "responses" in operation:
                operation["responses"] = {
                    code: transform(resp, spec) for code, resp in operation["responses"].items()
                }
    return spec


def main() -> None:
    if len(sys.argv) < 3:
        print(f"usage: {sys.argv[0]} <input.json> <output.json>", file=sys.stderr)
        sys.exit(1)
    input_path = Path(sys.argv[1])
    output_path = Path(sys.argv[2])
    spec = load_spec(input_path)
    prepared = prepare(spec)
    output_path.write_text(json.dumps(prepared, indent=2))
    print(f"wrote {output_path}")


if __name__ == "__main__":
    main()
