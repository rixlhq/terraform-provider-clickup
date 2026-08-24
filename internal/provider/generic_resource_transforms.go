package provider

import (
	"encoding/json"
	"strconv"
)

// extractAddList converts ClickUp's { add: [...], rem: [...] } update shape
// into a plain list for use in create request bodies.
func extractAddList(v any) any {
	if v == nil {
		return []any{}
	}
	m, ok := v.(map[string]any)
	if !ok {
		return v
	}
	add, ok := m["add"]
	if !ok || add == nil {
		return []any{}
	}
	return add
}

// listToAddRemObject converts a plain list returned by the ClickUp API into
// the { add: [...], rem: [] } object shape used by the Terraform schema for
// assignees, group_assignees, and watchers. When an element is an object with
// an "id" (or "group_id") field, the identifier is extracted; otherwise the
// primitive value is used as-is.
func listToAddRemObject(v any) any {
	if v == nil {
		return map[string]any{"add": []any{}, "rem": []any{}}
	}
	if m, ok := v.(map[string]any); ok {
		return m
	}
	s, ok := v.([]any)
	if !ok {
		return map[string]any{"add": []any{v}, "rem": []any{}}
	}
	add := make([]any, 0, len(s))
	for _, elem := range s {
		if id := extractListItemID(elem); id != nil {
			add = append(add, id)
			continue
		}
		add = append(add, elem)
	}
	return map[string]any{"add": add, "rem": []any{}}
}

func extractListItemID(v any) any {
	obj, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	for _, key := range []string{"id", "group_id", "user_id"} {
		if id, ok := obj[key]; ok && id != nil {
			return id
		}
	}
	return nil
}

// userObjectsToIntList converts a list of user/group objects (each with an
// "id" field) into a list of numeric IDs. Non-object values are passed through
// so that already-numeric lists remain intact.
func userObjectsToIntList(v any) any {
	list, ok := v.([]any)
	if !ok {
		return v
	}

	out := make([]any, 0, len(list))
	for _, elem := range list {
		if id := extractListItemID(elem); id != nil {
			out = append(out, id)
			continue
		}
		if elem != nil {
			out = append(out, elem)
		}
	}
	return out
}

// stringToInt parses a numeric string for use in a create request body where
// the create endpoint expects an integer but the update endpoint accepts a
// string (e.g., list assignee). Empty values and the sentinel "none" are
// treated as unset and returned as nil. Other non-numeric strings are left as
// strings so the ClickUp API returns a meaningful error instead of silently
// dropping the value.
func stringToInt(v any) any {
	if v == nil {
		return nil
	}
	s, ok := v.(string)
	if !ok {
		return v
	}
	if s == "" || s == "none" {
		return nil
	}
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return s
	}
	return i
}

// parseCustomFieldValues converts the JSON-encoded `value` strings in a task's
// custom_fields list back to the real JSON values the ClickUp API expects.
// Bare strings that are not valid JSON are sent unchanged, which is what the
// API expects for text and short-text custom fields.
func parseCustomFieldValues(v any) any {
	list, ok := v.([]any)
	if !ok {
		return v
	}
	out := make([]any, 0, len(list))
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			out = append(out, item)
			continue
		}
		if raw, ok := m["value"]; ok {
			if s, ok := raw.(string); ok && s != "" {
				var parsed any
				if err := json.Unmarshal([]byte(s), &parsed); err == nil {
					m["value"] = parsed
				}
				// If Unmarshal fails, leave the raw string as the value.
			}
		}
		out = append(out, m)
	}
	return out
}

// stringifyCustomFieldValues converts the polymorphic `value` returned by the
// ClickUp API into a JSON-encoded string so it can be stored in a Terraform
// StringAttribute. Practitioners can use jsondecode() when they need the
// original structure.
func stringifyCustomFieldValues(v any) any {
	list, ok := v.([]any)
	if !ok {
		return v
	}
	out := make([]any, 0, len(list))
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			out = append(out, item)
			continue
		}
		if raw, ok := m["value"]; ok && raw != nil {
			b, err := json.Marshal(raw)
			if err == nil {
				m["value"] = string(b)
			}
		}
		out = append(out, m)
	}
	return out
}
