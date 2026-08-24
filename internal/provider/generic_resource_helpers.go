package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	resource_schema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/numberplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/rixlhq/terraform-provider-clickup/internal/provider/clickupcommon"
)

func (r *genericResource) idFromValue(v tftypes.Value) (string, diag.Diagnostics) {
	var diags diag.Diagnostics
	obj, err := asObject(v)
	if err != nil {
		diags.AddError("Invalid State", err.Error())
		return "", diags
	}
	param := r.idParam()
	paramVal, ok := obj[param]
	if !ok {
		diags.AddError("Missing Resource ID", fmt.Sprintf("%q is required", param))
		return "", diags
	}
	s, err := valueAsString(paramVal)
	if err != nil {
		diags.AddError("Invalid Resource ID", err.Error())
		return "", diags
	}
	return s, diags
}

func (r *genericResource) idParam() string {
	if r.idField != "" {
		return r.idField
	}
	for _, path := range []string{r.deletePath, r.updatePath, r.readPath} {
		for _, match := range pathParamRegex.FindAllStringSubmatch(path, -1) {
			return match[1]
		}
	}
	return "id"
}

func (r *genericResource) buildPath(path string, v tftypes.Value) (string, diag.Diagnostics) {
	var diags diag.Diagnostics
	obj, err := asObject(v)
	if err != nil {
		diags.AddError("Invalid Value", err.Error())
		return "", diags
	}

	out := path
	for _, match := range pathParamRegex.FindAllStringSubmatch(path, -1) {
		param := match[1]
		normParam := clickupcommon.TerraformIdentifier(param)
		paramVal, ok := obj[normParam]
		if !ok {
			diags.AddError("Missing Path Parameter", fmt.Sprintf("%q is required", normParam))
			return "", diags
		}
		s, err := valueAsString(paramVal)
		if err != nil {
			diags.AddError("Invalid Path Parameter", fmt.Sprintf("%q: %s", normParam, err))
			return "", diags
		}
		out = strings.Replace(out, match[0], url.PathEscape(s), 1)
	}

	return out, diags
}

func (r *genericResource) buildBody(ctx context.Context, v tftypes.Value, fields []string, defaults map[string]any, transforms map[string]func(any) any) ([]byte, diag.Diagnostics) {
	var diags diag.Diagnostics
	j, err := clickupcommon.TfValueToJSON(ctx, v)
	if err != nil {
		diags.AddError("Request Body Error", err.Error())
		return nil, diags
	}

	all, ok := j.(map[string]any)
	if !ok {
		diags.AddError("Request Body Error", "plan value is not an object")
		return nil, diags
	}

	body := make(map[string]any, len(fields))
	for _, f := range fields {
		val, ok := all[f]
		if !ok || val == nil {
			if d, ok := defaults[f]; ok {
				val = d
			}
		}
		if val == nil {
			// Field is missing and has no default; omit it from the body.
			continue
		}
		if transform, ok := transforms[f]; ok && transform != nil {
			val = transform(val)
		}
		body[f] = val
	}

	raw, err := json.Marshal(body)
	if err != nil {
		diags.AddError("Request Body Error", err.Error())
		return nil, diags
	}
	return raw, diags
}

// buildUpdateBody creates the request body for an update. It uses the merged
// state+plan value for known fields, but omits any update body field that is
// unknown or null in the plan. This prevents sending unchanged computed values
// to incremental update endpoints while still preserving nested computed
// sub-attributes (e.g. the "rem" side of an add/rem object) when a field is
// changed.
func (r *genericResource) buildUpdateBody(ctx context.Context, merged, plan tftypes.Value, fields []string, defaults map[string]any, transforms map[string]func(any) any) ([]byte, diag.Diagnostics) {
	var diags diag.Diagnostics
	bodyValue, err := r.updateBodyValue(ctx, merged, plan, fields)
	if err != nil {
		diags.AddError("Update Body Error", err.Error())
		return nil, diags
	}
	return r.buildBody(ctx, bodyValue, fields, defaults, transforms)
}

