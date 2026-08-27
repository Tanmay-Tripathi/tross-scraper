package response

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// MarshalJSON writes { profile, <enabled sections…>, meta }. Hand-rolled because
// sections belong at the top level in a stable order, which a Go map will not give.
func (r ProfileResult) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')

	if err := writeField(&buf, "profile", r.Profile, true); err != nil {
		return nil, err
	}

	// Anything unordered is appended, so a new section cannot silently vanish.
	written := make(map[string]bool, len(r.Sections))
	for _, name := range r.SectionOrder {
		value, ok := r.Sections[name]
		if !ok {
			continue
		}
		if err := writeField(&buf, name, value, false); err != nil {
			return nil, err
		}
		written[name] = true
	}
	for name, value := range r.Sections {
		if written[name] {
			continue
		}
		if err := writeField(&buf, name, value, false); err != nil {
			return nil, err
		}
	}

	if err := writeField(&buf, "meta", r.Meta, false); err != nil {
		return nil, err
	}

	buf.WriteByte('}')
	return buf.Bytes(), nil
}

func writeField(buf *bytes.Buffer, name string, value any, first bool) error {
	if !first {
		buf.WriteByte(',')
	}

	key, err := json.Marshal(name)
	if err != nil {
		return fmt.Errorf("marshal key %q: %w", name, err)
	}
	buf.Write(key)
	buf.WriteByte(':')

	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal field %q: %w", name, err)
	}
	buf.Write(encoded)
	return nil
}
