#!/usr/bin/env python3
"""Generate internal/provider/resources.go from generator_config.yml and the prepared spec."""

import json
import re
from pathlib import Path

CONFIG_PATH = Path("generator_config.yml")
SPEC_PATH = Path("ClickUp_PUBLIC_API_V2.prepared.json")
OUTPUT_PATH = Path("internal/provider/resources.go")


VIEW_VARIANTS = [
    ("view", "/v2/team/{team_id}/view"),
    ("folder_view", "/v2/folder/{folder_id}/view"),
    ("list_view", "/v2/list/{list_id}/view"),
    ("space_view", "/v2/space/{space_id}/view"),
]

VIEW_OVERRIDES = {
    "createResponseRoot": "view",
    "readResponseRoot": "view",
    "preReadTransforms": {"filters": "stringifyFilterValues"},
    "createBodyTransforms": {"filters": "parseFilterValues"},
    "updateBodyTransforms": {"filters": "parseFilterValues"},
}


RESOURCE_VIEW_PKG = 'resource_view "github.com/rixlhq/terraform-provider-clickup/internal/provider/generated/resource_view"'


def snake_to_pascal(name: str) -> str:
    return "".join(part.capitalize() for part in name.split("_") if part)


def find_operation(spec: dict, path: str, method: str) -> dict | None:
    item = spec.get("paths", {}).get(path, {})
    return item.get(method)


def request_body_properties(op: dict | None) -> list[str]:
    if not op:
        return []
    schema = (
        op.get("requestBody", {})
        .get("content", {})
        .get("application/json", {})
        .get("schema", {})
    )
    return list(schema.get("properties", {}).keys())


def request_body_schema(op: dict | None) -> dict:
    if not op:
        return {}
    return (
        op.get("requestBody", {})
        .get("content", {})
        .get("application/json", {})
        .get("schema", {})
    )


def is_add_rem_object(schema: dict) -> bool:
    if schema.get("type") != "object":
        return False
    props = schema.get("properties", {})
    return "add" in props and "rem" in props


def is_custom_fields_value_object(schema: dict) -> bool:
    """Detect a custom_fields list whose items are objects with a value field."""
    if schema.get("type") != "array" or not isinstance(schema.get("items"), dict):
        return False
    item = schema["items"]
    props = item.get("properties", {})
    return "value" in props


def read_response_properties(op: dict | None) -> dict:
    """Return the read response body JSON schema properties, or {}."""
    if not op:
        return {}
    content = (
        op.get("responses", {})
        .get("200", {})
        .get("content", {})
        .get("application/json", {})
    )
    if not content:
        # Some read responses wrap the object under a "data" or resource-named key.
        # Fall back to the first available response content.
        for resp in op.get("responses", {}).values():
            c = resp.get("content", {}).get("application/json", {})
            if c:
                content = c
                break
    schema = content.get("schema", {})
    while schema.get("type") == "array":
        schema = schema.get("items", {})
    return schema.get("properties", {})


def create_body_transforms(create_op: dict | None, update_op: dict | None) -> dict[str, str]:
    """Return fields that need a create-body transform.

    Maps field name to the Go helper that performs the transform.
    """
    create_schema = request_body_schema(create_op)
    update_schema = request_body_schema(update_op)
    create_props = create_schema.get("properties", {})
    update_props = update_schema.get("properties", {})
    fields: dict[str, str] = {}
    for key, create_prop in create_props.items():
        update_prop = update_props.get(key)

        if key == "custom_fields" and is_custom_fields_value_object(create_prop):
            fields[key] = "parseCustomFieldValues"

        if not isinstance(update_prop, dict):
            continue

        create_type = create_prop.get("type")
        update_type = update_prop.get("type")

        if create_type == "array" and is_add_rem_object(update_prop):
            fields[key] = "extractAddList"
        elif create_type in ("integer", "number") and update_type == "string":
            fields[key] = "stringToInt"
    return fields


