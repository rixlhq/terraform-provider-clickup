package provider

import (
	"encoding/json"
	"strconv"
	"strings"
)

//nolint:goconst // "status" key is part of a set of string literal fallback keys.
func objectFieldToString(v any) any {
	if v == nil {
		return nil
	}
	if s, ok := v.(string); ok {
		return s
	}
	if n, ok := v.(json.Number); ok {
		return n.String()
	}
	if m, ok := v.(map[string]any); ok {
		for _, key := range []string{"status", "id", "name"} {
			if raw, ok := m[key]; ok && raw != nil {
				if s, ok := raw.(string); ok {
					return s
				}
				if n, ok := raw.(json.Number); ok {
					return n.String()
				}
			}
		}
	}
	return nil
}

// objectFieldToInt extracts an integer from a ClickUp response object.
// It is used for priority, where the API returns an object with numeric
// keys (orderindex, id) or a human-readable priority name that must be mapped.
func objectFieldToInt(v any) any {
	if v == nil {
		return nil
	}
	if m, ok := v.(map[string]any); ok {
		for _, key := range []string{"orderindex", "id"} {
			if raw, ok := m[key]; ok && raw != nil {
				if n := numericFromAny(raw); n != nil {
					return n
				}
			}
		}
		if s, ok := m["priority"].(string); ok {
			return priorityNameToInt(s)
		}
	}
	return numericFromAny(v)
}

func priorityNameToInt(name string) any {
	switch strings.ToLower(name) {
	case "urgent":
		return int64(1)
	case "high":
		return int64(2)
	case "normal":
		return int64(3)
	case "low":
		return int64(4)
	case "no", "none", "":
		return nil
	}
	// Fallback: if the name looks like a number, use it.
	if i, err := strconv.ParseInt(name, 10, 64); err == nil {
		return i
	}
	return nil
}

func numericFromAny(v any) any {
	switch x := v.(type) {
	case json.Number:
		if i, err := x.Int64(); err == nil {
			return i
		}
		if f, err := strconv.ParseFloat(x.String(), 64); err == nil {
			return int64(f)
		}
	case string:
		if x == "" {
			return nil
		}
		if i, err := strconv.ParseInt(x, 10, 64); err == nil {
			return i
		}
		if f, err := strconv.ParseFloat(x, 64); err == nil {
			return int64(f)
		}
	case int:
		return int64(x)
	case int32:
		return int64(x)
	case int64:
		return x
	case float64:
		return int64(x)
	}
	return nil
}

// tagObjectsToStrings extracts the "name" field from a list of tag objects.
// The ClickUp task GET response returns tags as objects, while the create
// and update request bodies accept a list of tag name strings.
func tagObjectsToStrings(v any) any {
	if v == nil {
		return nil
	}
	list, ok := v.([]any)
	if !ok {
		return v
	}
	out := make([]any, 0, len(list))
	for _, item := range list {
		if s, ok := item.(string); ok {
			out = append(out, s)
			continue
		}
		if m, ok := item.(map[string]any); ok {
			if raw, ok := m["name"]; ok && raw != nil {
				out = append(out, raw)
			}
		}
	}
	return out
}
