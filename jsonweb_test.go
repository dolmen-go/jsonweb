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
	var websrc map[string]interface{}
	dec := json.NewDecoder(r)
	if err = dec.Decode(&websrc); err != nil {
		panic(err)
	}
	r.Close()

	web, err := jsonweb.New(websrc)
	if err != nil {
		panic(err)
	}
	r, err = os.Open("testdata/wikipedia-0.json")
	if err != nil {
		panic(err)
	}

	err = web.Parse(
		`https://en.wikipedia.org/w/api.php?action=query&titles=Main%20Page&prop=revisions|info&rvprop=user&rvlimit=10&format=json&formatversion=2`,
		json.NewDecoder(r),
		func(ptr jsonptr.Pointer, ctx *jsonweb.Context) error {
			fmt.Println(ptr, ctx.Values())
			return nil
		},
	)
	if err != nil {
		panic(err)
	}

	// Output:
}