def update_body_transforms(
    update_op: dict | None,
) -> dict[str, str]:
    """Return fields that need an update-body transform.

    Currently handles custom_fields values, which are JSON-encoded strings in the
    Terraform schema but polymorphic in the ClickUp API.
    """
    if not update_op:
        return {}

    update_schema = request_body_schema(update_op)
    update_props = update_schema.get("properties", {})
    fields: dict[str, str] = {}

    for key, update_prop in update_props.items():
        if key == "custom_fields" and is_custom_fields_value_object(update_prop):
            fields[key] = "parseCustomFieldValues"

    return fields


def is_array_of_strings(schema: dict) -> bool:
    if schema.get("type") != "array":
        return False
    items = schema.get("items", {})
    return isinstance(items, dict) and items.get("type") == "string"


def is_array_of_objects(schema: dict) -> bool:
    if schema.get("type") != "array":
        return False
    items = schema.get("items", {})
    return isinstance(items, dict) and items.get("type") == "object"


def read_body_transforms(
    create_op: dict | None, update_op: dict | None, read_op: dict | None
) -> dict[str, str]:
    """Return fields that the API returns in one shape but the Terraform schema models as another."""
    create_schema = request_body_schema(create_op)
    create_props = create_schema.get("properties", {})
    update_schema = request_body_schema(update_op)
    update_props = update_schema.get("properties", {})
    read_props = read_response_properties(read_op)
    fields: dict[str, str] = {}

    # ClickUp returns assignees/group_assignees/watchers as a list of objects,
    # but the Terraform schema uses the update { add, rem } object for all operations.
    for key, update_prop in update_props.items():
        if not is_add_rem_object(update_prop):
            continue
        read_prop = read_props.get(key)
        if not isinstance(read_prop, dict):
            continue
        if read_prop.get("type") == "array":
            fields[key] = "listToAddRemObject"

    # Custom Fields are returned with polymorphic values; the schema uses a
    # JSON-encoded string, so we need to stringify the response values.
    if "custom_fields" in read_props and is_custom_fields_value_object(read_props["custom_fields"]):
        fields["custom_fields"] = "stringifyCustomFieldValues"

    # Some scalar fields are returned as objects (e.g. status, priority).
    # Resolve the scalar value from the object when possible.
    for key in {*create_props.keys(), *update_props.keys()}:
        if key in fields:
            continue
        read_prop = read_props.get(key)
        if not isinstance(read_prop, dict) or read_prop.get("type") != "object":
            continue
        update_type = update_props.get(key, {}).get("type")
        create_type = create_props.get(key, {}).get("type")
        if update_type == "string" or create_type == "string":
            fields[key] = "objectFieldToString"
        elif update_type in ("integer", "number") or create_type in ("integer", "number"):
            fields[key] = "objectFieldToInt"

    # Tags are returned as a list of objects but sent as a list of strings.
    # Skip fields that already have a transform, such as group_assignees,
    # which the update schema models as an { add, rem } object.
    for key, create_prop in create_props.items():
        if key in fields:
            continue
        if not is_array_of_strings(create_prop):
            continue
        read_prop = read_props.get(key)
        if is_array_of_objects(read_prop):
            fields[key] = "tagObjectsToStrings"

    return fields


# Resource names that are implemented by hand-written Go files rather than
# the tfplugingen-framework generator.
HAND_WRITTEN_RESOURCES = {"folder"}

# Override the auto-detected read transforms for resources where the ClickUp API
# response shape differs from what the OpenAPI schema declares.
READ_TRANSFORM_OVERRIDES = {
    "list": {"assignee": "objectFieldToString"},
    "folderless_list": {"assignee": "objectFieldToString"},
    "goal": {"owners": "userObjectsToIntList"},
}

# Resources whose GET response wraps the object under a named key.
READ_RESPONSE_ROOT_OVERRIDES = {
    "goal": "goal",
}


