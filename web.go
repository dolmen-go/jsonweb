package jsonweb

import (
	"fmt"
	"sort"
	"strings"

	"github.com/dolmen-go/jsonptr"
	"github.com/jtacoma/uritemplates"
)

type templated struct {
	uri       *uritemplates.UriTemplate
	extractor extractor
}

type Web struct {
	roots     map[string]extractor
	templates map[string]map[string]*templated // URL templates grouped by groups of variables
}

type BuildError struct {
	Ptr string
	Err error
}

func (e *BuildError) Error() string {
	return fmt.Sprintf("%q: %v", e.Ptr, e.Err)
}

func New(m map[string]interface{}) (*Web, error) {
	var web Web
	for k, v := range m {
		ptr := "/" + jsonptr.EscapeString(k)
		u, err := uritemplates.Parse(k)
		if err != nil {
			return nil, &BuildError{ptr, err}
		}
		ex, err := buildExtractor(ptr, v)
		if err != (*ExtractorError)(nil) { // This cast is necessary
			return nil, err
		}
		vars := u.Names()
		if len(vars) == 0 {
			if web.roots == nil {
				web.roots = make(map[string]extractor)
			}
			web.roots[k] = ex
		} else {
			if web.templates == nil {
				web.templates = make(map[string]map[string]*templated)
			}
			sort.Strings(vars)
			key := strings.Join(vars, ",")
			g := web.templates[key]
			if g == nil {
				g = make(map[string]*templated)
				web.templates[key] = g
			}
			g[k] = &templated{u, ex}
		}
	}
	return &web, nil
}

func (web *Web) Roots() []string {
	var roots []string
	for r := range web.roots {
		roots = append(roots, r)
	}
	return roots
}

func (web *Web) WithVars(names []string) []*uritemplates.UriTemplate {
	if len(names) == 0 { // Just for consistency
		roots := web.Roots()
		if roots == nil {
			return nil
		}
		res := make([]*uritemplates.UriTemplate, 0, len(roots))
		for _, u := range web.Roots() {
			tmpl, _ := uritemplates.Parse(u)
			res = append(res, tmpl)
		}
		return res
	}
	if !sort.StringsAreSorted(names) {
		// Clone
		names = append([]string(nil), names...)
		sort.Strings(names)
	}
	key := strings.Join(names, ",")
	g := web.templates[key]
	if g == nil {
		return nil
	}
	res := make([]*uritemplates.UriTemplate, 0, len(g))
	for _, v := range g {
		res = append(res, v.uri)
	}
	return res
}

func (web *Web) Parse(url string, doc interface{}, visit ContextVisitor) error {
	return web.roots[url].Parse((*Context)(nil), jsonptr.Pointer(nil), doc, visit)
}
