package clickupcommon

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

var (
	lowerToUpperReg     = regexp.MustCompile(`([a-z])[A-Z]`)
	unsupportedCharsReg = regexp.MustCompile(`[^a-zA-Z0-9_]+`)
	leadingNumbersReg   = regexp.MustCompile(`^(\d+)`)
)

// TerraformIdentifier converts a name to the snake_case form used by the Terraform code generator.
func TerraformIdentifier(original string) string {
	if len(original) == 0 {
		return original
	}
	removed := unsupportedCharsReg.ReplaceAllString(original, "")
	noLeading := leadingNumbersReg.ReplaceAllString(removed, "")
	inserted := lowerToUpperReg.ReplaceAllStringFunc(noLeading, func(s string) string {
		firstRune, size := utf8.DecodeRuneInString(s)
		return fmt.Sprintf("%s_%s", string(firstRune), strings.ToLower(s[size:]))
	})
	return strings.ToLower(inserted)
}

// DecodeJSONResponse decodes JSON using json.Number to preserve numeric precision.
func DecodeJSONResponse(body []byte) (any, error) {
	var v any
	dec := json.NewDecoder(strings.NewReader(string(body)))
	dec.UseNumber()
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	return v, nil
}

// JSONToTfValue converts a JSON-decoded value into a tftypes.Value that matches t.
func JSONToTfValue(ctx context.Context, t tftypes.Type, v any) (tftypes.Value, error) {
	return jsonToTfValue(ctx, t, v)
}

// TfValueToJSON converts a tftypes.Value into a Go value that can be JSON
// marshaled. Object attribute names are left as-is (snake_case). Unknown and
// null values are omitted so they are not sent in request bodies. Numbers are
// returned as json.Number to preserve precision.
func TfValueToJSON(ctx context.Context, v tftypes.Value) (any, error) {
	return tfValueToJSON(ctx, v)
}
