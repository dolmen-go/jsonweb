package jsonweb

import (
	"errors"
	"fmt"
	"sort"

	"github.com/dolmen-go/jsonptr"
)

type Context struct {
	parent *Context
	values map[string]interface{}
}

func (ctx *Context) Values() map[string]interface{} {
	if ctx == nil {
		return nil
	}
	if ctx.values == nil { // This should not happen
		return ctx.parent.Values()
	}
	values := make(map[string]interface{}, len(ctx.values))
	for k, v := range ctx.values {
		values[k] = v
	}
	ctx = ctx.parent
	for ctx != nil {
		for k, v := range ctx.values {
			if _, exists := values[k]; exists {
				continue
			}
			values[k] = v
		}
		ctx = ctx.parent
	}
	return values
}

type ContextVisitorError struct {
	Err error
}

func (e ContextVisitorError) Error() string {
	return e.Err.Error()
}

// ContextVisitor is a callback that visits all contexts of a JSON-style document.
// The pointer is reused between calls, so it must be .Copy()ed if you need to keep it
// beyond the duraction of the visit call.
type ContextVisitor func(ptr jsonptr.Pointer, ctx *Context) error

type extractor interface {
	Parse(parent *Context, ptr jsonptr.Pointer, doc interface{}, visit ContextVisitor) error
	collectVariables(vars map[string]struct{})
	Variables() []string
}

type objExtractor struct {
	values      map[string]string // key: JSON pointer for extraction; value: variable name
	subcontexts map[string]extractor
}

func (ex *objExtractor) collectVariables(vars map[string]struct{}) {
	if ex == nil {
		return
	}
	for _, v := range ex.values {
		vars[v] = struct{}{}
	}
	// Recurse in subcontexts
	for _, ex := range ex.subcontexts {
		ex.collectVariables(vars)
	}
}

func (ex *objExtractor) Variables() []string {
	if ex == nil {
		return nil
	}
	vars := make(map[string]struct{})
	ex.collectVariables(vars)
	if len(vars) == 0 {
		return nil
	}
	names := make([]string, 0, len(vars))
	for v := range vars {
		names = append(names, v)
	}
	sort.Strings(names)
	return names
}

func (ex *objExtractor) Parse(parent *Context, ptr jsonptr.Pointer, doc interface{}, visit ContextVisitor) error {
	if ex == nil {
		return nil
	}
	// If we are explicitely targetting a "" sub-context and the value is an array, we want to
	// browse the array.
	if ex.values == nil && len(ex.subcontexts) == 1 {
		if _, ok := ex.subcontexts[""]; ok {
			var err error
			doc, err = jsonptr.Get(doc, "") // Call Get to force any necessary unmarshalling
			if err != nil {
				return err
			}
			if arr, ok := doc.([]interface{}); ok {
				if arr == nil {
					return nil
				}
				ptr.Grow(1)
				for i, v := range arr {
					ptr.Index(i)
					if err := ex.Parse(parent, ptr, v, visit); err != nil {
						return err
					}
					ptr.Pop()
				}
				return nil
			}
		}
	}
	var ctx *Context
	for p, name := range ex.values {
		v, err := jsonptr.Get(doc, p)
		if err != nil {
			continue
		}
		if ctx == nil {
			ctx = &Context{parent: parent, values: make(map[string]interface{})}
		}
		ctx.values[name] = v
	}
	if ctx != nil {
		if err := visit(ptr, ctx); err != nil {
			return ContextVisitorError{err}
		}
	} else {
		ctx = parent
	}
	for p, ex := range ex.subcontexts {
		subptr, _ := jsonptr.Parse(p)
		v, err := subptr.In(doc)
		if err != nil {
			continue
		}
		if err := ex.Parse(ctx, append(ptr, subptr...), v, visit); err != nil {
			return err
		}
	}
	return nil
}

type arrExtractor struct {
	extractor
}

func (ex arrExtractor) Parse(parent *Context, ptr jsonptr.Pointer, doc interface{}, visit ContextVisitor) error {
	var err error
	doc, err = jsonptr.Get(doc, "") // Call Get to force any necessary unmarshalling
	if err != nil {
		return err
	}
	switch doc := doc.(type) {
	case nil:
		return nil
	case []interface{}:
		if doc == nil {
			return nil
		}
		ptr.Grow(1)
		for i, v := range doc {
			ptr.Index(i)
			if err := ex.extractor.Parse(parent, ptr, v, visit); err != nil {
				return err
			}
			ptr.Pop()
		}
		return nil
	case map[string]interface{}:
		if doc == nil {
			return nil
		}
		ptr.Grow(1)
		for k, v := range doc {
			ptr.Property(k)
			if err := ex.extractor.Parse(parent, ptr, v, visit); err != nil {
				return err
			}
			ptr.Pop()
		}
		return nil
	default:
		// Not an iterable value => just ignore
		return nil
	}
}

type ExtractorError struct {
	Ptr string
	Err error
}

func (e *ExtractorError) Error() string {
	//return fmt.Sprintf("%q: %v", e.Ptr, e.Err)
	return fmt.Sprintf("%#v", e)
}

func buildExtractor(ptr string, def interface{}) (extractor, *ExtractorError) {
	switch def := def.(type) {
	case nil:
		return nil, nil
	case string:
		if len(def) == 0 {
			return nil, &ExtractorError{ptr, errors.New(`invalid value ""`)}
		}
		// TODO check that def matches a variable name for URI template https://tools.ietf.org/html/rfc6570#section-2.3
		return &objExtractor{values: map[string]string{"": def}}, nil
	case map[string]interface{}:
		if len(def) == 0 {
			return nil, nil
		}
		var values map[string]string
		var subextractors map[string]extractor
		for k, v := range def {
			_, err := jsonptr.Parse(k)
			if err != nil {
				return nil, &ExtractorError{ptr, fmt.Errorf("invalid key %q: JSON pointer expected", k)}
			}
			switch v := v.(type) {
			case nil: // ignore
			case string:
				// TODO check that def matches a variable name for URI template https://tools.ietf.org/html/rfc6570#section-2.3
				if values == nil {
					values = make(map[string]string)
				}
				values[k] = v
			default:
				ex, err := buildExtractor(ptr+"/"+jsonptr.EscapeString(k), v)
				if err != nil {
					return nil, err
				}
				if ex != nil {
					if subextractors == nil {
						subextractors = make(map[string]extractor)
					}
					subextractors[k] = ex
				}
			}
		}
		return &objExtractor{values, subextractors}, nil
	case []interface{}:
		if len(def) != 1 {
			return nil, &ExtractorError{ptr, errors.New("element expected in array iterator")}
		}
		ex, err := buildExtractor(ptr+"/0", def[0])
		if err != nil {
			return nil, err
		}
		return arrExtractor{ex}, nil
	default:
		return nil, &ExtractorError{ptr, errors.New("invalid value type")}
	}
}
