[![license](https://img.shields.io/github/license/mashape/apistatus.svg?style=plastic)](./LICENSE)
[![CircleCI](https://circleci.com/gh/mashingan/smapping.svg?style=svg)](https://circleci.com/gh/mashingan/smapping)
[![GoDoc](https://godoc.org/github.com/mashingan/smapping?status.svg)](https://godoc.org/github.com/mashingan/smapping)

# smapping
Golang structs generic mapping.

## Motivation
Working with between ``struct``, and ``json`` with **Golang** has various
degree of difficulty.
The thing that makes difficult is that sometimes we get arbitrary ``json``
or have to make json with arbitrary fields.  
In **Golang** we can achieve that using ``interface{}`` that act as ``"Any"``
value.
While it seems good, however ``interface{}`` only gives us ability to *disable* 
type propagation and various reasoning.
It's a trade-off that one has to make to enable interfacing with dynamically
defined such as ``json``.

## Install
```
go get github.com/mashingan/smapping
```

## Example
```go
package main

import (
	"encoding/json"
	"fmt"

	"github.com/mashingan/smapping"
)

type Source struct {
	Label   string `json:"label"`
	Info    string `json:"info"`
	Version int    `json:"version"`
}

type Sink struct {
	Label string
	Info  string
}

type HereticSink struct {
	NahLabel string `json:"label"`
	HahaInfo string `json:"info"`
	Version  string `json:"heretic_version"`
}

type DifferentOneField struct {
	Name    string `json:"name"`
	Label   string `json:"label"`
	Code    string `json:"code"`
	Private string `json:"private" api:"internal"`
}

func main() {
	source := Source{
		Label:   "source",
		Info:    "the origin",
		Version: 1,
	}
	fmt.Println("source:", source)
	mapped := smapping.MapFields(&source)
	fmt.Println("mapped:", mapped)
	sink := Sink{}
	err := smapping.FillStruct(&sink, mapped)
	if err != nil {
		panic(err)
	}
	fmt.Println("sink:", sink)

	maptags := smapping.MapTags(&source, "json")
	fmt.Println("maptags:", maptags)
	hereticsink := HereticSink{}
	err = smapping.FillStructByTags(&hereticsink, maptags, "json")
	if err != nil {
		panic(err)
	}
	fmt.Println("heretic sink:", hereticsink)

	fmt.Println("=============")
	recvjson := []byte(`{"name": "bella", "label": "balle", "code": "albel", "private": "allbe"}`)
	dof := DifferentOneField{}
	_ = json.Unmarshal(recvjson, &dof)
	fmt.Println("unmarshaled struct:", dof)

	marshaljson, _ := json.Marshal(dof)
	fmt.Println("marshal back:", string(marshaljson))

	// What we want actually "internal" instead of "private" field
	// we use the api tags on to make the json
	apijson, _ := json.Marshal(smapping.MapTagsWithDefault(&dof, "api", "json"))
	fmt.Println("api marshal:", string(apijson))

	fmt.Println("=============")
	// This time is the reverse, we receive "internal" field when
	// we need to receive "private" field to match our json tag field
	respjson := []byte(`{"name": "bella", "label": "balle", "code": "albel", "internal": "allbe"}`)
	respdof := DifferentOneField{}
	_ = json.Unmarshal(respjson, &respdof)
	fmt.Println("unmarshal resp:", respdof)

	// to get that, we should put convert the json to Mapped first
	jsonmapped := smapping.Mapped{}
	_ = json.Unmarshal(respjson, &jsonmapped)
	// now we fill our struct respdof
	_ = smapping.FillStructByTags(&respdof, jsonmapped, "api")
	fmt.Println("full resp:", respdof)
	returnback, _ := json.Marshal(respdof)
	fmt.Println("marshal resp back:", string(returnback))
	// first we unmarshal respdof, we didn't get the "private" field
	// but after our mapping, we get "internal" field value and
	// simply marshaling back to `returnback`
}
```
## LICENSE
MIT
