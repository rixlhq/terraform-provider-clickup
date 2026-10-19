package clickupclient

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// APIError is returned when the ClickUp API responds with a non-2xx status code.
type APIError struct {
	StatusCode int
	Body       string
}

// Error formats the APIError including status code and parsed body details.
func (e *APIError) Error() string {
	formatted := formatErrorBody(e.Body)
	if formatted != "" {
		return fmt.Sprintf("clickup API error %d: %s", e.StatusCode, formatted)
	}
	return fmt.Sprintf("clickup API error %d: %s", e.StatusCode, e.Body)
}

type rawAPIError struct {
	Err     string          `json:"err"`
	Error   string          `json:"error"`
	Message string          `json:"message"`
	ECODE   string          `json:"ECODE"`
	Code    string          `json:"code"`
	TraceID string          `json:"trace_id"`
	Errors  json.RawMessage `json:"errors"`
}

type fieldError struct {
	Field   string `json:"field"`
	Param   string `json:"param"`
	Key     string `json:"key"`
	Message string `json:"message"`
	Msg     string `json:"msg"`
	Err     string `json:"err"`
}

// extractPrimaryMessage returns the primary human-readable error message from rawAPIError.
func extractPrimaryMessage(raw *rawAPIError) string {
	if raw.Message != "" {
		return raw.Message
	}
	if raw.Err != "" {
		return raw.Err
	}
	return raw.Error
}

// extractFieldError formats a single field-level error into a descriptive message.
func extractFieldError(fe fieldError) string {
	f := fe.Field
	if f == "" {
		f = fe.Param
	}
	if f == "" {
		f = fe.Key
	}
	m := fe.Message
	if m == "" {
		m = fe.Msg
	}
	if m == "" {
		m = fe.Err
	}

	switch {
	case f != "" && m != "":
		return fmt.Sprintf("field %q: %s", f, m)
	case m != "":
		return m
	case f != "":
		return fmt.Sprintf("field %q", f)
	default:
		return ""
	}
}

// extractDetailParts unmarshals and formats any nested or field-level validation errors.
func extractDetailParts(errorsRaw json.RawMessage) []string {
	if len(errorsRaw) == 0 {
		return nil
	}

	var fieldErrors []fieldError
	if err := json.Unmarshal(errorsRaw, &fieldErrors); err == nil && len(fieldErrors) > 0 {
		var details []string
		for _, fe := range fieldErrors {
			if s := extractFieldError(fe); s != "" {
				details = append(details, s)
			}
		}
		return details
	}

	var strErrors []string
	if err := json.Unmarshal(errorsRaw, &strErrors); err == nil && len(strErrors) > 0 {
		return strErrors
	}

	return nil
}

// formatErrorBody parses a raw JSON error response and returns a formatted error summary.
func formatErrorBody(body string) string {
	trimmed := strings.TrimSpace(body)
	if !strings.HasPrefix(trimmed, "{") {
		return body
	}

	var raw rawAPIError
	if err := json.Unmarshal([]byte(trimmed), &raw); err != nil {
		return body
	}

	primaryMsg := extractPrimaryMessage(&raw)
	detailParts := extractDetailParts(raw.Errors)

	var b strings.Builder
	if primaryMsg != "" {
		b.WriteString(primaryMsg)
	}

	if len(detailParts) > 0 {
		if b.Len() > 0 {
			b.WriteString(" (")
			b.WriteString(strings.Join(detailParts, "; "))
			b.WriteString(")")
		} else {
			b.WriteString(strings.Join(detailParts, "; "))
		}
	}

	code := raw.ECODE
	if code == "" {
		code = raw.Code
	}
	if code != "" {
		if b.Len() > 0 {
			b.WriteString(" [code: ")
			b.WriteString(code)
			b.WriteString("]")
		} else {
			b.WriteString("[code: ")
			b.WriteString(code)
			b.WriteString("]")
		}
	}

	if raw.TraceID != "" {
		if b.Len() > 0 {
			b.WriteString(" [trace_id: ")
			b.WriteString(raw.TraceID)
			b.WriteString("]")
		} else {
			b.WriteString("[trace_id: ")
			b.WriteString(raw.TraceID)
			b.WriteString("]")
		}
	}

	if b.Len() == 0 {
		return body
	}
	return b.String()
}

// IsNotFound reports whether err is an APIError with HTTP 404 status.
func IsNotFound(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound
}
