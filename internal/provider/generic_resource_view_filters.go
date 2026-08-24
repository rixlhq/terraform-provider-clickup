package provider

import (
	"encoding/json"
)

// stringifyFilterValues converts the polymorphic `values` field of each
// filter object into a JSON-encoded string before the response is converted
// to Terraform state. The view schema stores `values` as a string so that
// practitioners can use jsonencode() to represent strings, arrays, or objects.
func stringifyFilterValues(v any) any {
	m, ok := v.(map[string]any)
	if !ok {
		return v
	}

	fields, ok := m["fields"].([]any)
	if !ok {
		return v
	}

	for _, f := range fields {
		fm, ok := f.(map[string]any)
		if !ok {
			continue
		}

		raw, ok := fm["values"]
		if !ok || raw == nil {
			delete(fm, "values")
			continue
		}

		b, err := json.Marshal(raw)
		if err != nil {
			continue
		}
		fm["values"] = string(b)
	}

	return m
}

// parseFilterValues reverses stringifyFilterValues, turning the JSON-encoded
// `values` strings from the Terraform plan back into the real values the
// ClickUp API expects.
func parseFilterValues(v any) any {
	m, ok := v.(map[string]any)
	if !ok {
		return v
	}

	fields, ok := m["fields"].([]any)
	if !ok {
		return v
	}

	for _, f := range fields {
		fm, ok := f.(map[string]any)
		if !ok {
			continue
		}

		raw, ok := fm["values"]
		if !ok || raw == nil {
			continue
		}

		s, ok := raw.(string)
		if !ok {
			continue
		}

		if s == "" {
			delete(fm, "values")
			continue
		}

		var parsed any
		if err := json.Unmarshal([]byte(s), &parsed); err == nil {
			if parsed == nil {
				delete(fm, "values")
			} else {
				fm["values"] = parsed
			}
		}
	}

	return m
}
