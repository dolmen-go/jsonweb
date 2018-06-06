package jsonweb

import (
	"errors"

	"github.com/dolmen-go/jsonptrerror"
)

// Map stores a group of Extractor by names.
type Map struct {
	m map[string]*Extractor
}

func (m *Map) UnmarshalJSON(b []byte) error {
	err := jsonptrerror.Unmarshal(b, &m.m)
	if err != nil {
		return err
	}
	if m.m == nil {
		return errors.New("/: empty webmap")
	}
	return nil
}
