package smapping

import (
	"encoding/json"
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

func printIfNotExists(mapped Mapped, keys ...string) {
	for _, key := range keys {
		if _, ok := mapped[key]; !ok {
			fmt.Println(key, ": not exists")
		}
	}
}

func ExampleMapFields() {
	mapped := MapFields(&sourceobj)
	printIfNotExists(mapped, "Label", "Info", "Version")
	// Output:
}

func ExampleMapTags_basic() {
	maptags := MapTags(&sourceobj, "json")
	printIfNotExists(maptags, "label", "info", "version")
	// Output:
}

type generalFields struct {
	Name     string `json:"name" api:"general_name"`
	Rank     string `json:"rank" api:"general_rank"`
	Code     int    `json:"code" api:"general_code"`
	nickname string // won't be mapped because not exported
}

func ExampleMapTags_twoTags() {

	general := generalFields{
		Name:     "duran",
		Rank:     "private",
		Code:     1337,
		nickname: "drone",
	}
	mapjson := MapTags(&general, "json")
	printIfNotExists(mapjson, "name", "rank", "code")

	mapapi := MapTags(&general, "api")
	printIfNotExists(mapapi, "general_name", "general_rank", "general_code")

	// Output:
}

func ExampleMapTagsWithDefault() {
	type higherCommon struct {
		General     generalFields `json:"general"`
		Communality string        `json:"common"`
		Available   bool          `json:"available" api:"is_available"`
	}
	rawjson := []byte(`{
	    "general": {
		name:     "duran",
		rank:     "private",
		code:     1337,
	    },
	    "common": "rare",
	    "available": true
	}`)
	hc := higherCommon{}
	_ = json.Unmarshal(rawjson, &hc)
	maptags := MapTagsWithDefault(&hc, "api", "json")
	printIfNotExists(maptags, "available")
	// Output: available : not exists
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
