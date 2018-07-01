package smapping

import (
	"fmt"
)

type source struct {
	Label   string `json:"label"`
	Info    string `json:"info"`
	Version int    `json:"version"`
}

type sink struct {
	Label string
	Info  string
}

type differentSink struct {
	DiffLabel string `json:"label"`
	NiceInfo  string `json:"info"`
	Version   string `json:"unversion"`
}

var sourceobj source = source{
	Label:   "source",
	Info:    "the origin",
	Version: 1,
}

func ExampleMapFields() {
	mapped := MapFields(&sourceobj)
	fmt.Println(mapped)
	// Output: map[Label:source Info:the origin Version:1]
}

func ExampleMapTags_basic() {
	maptags := MapTags(&sourceobj, "json")
	fmt.Println(maptags)
	// Output: map[label:source info:the origin version:1]
}

func ExampleMapTags_twoTags() {
	type generalFields struct {
		Name     string `json:"name" api:"general_name"`
		Rank     string `json:"rank" api:"general_rank"`
		Code     int    `json:"code" api:"general_code"`
		nickname string // won't be mapped because not exported
	}

	general := generalFields{
		Name:     "duran",
		Rank:     "private",
		Code:     1337,
		nickname: "drone",
	}
	mapjson := MapTags(&general, "json")
	fmt.Println(mapjson)

	mapapi := MapTags(&general, "api")
	fmt.Println(mapapi)

	// Output:
	// map[name:duran rank:private code:1337]
	// map[general_name:duran general_rank:private general_code:1337]
}

func ExampleFillStruct() {
	mapped := MapFields(&sourceobj)
	sinked := sink{}
	err := FillStruct(&sinked, mapped)
	if err != nil {
		panic(err)
	}
	fmt.Println(sinked)
	// Output: {source the origin}
}

func ExampleFillStructByTags() {
	maptags := MapTags(&sourceobj, "json")
	diffsink := differentSink{}
	err := FillStructByTags(&diffsink, maptags, "json")
	if err != nil {
		panic(err)
	}
	fmt.Println(diffsink)
	// Output: {source the origin }
}
