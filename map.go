package jsonweb

import (
	"errors"

	"github.com/dolmen-go/jsonptr"
	"github.com/dolmen-go/jsonptrerror"
)

// Map stores a group of Extractor by names.
type Map map[string]*Extractor

func (m *Map) UnmarshalJSON(b []byte) error {
	var nude map[string]*Extractor
	err := jsonptrerror.Unmarshal(b, &nude)
	if err != nil {
		return err
	}
	if nude == nil {
		return errors.New("/: empty webmap")
	}
	*m = nude
	return nil
}

func (m Map) Parse(key string, doc interface{}, visit ContextVisitor) error {
	return m[key].Parse((*Context)(nil), jsonptr.Pointer(nil), doc, visit)
}
