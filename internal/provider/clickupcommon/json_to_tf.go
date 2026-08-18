package clickupcommon

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func jsonToTfValue(ctx context.Context, t tftypes.Type, v any) (tftypes.Value, error) {
	if v == nil {
		return tftypes.NewValue(t, nil), nil
	}

	if t.Is(tftypes.String) {
		s, ok := toString(v)
		if !ok {
			return tftypes.Value{}, fmt.Errorf("cannot convert %T to string", v)
		}
		return tftypes.NewValue(tftypes.String, s), nil
	}
	if t.Is(tftypes.Number) {
		n, ok := toBigFloat(v)
		if !ok {
			return tftypes.Value{}, fmt.Errorf("cannot convert %T to number", v)
		}
		return tftypes.NewValue(tftypes.Number, n), nil
	}
	if t.Is(tftypes.Bool) {
		switch b := v.(type) {
		case bool:
			return tftypes.NewValue(tftypes.Bool, b), nil
		case string:
			if b == "true" || b == "1" {
				return tftypes.NewValue(tftypes.Bool, true), nil
			}
			if b == "false" || b == "0" {
				return tftypes.NewValue(tftypes.Bool, false), nil
			}
		}
		return tftypes.Value{}, fmt.Errorf("cannot convert %T to bool", v)
	}
	if t.Is(tftypes.DynamicPseudoType) {
		return dynamicjsonToTfValue(v)
	}

	switch ty := t.(type) {
	case tftypes.Object:
		return jsonToTfObject(ctx, t, ty, v)
	case tftypes.List:
		return jsonToTfList(ctx, t, ty.ElementType, v)
	case tftypes.Set:
		return jsonToTfList(ctx, t, ty.ElementType, v)
	case tftypes.Map:
		return jsonToTfMap(ctx, t, ty, v)
	case tftypes.Tuple:
		return jsonToTfTuple(ctx, t, ty, v)
	default:
		return tftypes.Value{}, fmt.Errorf("unsupported tftypes.Type %T", t)
	}
}

func jsonToTfObject(ctx context.Context, t tftypes.Type, ty tftypes.Object, v any) (tftypes.Value, error) {
	m, ok := v.(map[string]any)
	if !ok {
		return tftypes.NewValue(t, nil), nil
	}
	// The generated Terraform schemas use snake_case attribute names, while
	// the ClickUp API returns camelCase keys. Build a lookup that preserves the
	// original values (including map keys) but normalizes only this object's
	// attribute names.
	normalized := make(map[string]any, len(m))
	for k, val := range m {
		normalized[TerraformIdentifier(k)] = val
	}
	vals := make(map[string]tftypes.Value, len(ty.AttributeTypes))
	for attr, attrType := range ty.AttributeTypes {
		attrVal, ok := normalized[attr]
		if !ok {
			vals[attr] = tftypes.NewValue(attrType, nil)
			continue
		}
		converted, err := jsonToTfValue(ctx, attrType, attrVal)
		if err != nil {
			return tftypes.Value{}, err
		}
		vals[attr] = converted
	}
	return tftypes.NewValue(t, vals), nil
}

func jsonToTfList(ctx context.Context, t tftypes.Type, elemType tftypes.Type, v any) (tftypes.Value, error) {
	s, ok := v.([]any)
	if !ok {
		return tftypes.NewValue(t, nil), nil
	}
	vals := make([]tftypes.Value, 0, len(s))
	for _, elem := range s {
		converted, err := jsonToTfValue(ctx, elemType, elem)
		if err != nil {
			return tftypes.Value{}, err
		}
		vals = append(vals, converted)
	}
	return tftypes.NewValue(t, vals), nil
}

func jsonToTfMap(ctx context.Context, t tftypes.Type, ty tftypes.Map, v any) (tftypes.Value, error) {
	m, ok := v.(map[string]any)
	if !ok {
		return tftypes.NewValue(t, nil), nil
	}
	vals := make(map[string]tftypes.Value, len(m))
	for k, elem := range m {
		converted, err := jsonToTfValue(ctx, ty.ElementType, elem)
		if err != nil {
			return tftypes.Value{}, err
		}
		vals[k] = converted
	}
	return tftypes.NewValue(t, vals), nil
}

func jsonToTfTuple(ctx context.Context, t tftypes.Type, ty tftypes.Tuple, v any) (tftypes.Value, error) {
	s, ok := v.([]any)
	if !ok || len(s) != len(ty.ElementTypes) {
		return tftypes.NewValue(t, nil), nil
	}
	vals := make([]tftypes.Value, 0, len(s))
	for i, elem := range s {
		converted, err := jsonToTfValue(ctx, ty.ElementTypes[i], elem)
		if err != nil {
			return tftypes.Value{}, err
		}
		vals = append(vals, converted)
	}
	return tftypes.NewValue(t, vals), nil
}

func toString(v any) (string, bool) {
	switch val := v.(type) {
	case string:
		return val, true
	case json.Number:
		return val.String(), true
	case nil:
		return "", true
	default:
		// Return false so the caller can treat the value as null instead of
		// coercing an object or array into a string.
		return "", false
	}
}

func toBigFloat(v any) (*big.Float, bool) {
	switch val := v.(type) {
	case json.Number:
		f, _, err := big.NewFloat(0).SetPrec(128).Parse(val.String(), 10)
		if err != nil {
			return nil, false
		}
		return f, true
	case float64:
		return big.NewFloat(val).SetPrec(128), true
	case int:
		return big.NewFloat(float64(val)).SetPrec(128), true
	case int64:
		return big.NewFloat(float64(val)).SetPrec(128), true
	case string:
		f, _, err := big.NewFloat(0).SetPrec(128).Parse(val, 10)
		if err != nil {
			return nil, false
		}
		return f, true
	default:
		return nil, false
	}
}

func dynamicjsonToTfValue(v any) (tftypes.Value, error) {
	switch val := v.(type) {
	case nil:
		return tftypes.NewValue(tftypes.DynamicPseudoType, nil), nil
	case bool:
		return tftypes.NewValue(tftypes.DynamicPseudoType, val), nil
	case json.Number:
		n, ok := toBigFloat(val)
		if !ok {
			return tftypes.NewValue(tftypes.DynamicPseudoType, nil), nil
		}
		return tftypes.NewValue(tftypes.DynamicPseudoType, n), nil
	case float64:
		return tftypes.NewValue(tftypes.DynamicPseudoType, big.NewFloat(val).SetPrec(128)), nil
	case int:
		return tftypes.NewValue(tftypes.DynamicPseudoType, big.NewFloat(float64(val)).SetPrec(128)), nil
	case string:
		return tftypes.NewValue(tftypes.DynamicPseudoType, val), nil
	case []any:
		elems := make([]tftypes.Value, 0, len(val))
		for _, e := range val {
			el, err := dynamicjsonToTfValue(e)
			if err != nil {
				return tftypes.Value{}, err
			}
			elems = append(elems, el)
		}
		return tftypes.NewValue(tftypes.DynamicPseudoType, elems), nil
	case map[string]any:
		vals := make(map[string]tftypes.Value, len(val))
		for k, e := range val {
			el, err := dynamicjsonToTfValue(e)
			if err != nil {
				return tftypes.Value{}, err
			}
			vals[k] = el
		}
		return tftypes.NewValue(tftypes.DynamicPseudoType, vals), nil
	default:
		return tftypes.Value{}, fmt.Errorf("unsupported dynamic value type %T", val)
	}
}
