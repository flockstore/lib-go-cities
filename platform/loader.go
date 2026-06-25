package platform

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// LoadFile loads a matcher from a JSON file path.
func LoadFile(path string) (*Matcher, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open cities source: %w", err)
	}
	defer file.Close()

	return LoadReader(path, file)
}

// LoadReader loads a matcher from a JSON reader.
func LoadReader(source string, reader io.Reader) (*Matcher, error) {
	var records []City
	if err := json.NewDecoder(reader).Decode(&records); err != nil {
		return nil, fmt.Errorf("decode cities source: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("cities source is empty")
	}

	return NewMatcher(source, records), nil
}
