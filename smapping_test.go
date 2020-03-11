package smapping

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
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

type differentSourceSink struct {
	Source   source        `json:"source"`
	DiffSink differentSink `json:"differentSink"`
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

func ExampleMapTags_nested() {
	nestedSource := differentSourceSink{
		Source: sourceobj,
		DiffSink: differentSink{
			DiffLabel: "nested diff",
			NiceInfo:  "nested info",
			Version:   "next version",
		},
	}
	nestedMap := MapTags(&nestedSource, "json")
	for k, v := range nestedMap {
		fmt.Println("top key:", k)
		for kk, vv := range v.(Mapped) {
			fmt.Println("    nested:", kk, vv)
		}
		fmt.Println()
	}
	// Unordered Output:
	// top key: source
	//     nested: label source
	//     nested: info the origin
	//     nested: version 1
	//
	// top key: differentSink
	//     nested: label nested diff
	//     nested: info nested info
	//     nested: unversion next version
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
	for k, v := range maptags {
		fmt.Printf("maptags[%s]: %v\n", k, v)
	}
	diffsink := differentSink{}
	err := FillStructByTags(&diffsink, maptags, "json")
	if err != nil {
		panic(err)
	}
	fmt.Println(diffsink)
	// Unordered Output:
	// maptags[label]: source
	// maptags[info]: the origin
	// maptags[version]: 1
	// {source the origin }
}

type RefLevel3 struct {
	What string `json:"finally"`
}
type Level2 struct {
	*RefLevel3 `json:"ref_level3"`
}
type Level1 struct {
	Level2 `json:"level2"`
}
type TopLayer struct {
	Level1 `json:"level1"`
}
type MadNest struct {
	TopLayer `json:"top"`
}

var madnestStruct MadNest = MadNest{
	TopLayer: TopLayer{
		Level1: Level1{
			Level2: Level2{
				RefLevel3: &RefLevel3{
					What: "matryoska",
				},
			},
		},
	},
}

func TestMapTags_nested(t *testing.T) {
	madnestMap := MapTags(&madnestStruct, "json")
	if len(madnestMap) != 1 {
		t.Errorf("Got empty Mapped, expected 1")
		return
	}
	top, ok := madnestMap["top"]
	if !ok {
		t.Errorf("Failed to get top field")
		return
	}
	lv1, ok := top.(Mapped)["level1"]
	if !ok {
		t.Errorf("Failed to get level 1 field")
		return
	}
	lv2, ok := lv1.(Mapped)["level2"]
	if !ok {
		t.Errorf("Failed to get level 2 field")
		return
	}
	reflv3, ok := lv2.(Mapped)["ref_level3"]
	if !ok {
		t.Errorf("Failed to get ref level 3 field")
		return
	}
	what, ok := reflv3.(Mapped)["finally"]
	if !ok {
		t.Errorf("Failed to get the inner ref level 3")
		return
	}
	switch v := what.(type) {
	case string:
		theval := what.(string)
		if theval != "matryoska" {
			t.Errorf("Expected matryoska, got %s", theval)
		}
	default:
		t.Errorf("Expected string, got %T", v)
	}
}

func FillStructNestedTest(bytag bool, t *testing.T) {
	var madnestObj MadNest
	var err error
	if bytag {
		madnestMap := MapTags(&madnestStruct, "json")
		err = FillStructByTags(&madnestObj, madnestMap, "json")
	} else {
		madnestMap := MapFields(&madnestStruct)
		err = FillStruct(&madnestObj, madnestMap)
	}
	if err != nil {
		t.Errorf("%s", err.Error())
		return
	}
	if madnestObj.TopLayer.Level1.Level2.RefLevel3.What != "matryoska" {
		t.Errorf("Error: expected \"matroska\" got \"%s\"", madnestObj.Level1.Level2.RefLevel3.What)
	}
}

func TestFillStructByTags_nested(t *testing.T) {
	FillStructNestedTest(true, t)
}

func TestFillStruct_nested(t *testing.T) {
	FillStructNestedTest(false, t)
}

func fillStructTime(bytag bool, t *testing.T) {
	type timeMap struct {
		Label string    `json:"label"`
		Time  time.Time `json:"definedTime"`
	}
	now := time.Now()
	obj := timeMap{Label: "test", Time: now}
	objTarget := timeMap{}
	if bytag {
		jsbyte, err := json.Marshal(obj)
		if err != nil {
			t.Error(err)
			return
		}
		mapp := Mapped{}
		err = json.Unmarshal(jsbyte, &mapp)
		err = FillStructByTags(&objTarget, mapp, "json")
		if err != nil {
			t.Error(err)
			return
		}
	} else {
		mapfield := MapFields(&obj)
		jsbyte, err := json.Marshal(mapfield)
		if err != nil {
			t.Error(err)
			return
		}
		mapp := Mapped{}
		err = json.Unmarshal(jsbyte, &mapp)
		if err != nil {
			t.Error(err)
			return
		}
		err = FillStruct(&objTarget, mapp)
		if err != nil {
			t.Error(err)
			return
		}
	}
	if !objTarget.Time.Equal(obj.Time) {
		t.Errorf("Error value conversion: %s not equal with %s",
			objTarget.Time.Format(time.RFC3339),
			obj.Time.Format(time.RFC3339),
		)
		return
	}
}
func TestFillStructByTags_time_conversion(t *testing.T) {
	fillStructTime(true, t)
}

func TestFillStruct_time_conversion(t *testing.T) {
	fillStructTime(false, t)
}
