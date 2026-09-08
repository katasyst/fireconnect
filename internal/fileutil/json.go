package fileutil

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

// ReadJSONFile reads JSON from path into v. Returns an empty object when the file does not exist.
func ReadJSONFile(path string, v interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if m, ok := v.(*map[string]interface{}); ok && *m == nil {
				*m = map[string]interface{}{}
			}
			return nil
		}
		return err
	}
	if len(data) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("%s is not valid JSON: %w", path, err)
	}
	return nil
}

// WriteJSONFile writes v as indented JSON using an atomic write.
func WriteJSONFile(path string, v interface{}, mode os.FileMode) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return WriteFileAtomic(path, data, mode)
}

// ReadJSONMap reads a JSON object file as a generic map.
func ReadJSONMap(path string) (map[string]interface{}, error) {
	out := map[string]interface{}{}
	if err := ReadJSONFile(path, &out); err != nil {
		return nil, err
	}
	return out, nil
}