def go_string_slice(vals: list[str]) -> str:
    quoted = [f'"{v}"' for v in vals]
    return f"[]string{{{', '.join(quoted)}}}"


def go_map_string_func(m: dict[str, str]) -> str:
    if not m:
        return ""
    pairs = [f'\t\t\t"{k}": {v},' for k in sorted(m) for v in [m[k]]]
    return "map[string]func(any) any{\n" + "\n".join(pairs) + "\n\t\t}"


def write_resource(
    lines: list[str],
    name: str,
    create_path: str,
    read_path: str,
    update_path: str,
    delete_path: str,
    update_method: str,
    create_body: list[str],
    update_body: list[str],
    create_body_transforms: dict[str, str],
    update_body_transforms: dict[str, str],
    read_transforms: dict[str, str],
    pre_read_transforms: dict[str, str],
    read_from_list: bool,
    read_list_root: str,
    read_list_id_field: str,
    create_response_root: str,
    read_response_root: str,
    id_from_body: list[str],
    schema_func: str,
) -> None:
    pascal = snake_to_pascal(name)
    lines.append(f"func new{pascal}Resource() resource.Resource {{")
    lines.append("\treturn &genericResource{")
    lines.append(f'\t\tname:             "{name}",')
    lines.append(f'\t\tcreatePath:       "{create_path}",')
    lines.append(f'\t\treadPath:         "{read_path}",')
    lines.append(f'\t\tupdatePath:       "{update_path}",')
    lines.append(f'\t\tdeletePath:       "{delete_path}",')
    lines.append(f'\t\tupdateMethod:     "{update_method}",')
    lines.append(f"\t\tcreateBodyFields: {go_string_slice(create_body)},")
    lines.append(f"\t\tupdateBodyFields: {go_string_slice(update_body)},")
    if create_body_transforms:
        lines.append("\t\tcreateBodyTransforms: " + go_map_string_func(create_body_transforms) + ",")
    if update_body_transforms:
        lines.append("\t\tupdateBodyTransforms: " + go_map_string_func(update_body_transforms) + ",")
    if read_transforms:
        lines.append("\t\treadTransforms: " + go_map_string_func(read_transforms) + ",")
    if pre_read_transforms:
        lines.append("\t\tpreReadTransforms: " + go_map_string_func(pre_read_transforms) + ",")
    if read_from_list:
        lines.append("\t\treadFromList:     true,")
        lines.append(f'\t\treadListRoot:     "{read_list_root}",')
        lines.append(f'\t\treadListIDField:  "{read_list_id_field}",')
    if create_response_root:
        lines.append(f'\t\tcreateResponseRoot: "{create_response_root}",')
    if read_response_root:
        lines.append(f'\t\treadResponseRoot: "{read_response_root}",')
    if id_from_body:
        lines.append(f"\t\tidFromBody: {go_string_slice(id_from_body)},")
    lines.append(f"\t\tschemaFunc:       {schema_func},")
    lines.append("\t}")
    lines.append("}")
    lines.append("")


