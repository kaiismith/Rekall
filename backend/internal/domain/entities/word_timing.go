package entities

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// WordTiming is one entry in a transcript segment's per-word timing array.
// JSON tags match the ASR proto (asr/proto/asr.proto WordTiming) exactly so
// the JSONB stored in transcript_segments.words is a faithful round-trip of
// what the ASR service emitted.
type WordTiming struct {
	Word        string  `json:"w"`
	StartMs     uint32  `json:"start_ms"`
	EndMs       uint32  `json:"end_ms"`
	Probability float32 `json:"p"`
}

// WordTimings is a JSONB-backed slice of WordTiming. Implements driver.Valuer
// and sql.Scanner so GORM marshals to/from the JSONB column transparently.
type WordTimings []WordTiming

// Value serialises the slice to JSON for storage. A nil slice maps to SQL NULL.
func (w WordTimings) Value() (driver.Value, error) {
	if w == nil {
		return nil, nil
	}
	return json.Marshal(w)
}

// Scan deserialises a JSON value from the database into the slice.
// NULL maps to a nil slice.
func (w *WordTimings) Scan(value interface{}) error {
	if value == nil {
		*w = nil
		return nil
	}
	var raw []byte
	switch v := value.(type) {
	case []byte:
		raw = v
	case string:
		raw = []byte(v)
	default:
		return fmt.Errorf("WordTimings.Scan: unsupported type %T", value)
	}
	if len(raw) == 0 {
		*w = nil
		return nil
	}
	return json.Unmarshal(raw, w)
}
