#!/usr/bin/env python3
"""Flatten unsupported OpenAPI schema composition for terraform-plugin-codegen-openapi."""

import json
import re
import sys
from pathlib import Path


def load_spec(path: Path) -> dict:
    text = path.read_text()
    return json.loads(text)


def _decode_json_pointer(part: str) -> str:
    """Decode a JSON pointer segment (~1 -> /, ~0 -> ~)."""
    return part.replace("~1", "/").replace("~0", "~")


def normalize_path_key(path: str) -> str:
    """Lower-case path placeholders so they match Terraform attribute names."""
    return re.sub(r"\{(\w+)\}", lambda m: "{" + m.group(1).lower() + "}", path)


def normalize_path_params_in_spec(spec: dict) -> None:
    """Lower-case path parameter names and path keys so code generation sees
    consistent snake_case placeholders (e.g. {team_id} not {team_Id})."""
    if "paths" not in spec:
        return
    paths = spec["paths"]
    new_paths: dict = {}
    for path, path_item in paths.items():
        norm_path = normalize_path_key(path)
        new_paths[norm_path] = path_item
        for op in path_item.values():
            if not isinstance(op, dict):
                continue
            for param in op.get("parameters", []):
                if isinstance(param, dict) and param.get("in") == "path":
                    param["name"] = param["name"].lower()
    spec["paths"] = new_paths


def resolve(ref_or_schema: any, spec: dict) -> any:
    """Follow a chain of $ref values and return the raw target schema.

    Returns an opaque object for external references or cycles in the $ref
    chain itself.
    """
    if not isinstance(ref_or_schema, dict):
        return ref_or_schema

    if "$ref" not in ref_or_schema:
        return ref_or_schema

    seen = set()
    current = ref_or_schema
    while isinstance(current, dict) and "$ref" in current:
        ref = current["$ref"]
        if not ref.startswith("#/") or ref in seen:
            return {"type": "object"}
        seen.add(ref)

        if ref.startswith("#/components/schemas/"):
            name = ref.split("/")[-1]
            current = spec.get("components", {}).get("schemas", {}).get(name, {})
        else:
            parts = [_decode_json_pointer(p) for p in ref.lstrip("#/").split("/") if p != ""]
            current = spec
            for part in parts:
                if not isinstance(current, dict) or part not in current:
                    current = {}
                    break
                current = current[part]

    return current


def normalize_type(t: any) -> str | None:
    """Return a single string type, handling OpenAPI 3.1 type arrays."""
    if isinstance(t, list):
        for candidate in t:
            if candidate != "null":
                return candidate
        # A type of ["null"] only means the field is nullable with no other
        # type constraints; default to string so the generator can still emit
        # an attribute.
        return "string"
    return t


def merge_schemas(
    schemas: list,
    spec: dict,
    cache: dict[str, any],
    visiting: set[str],
) -> dict:
    merged: dict = {
        "type": "object",
        "properties": {},
        "required": [],
    }

    for s in schemas:
        s = transform(s, spec, cache, visiting)
        if not isinstance(s, dict):
            continue

        if "description" in s:
            merged["description"] = s["description"]
        if "type" in s:
            merged["type"] = normalize_type(s["type"])

        for prop, prop_schema in s.get("properties", {}).items():
            merged["properties"][prop] = transform(prop_schema, spec, cache, visiting)

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
            merged["additionalProperties"] = transform(s["additionalProperties"], spec, cache, visiting)

    if not merged["required"]:
        del merged["required"]
    if not merged["properties"]:
        del merged["properties"]
    return merged


def flatten_one_of(
    schemas: list,
    spec: dict,
    cache: dict[str, any],
    visiting: set[str],
) -> dict:
    resolved_schemas = [transform(s, spec, cache, visiting) for s in schemas]
    types = set()
    for s in resolved_schemas:
        if isinstance(s, dict) and "type" in s:
            t = normalize_type(s["type"])
            if t:
                types.add(t)

    if "object" in types or all(isinstance(s, dict) and normalize_type(s.get("type")) == "object" for s in resolved_schemas if isinstance(s, dict)):
        return merge_schemas(schemas, spec, cache, visiting)

    if types == {"integer", "string"} or types == {"number", "string"}:
        merged = {"type": "string"}
        enums = []
        for s in schemas:
            s = transform(s, spec, cache, visiting)
            if isinstance(s, dict) and "enum" in s:
                for v in s["enum"]:
                    sv = str(v)
                    if sv not in enums:
                        enums.append(sv)
        if enums:
            merged["enum"] = enums
        return merged

    if len(resolved_schemas) > 0:
        first = resolved_schemas[0]
        if isinstance(first, dict):
            return transform(first, spec, cache, visiting)

    return {"type": "string"}