def main() -> None:
    config = json.loads(CONFIG_PATH.read_text())
    spec = json.loads(SPEC_PATH.read_text())

    resources = config.get("resources", {})

    lines = [
        "// Code generated by tools/generate_resources.py; DO NOT EDIT.",
        "",
        "package provider",
        "",
        "import (",
        '\t"github.com/hashicorp/terraform-plugin-framework/resource"',
        "",
    ]
    for name in sorted(resources):
        if name in HAND_WRITTEN_RESOURCES:
            continue
        if not Path(f"internal/provider/generated/resource_{name}").exists():
            continue
        lines.append(
            f'\tresource_{name} "github.com/rixlhq/terraform-provider-clickup/internal/provider/generated/resource_{name}"'
        )

    # view is hand-written and not in generator_config.yml, but it (and its
    # hierarchy variants) still need to be wired into the provider.
    lines.append("\t" + RESOURCE_VIEW_PKG)
    lines.append(")")
    lines.append("")

    # Build the resource factory list from generated config entries...
    factory_names: list[str] = []
    for name in sorted(resources):
        if name in HAND_WRITTEN_RESOURCES:
            continue
        if not Path(f"internal/provider/generated/resource_{name}").exists():
            continue
        pascal = snake_to_pascal(name)
        factory_names.append(f"\tnew{pascal}Resource,")

    # ...and append the view variants.
    for name, _ in VIEW_VARIANTS:
        factory_names.append(f"\tnew{snake_to_pascal(name)}Resource,")

    lines.append("var resourceFactories = []func() resource.Resource{")
    lines.extend(factory_names)
    lines.append("}")
    lines.append("")

    # Generated resources from the config.
    for name, cfg in sorted(resources.items()):
        if name in HAND_WRITTEN_RESOURCES:
            continue
        if not Path(f"internal/provider/generated/resource_{name}").exists():
            continue
        create = cfg["create"]
        read = cfg["read"]
        update = cfg["update"]
        delete = cfg["delete"]

        create_op = find_operation(spec, create["path"], create["method"])
        read_op = find_operation(spec, read["path"], read["method"])
        update_op = find_operation(spec, update["path"], update["method"])
        create_body = request_body_properties(create_op)
        update_body = request_body_properties(update_op)
        create_transform_fields = create_body_transforms(create_op, update_op)
        update_transform_fields = update_body_transforms(update_op)
        read_transform_fields = read_body_transforms(create_op, update_op, read_op)
        read_transform_fields.update(READ_TRANSFORM_OVERRIDES.get(name, {}))

        write_resource(
            lines,
            name=name,
            create_path=create["path"],
            read_path=read["path"],
            update_path=update["path"],
            delete_path=delete["path"],
            update_method=update["method"],
            create_body=create_body,
            update_body=update_body,
            create_body_transforms=create_transform_fields,
            update_body_transforms=update_transform_fields,
            read_transforms=read_transform_fields,
            pre_read_transforms={},
            read_from_list=False,
            read_list_root="",
            read_list_id_field="",
            create_response_root="",
            read_response_root=READ_RESPONSE_ROOT_OVERRIDES.get(name, ""),
            id_from_body=[],
            schema_func=f"resource_{name}.{snake_to_pascal(name)}ResourceSchema",
        )

    # Hand-wired view resources (team/folder/list/space) that share the
    # hand-written resource_view schema.
    view_create_body = ["name", "type", "grouping", "divide", "sorting", "filters", "columns", "team_sidebar", "settings"]
    view_update_body = ["name", "type", "parent", "grouping", "divide", "sorting", "filters", "columns", "team_sidebar", "settings"]
    view_create_transforms = VIEW_OVERRIDES["createBodyTransforms"]
    view_update_transforms = VIEW_OVERRIDES["updateBodyTransforms"]
    view_pre_read_transforms = VIEW_OVERRIDES["preReadTransforms"]

    for name, create_path in VIEW_VARIANTS:
        write_resource(
            lines,
            name=name,
            create_path=create_path,
            read_path="/v2/view/{view_id}",
            update_path="/v2/view/{view_id}",
            delete_path="/v2/view/{view_id}",
            update_method="put",
            create_body=view_create_body,
            update_body=view_update_body,
            create_body_transforms=view_create_transforms,
            update_body_transforms=view_update_transforms,
            read_transforms={},
            pre_read_transforms=view_pre_read_transforms,
            read_from_list=False,
            read_list_root="",
            read_list_id_field="",
            create_response_root=VIEW_OVERRIDES["createResponseRoot"],
            read_response_root=VIEW_OVERRIDES["readResponseRoot"],
            id_from_body=[],
            schema_func="resource_view.ViewResourceSchema",
        )

    OUTPUT_PATH.write_text("\n".join(lines) + "\n")
    print(f"wrote {OUTPUT_PATH}")


if __name__ == "__main__":
    main()
