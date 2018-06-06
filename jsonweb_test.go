package jsonweb_test

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/dolmen-go/jsonptr"
	"github.com/dolmen-go/jsonweb"
)

func Example() {
	r, err := os.Open("testdata/wikipedia-0.web.json")
	if err != nil {
		panic(err)
	}
	dec := json.NewDecoder(r)
	var browser jsonweb.Browser
	if err = dec.Decode(&browser); err != nil {
		panic(err)
	}
	r.Close()

	r, err = os.Open("testdata/wikipedia-0.json")
	if err != nil {
		panic(err)
	}

	err = browser.Parse(
		`https://en.wikipedia.org/w/api.php?action=query&titles=Main%20Page&prop=revisions|info&rvprop=user&rvlimit=10&format=json&formatversion=2`,
		json.NewDecoder(r),
		func(ptr jsonptr.Pointer, ctx *jsonweb.Context) error {
			values, _ := json.Marshal(ctx.Values())
			fmt.Printf("%s: %s\n", ptr, values)
			return nil
		},
	)
	if err != nil {
		panic(err)
	}

	// Output:
	// /query/pages/0: {"title":"Main Page"}
	// /query/pages/0/revisions/0: {"title":"Main Page","user":"The Blade of the Northern Lights"}
	// /query/pages/0/revisions/1: {"title":"Main Page","user":"Bearcat"}
	// /query/pages/0/revisions/2: {"title":"Main Page","user":"Bearcat"}
	// /query/pages/0/revisions/3: {"title":"Main Page","user":"Optimist on the run"}
	// /query/pages/0/revisions/4: {"title":"Main Page","user":"Ian.thomson"}
	// /query/pages/0/revisions/5: {"title":"Main Page","user":"Ian.thomson"}
	// /query/pages/0/revisions/6: {"title":"Main Page","user":"Alex Shih"}
	// /query/pages/0/revisions/7: {"title":"Main Page","user":"TheDJ"}
	// /query/pages/0/revisions/8: {"title":"Main Page","user":"TheDJ"}
	// /query/pages/0/revisions/9: {"title":"Main Page","user":"Doc James"}
}
