package models

import (
	"bytes"
	"encoding/json"
)

// Timestamp represents a value that the Assinafy API may serialize either as
// an ISO-8601 string or as a Unix timestamp number. It is decoded into the
// canonical string form and re-encoded as a JSON string.
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
	*t = Timestamp(string(data))
	return nil
}

// MarshalJSON always emits the timestamp as a JSON string.
func (t Timestamp) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(t))
}
