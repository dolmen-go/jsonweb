package jsonweb

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/dolmen-go/jsonptr"
	"github.com/jtacoma/uritemplates"
)

type templated struct {
	raw string
	uri *uritemplates.UriTemplate
}

type Browser struct {
	m         Map
	roots     map[string]struct{}
	templates map[string]map[string]*uritemplates.UriTemplate // URI templates grouped by groups of variables
}

type BuildError struct {
	Ptr string
	Err error
}

func (e *BuildError) Error() string {
	return fmt.Sprintf("%q: %v", e.Ptr, e.Err)
}

func NewBrowser(m Map) (*Browser, error) {
	browser := Browser{m: m}
	for k := range m {
		u, err := uritemplates.Parse(k)
		if err != nil {
			return nil, &BuildError{"/" + jsonptr.EscapeString(k), err}
		}
		vars := u.Names()
		if len(vars) == 0 {
			if browser.roots == nil {
				browser.roots = make(map[string]struct{})
			}
			browser.roots[k] = struct{}{}
		} else {
			if browser.templates == nil {
				browser.templates = make(map[string]map[string]*uritemplates.UriTemplate)
			}
			sort.Strings(vars)
			key := strings.Join(vars, ",")
			g := browser.templates[key]
			if g == nil {
				g = make(map[string]*uritemplates.UriTemplate)
				browser.templates[key] = g
			}
			g[key] = u
		}
	}
	return &browser, nil
}

func (browser *Browser) UnmarshalJSON(b []byte) error {
	var m Map
	if err := m.UnmarshalJSON(b); err != nil {
		return err
	}
	newBrowser, err := NewBrowser(m)
	if err != nil {
		return err
	}
	*browser = *newBrowser
	return nil
}

func (browser *Browser) MarshalJSON() ([]byte, error) {
	return json.Marshal(browser.m)
}

func (browser *Browser) Roots() []string {
	var roots []string
	for r := range browser.roots {
		roots = append(roots, r)
	}
	return roots
}

func (browser *Browser) WithVars(names []string) []*uritemplates.UriTemplate {
	if len(names) == 0 { // Just for consistency
		if len(browser.roots) == 0 {
			return nil
		}
		res := make([]*uritemplates.UriTemplate, 0, len(browser.roots))
		for u := range browser.roots {
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
	g := browser.templates[key]
	if g == nil {
		return nil
	}
	res := make([]*uritemplates.UriTemplate, 0, len(g))
	for _, v := range g {
		res = append(res, v)
	}
	return res
}

func (browser *Browser) Parse(url string, doc interface{}, visit ContextVisitor) error {
	return browser.m.Parse(url, doc, visit)
}
