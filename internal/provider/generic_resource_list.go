package provider

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

func (r *genericResource) listItems(jv any) ([]any, error) {
	v := jv
	for key := range strings.SplitSeq(r.readListRoot, ".") {
		m, ok := v.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("list response did not contain %q", r.readListRoot)
		}
		v, ok = m[key]
		if !ok {
			return nil, fmt.Errorf("list response did not contain %q", r.readListRoot)
		}
	}

	items, ok := v.([]any)
	if !ok {
		return nil, fmt.Errorf("list response did not contain %q array", r.readListRoot)
	}
	return items, nil
}

func (r *genericResource) findInList(jv any, id string) (any, error) {
	if _, ok := jv.(map[string]any); !ok {
		return nil, errors.New("list response is not a JSON object")
	}

	items, err := r.listItems(jv)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		v, ok := obj[r.readListIDField]
		if !ok {
			continue
		}
		if valueToIDString(v) == id {
			return item, nil
		}
	}
	return nil, &notFoundError{message: fmt.Sprintf("%s with %s=%q not found in list", r.name, r.readListIDField, id)}
}

func (r *genericResource) nextListPageQuery(jv any, base url.Values, seen map[string]bool) (url.Values, bool) {
	root, ok := jv.(map[string]any)
	if !ok {
		return nil, false
	}
	if lp, ok := root["last_page"].(bool); ok && lp {
		return nil, false
	}

	items, err := r.listItems(jv)
	if err != nil || len(items) == 0 {
		return nil, false
	}

	last, ok := items[len(items)-1].(map[string]any)
	if !ok {
		return nil, false
	}

	id, ok := last[r.readListIDField]
	if !ok {
		return nil, false
	}
	idStr := valueToIDString(id)
	if idStr == "" || seen[idStr] {
		return nil, false
	}
	seen[idStr] = true

	date, ok := last["date"]
	if !ok || date == nil {
		return nil, false
	}
	start := valueToIDString(date)
	if start == "" {
		return nil, false
	}

	next := cloneURLValues(base)
	next.Set("start_id", idStr)
	next.Set("start", start)
	return next, true
}

func cloneURLValues(v url.Values) url.Values {
	if v == nil {
		return url.Values{}
	}
	out := make(url.Values, len(v))
	for k, vv := range v {
		out[k] = append([]string(nil), vv...)
	}
	return out
}
