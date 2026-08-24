package provider

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/rixlhq/terraform-provider-clickup/internal/provider/clickupcommon"
)

func extractWebhookID(raw []byte) (string, error) {
	jv, err := clickupcommon.DecodeJSONResponse(raw)
	if err != nil {
		return "", err
	}
	root, ok := jv.(map[string]any)
	if !ok {
		return "", errors.New("create webhook response is not an object")
	}
	if id, ok := root["id"].(string); ok {
		return id, nil
	}
	if n, ok := root["id"].(json.Number); ok {
		return n.String(), nil
	}
	if webhook, ok := root["webhook"].(map[string]any); ok {
		if id, ok := webhook["id"].(string); ok {
			return id, nil
		}
		if n, ok := webhook["id"].(json.Number); ok {
			return n.String(), nil
		}
	}
	return "", errors.New("create webhook response did not contain an id")
}

func stringOrNull(v any) types.String {
	switch x := v.(type) {
	case string:
		return types.StringValue(x)
	case json.Number:
		return types.StringValue(x.String())
	default:
		return types.StringNull()
	}
}

func int64OrNull(v any) types.Int64 {
	switch x := v.(type) {
	case json.Number:
		i, err := x.Int64()
		if err == nil {
			return types.Int64Value(i)
		}
	case int64:
		return types.Int64Value(x)
	case int:
		return types.Int64Value(int64(x))
	case float64:
		return types.Int64Value(int64(x))
	case string:
		i, err := strconv.ParseInt(x, 10, 64)
		if err == nil {
			return types.Int64Value(i)
		}
	}
	return types.Int64Null()
}

func listValueFromStrings(ctx context.Context, raw []any) types.List {
	if len(raw) == 0 {
		return types.ListNull(types.StringType)
	}
	values := make([]string, 0, len(raw))
	for _, e := range raw {
		switch x := e.(type) {
		case string:
			values = append(values, x)
		case json.Number:
			values = append(values, x.String())
		}
	}
	lv, _ := types.ListValueFrom(ctx, types.StringType, values)
	return lv
}

// parseIntString parses a string that contains an integer ID.
// It returns the original string if it cannot be parsed, allowing the API to
// decide what to do with it.
func parseIntString(s string) any {
	if s == "" {
		return nil
	}
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return s
	}
	return i
}

// jsonOrNull returns the JSON encoding of v if it is a map or slice, otherwise
// falls back to the string representation of v. It is used for the webhook
// health object, which the API returns as an object but the schema exposes as
// a JSON-encoded string.
func jsonOrNull(v any) types.String {
	switch x := v.(type) {
	case string:
		return types.StringValue(x)
	case json.Number:
		return types.StringValue(x.String())
	case nil:
		return types.StringNull()
	default:
		b, err := json.Marshal(v)
		if err == nil {
			return types.StringValue(string(b))
		}
		return types.StringNull()
	}
}
