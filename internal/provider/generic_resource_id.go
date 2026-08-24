package provider

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/rixlhq/terraform-provider-clickup/internal/provider/clickupcommon"
)

func (r *genericResource) extractID(raw []byte) (string, error) {
	v, err := clickupcommon.DecodeJSONResponse(raw)
	if err != nil {
		return "", err
	}

	data, ok := v.(map[string]any)
	if !ok {
		return "", errors.New("create response is not a JSON object")
	}

	if id, ok := data["id"]; ok {
		return valueToIDString(id), nil
	}

	// Some ClickUp create responses (e.g. Goal) wrap the object under a key
	// matching the resource name.
	roots := []string{"data", "goal", "task", "list", "space", "view", "key_result"}
	if r.createResponseRoot != "" {
		// Check the explicit root first, then the common ones.
		roots = append([]string{r.createResponseRoot}, roots...)
	}
	for _, key := range roots {
		if d, ok := data[key].(map[string]any); ok {
			// If createResponseItemArray is set, extract ID from the last
			// element of the array at that key (e.g. checklist.items[-1].id).
			if r.createResponseItemArray != "" {
				if arr, ok := d[r.createResponseItemArray].([]any); ok && len(arr) > 0 {
					if last, ok := arr[len(arr)-1].(map[string]any); ok {
						if id, ok := last["id"]; ok {
							return valueToIDString(id), nil
						}
					}
				}
			}
			if id, ok := d["id"]; ok {
				return valueToIDString(id), nil
			}
		}
	}

	return "", errors.New("create response did not contain an id")
}

// resolveResourceID returns an ID from the API response if possible, then
// falls back to the request body when the API returns an empty or incomplete
// response. This is used for both create and update.
func (r *genericResource) resolveResourceID(raw, body []byte) (string, error) {
	id, err := r.extractID(raw)
	if err == nil {
		return id, nil
	}

	if len(r.idFromBody) == 0 {
		return "", err
	}

	var payload any
	if err2 := json.Unmarshal(body, &payload); err2 != nil {
		return "", err
	}
	v := payload
	for _, key := range r.idFromBody {
		m, ok := v.(map[string]any)
		if !ok {
			return "", err
		}
		v, ok = m[key]
		if !ok {
			return "", err
		}
	}
	return valueToIDString(v), nil
}

func valueToIDString(v any) string {
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	case json.Number:
		return x.String()
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case int:
		return strconv.FormatInt(int64(x), 10)
	case int32:
		return strconv.FormatInt(int64(x), 10)
	case int64:
		return strconv.FormatInt(x, 10)
	}
	return fmt.Sprint(v)
}

// notFoundError is used to signal that a resource was not found during a read.
type notFoundError struct{ message string }

func (e *notFoundError) Error() string { return e.message }