// updateBodyValue returns a tftypes.Value that contains the merged value for
// every update body field that is known in the plan, and null for update body
// fields that are unknown or null in the plan. Non-body attributes are left as
// they are in the merged value.
func (r *genericResource) updateBodyValue(_ context.Context, merged, plan tftypes.Value, fields []string) (tftypes.Value, error) {
	if !merged.IsKnown() || merged.IsNull() {
		return merged, nil
	}

	typ, ok := merged.Type().(tftypes.Object)
	if !ok {
		return merged, nil
	}

	mergedObj, err := asObject(merged)
	if err != nil {
		return merged, err
	}

	planObj := map[string]tftypes.Value{}
	if plan.IsKnown() && !plan.IsNull() {
		if m, err := asObject(plan); err == nil {
			planObj = m
		}
	}

	fieldSet := make(map[string]bool, len(fields))
	for _, f := range fields {
		fieldSet[f] = true
	}

	vals := make(map[string]tftypes.Value, len(typ.AttributeTypes))
	for attr, attrType := range typ.AttributeTypes {
		planVal, ok := planObj[attr]
		if !ok {
			planVal = tftypes.NewValue(attrType, nil)
		}
		mergedVal, ok := mergedObj[attr]
		if !ok {
			mergedVal = tftypes.NewValue(attrType, nil)
		}

		if fieldSet[attr] && (!planVal.IsKnown() || planVal.IsNull()) {
			// The practitioner did not configure this field for the update.
			vals[attr] = tftypes.NewValue(attrType, nil)
		} else {
			vals[attr] = mergedVal
		}
	}

	return tftypes.NewValue(merged.Type(), vals), nil
}

func (r *genericResource) tfType(ctx context.Context) tftypes.Type {
	return r.schema(ctx).Type().TerraformType(ctx)
}

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

func (r *genericResource) addMissingPathParamAttributes(s resource_schema.Schema) resource_schema.Schema {
	if r.createPath == "" || s.Attributes == nil {
		return s
	}
	for _, match := range pathParamRegex.FindAllStringSubmatch(r.createPath, -1) {
		param := clickupcommon.TerraformIdentifier(match[1])
		if attr, ok := s.Attributes[param]; ok {
			// The attribute already exists but it identifies the collection where
			// the object is created, so changes must trigger replacement.
			s.Attributes[param] = withRequiresReplace(attr)
			continue
		}
		// Default to a Required StringAttribute so the path parameter accepts
		// numeric and non-numeric IDs and is converted to a string when building
		// the API path.
		s.Attributes[param] = resource_schema.StringAttribute{
			Required:            true,
			MarkdownDescription: fmt.Sprintf("ClickUp %s path parameter required to create this %s.", param, r.name),
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		}
	}
	return s
}

func withRequiresReplace(attr resource_schema.Attribute) resource_schema.Attribute {
	switch a := attr.(type) {
	case resource_schema.StringAttribute:
		a.PlanModifiers = append(a.PlanModifiers, stringplanmodifier.RequiresReplace())
		return a
	case resource_schema.Int64Attribute:
		a.PlanModifiers = append(a.PlanModifiers, int64planmodifier.RequiresReplace())
		return a
	case resource_schema.NumberAttribute:
		a.PlanModifiers = append(a.PlanModifiers, numberplanmodifier.RequiresReplace())
		return a
	}
	return attr
}

func (r *genericResource) withPathParam(ctx context.Context, v tftypes.Value, param, id string) (tftypes.Value, diag.Diagnostics) {
	var diags diag.Diagnostics
	obj, err := asObject(v)
	if err != nil {
		diags.AddError("Invalid Value", err.Error())
		return tftypes.Value{}, diags
	}

	fullType, ok := r.tfType(ctx).(tftypes.Object)
	if !ok {
		diags.AddError("Invalid Value", "expected object type")
		return tftypes.Value{}, diags
	}

	attrType, ok := fullType.AttributeTypes[param]
	if !ok {
		diags.AddError("Invalid Resource Schema", fmt.Sprintf("path parameter %q is not an attribute", param))
		return tftypes.Value{}, diags
	}

	newVal, err := clickupcommon.JSONToTfValue(ctx, attrType, id)
	if err != nil {
		diags.AddError("ID Conversion Error", err.Error())
		return tftypes.Value{}, diags
	}

	vals := make(map[string]tftypes.Value, len(fullType.AttributeTypes))
	for k, at := range fullType.AttributeTypes {
		if ov, ok := obj[k]; ok {
			vals[k] = ov
		} else {
			vals[k] = tftypes.NewValue(at, tftypes.UnknownValue)
		}
	}
	vals[param] = newVal

	return tftypes.NewValue(fullType, vals), diags
}

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
