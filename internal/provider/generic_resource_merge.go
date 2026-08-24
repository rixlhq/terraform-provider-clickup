package provider

import (
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// mergeTfValues returns a new tftypes.Value that prefers the latest value.
// Unknown values in the latest value (e.g. fields omitted from an API response)
// fall back to the base value so request-only and path attributes are preserved.
// Explicit null values in the latest value are preserved as null so a field that
// the API returns as null will clear the corresponding state value.
func mergeTfValues(base, latest tftypes.Value) (tftypes.Value, error) {
	if !latest.IsKnown() {
		if base.IsKnown() && !base.IsNull() {
			return base, nil
		}
		return tftypes.NewValue(latest.Type(), nil), nil
	}
	if latest.IsNull() {
		return tftypes.NewValue(latest.Type(), nil), nil
	}
	if !base.IsKnown() || base.IsNull() || !base.Type().Is(latest.Type()) {
		return latest, nil
	}

	switch t := latest.Type().(type) {
	case tftypes.Object:
		return mergeTfObjects(t, base, latest)
	case tftypes.List, tftypes.Set, tftypes.Tuple:
		// Keep a known empty collection, but fall back to base when the API
		// returns a non-empty collection that only contains nulls.
		if allElementsNull(latest) {
			return base, nil
		}
		return latest, nil
	}

	return latest, nil
}

func mergeTfObjects(t tftypes.Object, base, latest tftypes.Value) (tftypes.Value, error) {
	baseObj, err := asObject(base)
	if err != nil {
		return latest, nil
	}
	latestObj, err := asObject(latest)
	if err != nil {
		return latest, nil
	}
	merged := make(map[string]tftypes.Value, len(t.AttributeTypes))
	for attr, attrType := range t.AttributeTypes {
		baseVal := baseObj[attr]
		latestVal := latestObj[attr]
		m, err := mergeTfValues(baseVal, latestVal)
		if err != nil {
			return tftypes.Value{}, err
		}
		if m.Type() == nil {
			m = tftypes.NewValue(attrType, nil)
		}
		merged[attr] = m
	}
	return tftypes.NewValue(t, merged), nil
}

func allElementsNull(v tftypes.Value) bool {
	var elems []tftypes.Value
	if err := v.As(&elems); err != nil {
		return false
	}
	if len(elems) == 0 {
		return false
	}
	for _, e := range elems {
		if e.IsKnown() && !e.IsNull() {
			return false
		}
	}
	return true
}