def transform(
    schema: any,
    spec: dict,
    cache: dict[str, any] | None = None,
    visiting: set[str] | None = None,
) -> any:
    if cache is None:
        cache = {}
    if visiting is None:
        visiting = set()

    if not isinstance(schema, dict):
        return schema

    if "$ref" in schema:
        ref = schema["$ref"]
        if not ref.startswith("#/"):
            return schema

        # A cycle in the current transform branch becomes an opaque object.
        if ref in visiting:
            return {"type": "object"}

        # Reuse an already-transformed schema.
        if ref in cache:
            return cache[ref]

        visiting.add(ref)
        raw = resolve(schema, spec)
        transformed = transform(raw, spec, cache, visiting)
        visiting.remove(ref)
        cache[ref] = transformed
        return transformed

    if "allOf" in schema:
        merged = merge_schemas(schema["allOf"], spec, cache, visiting)
        for k, v in schema.items():
            if k not in ("allOf",):
                merged[k] = v
        return transform(merged, spec, cache, visiting)

    if "anyOf" in schema or "oneOf" in schema:
        alternatives = schema.get("anyOf") or schema.get("oneOf") or []
        merged = flatten_one_of(alternatives, spec, cache, visiting)
        for k, v in schema.items():
            if k not in ("anyOf", "oneOf"):
                merged[k] = v
        return transform(merged, spec, cache, visiting)

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
            out[k] = {prop: transform(prop_schema, spec, cache, visiting) for prop, prop_schema in v.items()}
        elif k in ("items", "additionalProperties") and isinstance(v, (dict, bool)):
            out[k] = v if isinstance(v, bool) else transform(v, spec, cache, visiting)
        elif isinstance(v, dict):
            out[k] = transform(v, spec, cache, visiting)
        elif isinstance(v, list):
            out[k] = [transform(e, spec, cache, visiting) if isinstance(e, dict) else e for e in v]
        else:
            out[k] = v
    return out


def normalize_refs(spec: dict) -> None:
    """Lower-case path parameter placeholders inside $ref values.

    Path normalization rewrites path keys like /v2/team/{team_Id}/task to
    /v2/team/{team_id}/task, but $ref JSON pointers still use the original
    placeholder spelling. Rewriting the $ref keeps internal pointers valid so
    that path-level schemas (for example, the tag object shared by task and
    time entry endpoints) are resolved correctly.
    """

    def normalize_ref_value(v: any) -> any:
        if isinstance(v, dict):
            for k, val in v.items():
                if k == "$ref" and isinstance(val, str):
                    v[k] = re.sub(r"\{(\w+)\}", lambda m: "{" + m.group(1).lower() + "}", val)
                else:
                    v[k] = normalize_ref_value(val)
            return v
        if isinstance(v, list):
            return [normalize_ref_value(e) for e in v]
        return v

    for path, path_item in spec.get("paths", {}).items():
        normalize_ref_value(path_item)
    for name, schema in spec.get("components", {}).get("schemas", {}).items():
        spec["components"]["schemas"][name] = normalize_ref_value(schema)


def prepare(spec: dict) -> dict:
    spec = json.loads(json.dumps(spec))
    normalize_path_params_in_spec(spec)
    normalize_refs(spec)
    uniquify_titles(spec)
    spec["components"]["schemas"] = {
        name: transform(schema, spec)
        for name, schema in spec.get("components", {}).get("schemas", {}).items()
    }
    for path, path_item in spec.get("paths", {}).items():
        # Merge path-level parameters into each operation, with operation-level
        # parameters overriding path-level parameters that share (name, in).
        path_params = [transform(p, spec) for p in path_item.pop("parameters", []) if isinstance(p, dict)]
        for _method, operation in path_item.items():
            if not isinstance(operation, dict):
                continue
            op_params = [
                transform(p, spec)
                for p in operation.get("parameters", [])
                if isinstance(p, dict)
            ]
            op_keys = {(p.get("name"), p.get("in")) for p in op_params}
            inherited = [p for p in path_params if (p.get("name"), p.get("in")) not in op_keys]
            merged_params = inherited + op_params
            if merged_params:
                operation["parameters"] = merged_params
            if "requestBody" in operation:
                operation["requestBody"] = transform(operation["requestBody"], spec)
            if "responses" in operation:
                operation["responses"] = {
                    code: transform(resp, spec) for code, resp in operation["responses"].items()
                }
    return spec


def uniquify_titles(spec: dict) -> None:
    """Make sure every object schema has a unique title so the code generator
    does not produce duplicate Go type names within a package."""
    seen = set()

    def walk(v: any) -> None:
        if isinstance(v, dict):
            if v.get("type") == "object" and isinstance(v.get("title"), str):
                title = v["title"]
                original = title
                counter = 1
                while title in seen:
                    title = f"{original}_{counter}"
                    counter += 1
                seen.add(title)
                v["title"] = title
            for item in v.values():
                walk(item)
        elif isinstance(v, list):
            for item in v:
                walk(item)

    walk(spec.get("components", {}).get("schemas", {}))
    walk(spec.get("paths", {}))


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
