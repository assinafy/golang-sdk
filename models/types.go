package models

import (
	"bytes"
	"encoding/json"
	"errors"
	"strconv"
)

// Timestamp represents a value that the Assinafy API may serialize either as
// an ISO-8601 string or as a Unix timestamp number. It is decoded into the
// canonical string form. Numeric values are re-encoded as JSON numbers; other
// values are encoded as strings.
type Timestamp string

// String returns the underlying value.
func (t Timestamp) String() string { return string(t) }

// IsZero reports whether the timestamp is empty.
func (t Timestamp) IsZero() bool { return t == "" }

// UnmarshalJSON accepts either a JSON string or number.
func (t *Timestamp) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		*t = ""
		return nil
	}
	if data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		*t = Timestamp(s)
		return nil
	}
	var number json.Number
	if err := json.Unmarshal(data, &number); err != nil {
		return errors.New("assinafy: timestamp must be a JSON string, number, or null")
	}
	*t = Timestamp(number.String())
	return nil
}

// MarshalJSON preserves canonical integer timestamps as JSON numbers and emits
// date-time values as JSON strings.
func (t Timestamp) MarshalJSON() ([]byte, error) {
	value := string(t)
	if number, err := strconv.ParseInt(value, 10, 64); err == nil && strconv.FormatInt(number, 10) == value {
		return []byte(value), nil
	}
	return json.Marshal(string(t))
}

// Payload is a forgiving map[string]any that accepts the empty-array form
// (`[]`) the Assinafy API emits when an activity or webhook dispatch carries
// no parameters. JSON null and an empty array decode to a nil map.
type Payload map[string]any

// UnmarshalJSON decodes an object as a map, and `null` or an empty array as
// a nil map. A non-empty JSON array is treated as a decoding error so
// surprising upstream changes surface instead of being silently dropped.
func (p *Payload) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		*p = nil
		return nil
	}
	if trimmed[0] == '[' {
		var arr []json.RawMessage
		if err := json.Unmarshal(trimmed, &arr); err != nil {
			return err
		}
		if len(arr) != 0 {
			return errors.New("assinafy: payload array form must be empty")
		}
		*p = nil
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal(trimmed, &m); err != nil {
		return err
	}
	*p = m
	return nil
}

// MarshalJSON emits the payload as a JSON object, or `null` when the map is nil.
func (p Payload) MarshalJSON() ([]byte, error) {
	if p == nil {
		return []byte("null"), nil
	}
	return json.Marshal(map[string]any(p))
}
