package platform

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// LoadFile loads a matcher from a JSON file path.
func LoadFile(path string) (*Matcher, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read cities source: %w", err)
	}

	return LoadBytes(path, data)
}

// LoadBytes loads a matcher from JSON bytes.
func LoadBytes(source string, data []byte) (*Matcher, error) {
	return load(source, bytes.NewReader(data))
}

// LoadReader loads a matcher from a JSON reader.
func LoadReader(source string, reader io.Reader) (*Matcher, error) {
	return load(source, reader)
}

func load(source string, reader io.Reader) (*Matcher, error) {
	var records []City
	if err := json.NewDecoder(reader).Decode(&records); err != nil {
		return nil, fmt.Errorf("decode cities source: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("cities source is empty")
	}

	return NewMatcher(source, records), nil
}
