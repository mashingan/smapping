// Copyright (c) 2018 Rahmatullah
// This library is licensed with MIT license which can be found
// in LICENSE

/*
Package smapping is Library for collecting various operations on struct and its mapping
to interface{} and/or map[string]interface{} type.
Implemented to ease the conversion between Golang struct and json format
together with ease of mapping selections using different part of field tagging.

The implementation is abstraction on top reflection package, reflect.

Examples

The snippet code below will be used accross example for brevity

    type source struct {
	Label   string `json:"label"`
	Info    string `json:"info"`
	Version int    `json:"version"`
    }

    type sink struct {
	Label string	// note that we don't include struct tag
	Info  string
    }

    type differentSink struct {
	DiffLabel string `json:"label"`	    // note that this struct
	NiceInfo  string `json:"info"`	    // has different field name
	Version   string `json:"unversion"` // but same json tag
    }

    var sourceobj source = source{
	Label:   "source",
	Info:    "the origin",
	Version: 1,
    }

    func printIfNotExists(mapped Mapped, keys ...string) {
	for _, key := range keys {
	    if _, ok := mapped[key]; !ok {
		fmt.Println(false)
	    }
	}
    }
*/
package smapping
