package smapping

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

type source struct {
	Label   string    `json:"label"`
	Info    string    `json:"info"`
	Version int       `json:"version"`
	Toki    time.Time `json:"tomare"`
}

type sink struct {
	Label string
	Info  string
}

type differentSink struct {
	DiffLabel string    `json:"label"`
	NiceInfo  string    `json:"info"`
	Version   string    `json:"unversion"`
	Toki      time.Time `json:"doki"`
}

type differentSourceSink struct {
	Source   source        `json:"source"`
	DiffSink differentSink `json:"differentSink"`
}

var toki = time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)
var sourceobj source = source{
	Label:   "source",
	Info:    "the origin",
	Version: 1,
	Toki:    toki,
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
			Toki:      toki,
		},
	}
	nestedMap := MapTags(&nestedSource, "json")
	for k, v := range nestedMap {
		fmt.Println("top key:", k)
		for kk, vv := range v.(Mapped) {
			if vtime, ok := vv.(time.Time); ok {
				fmt.Println("    nested:", kk, vtime.Format(time.RFC3339))
			} else {

				fmt.Println("    nested:", kk, vv)
			}
		}
		fmt.Println()
	}
	// Unordered Output:
	// top key: source
	//     nested: label source
	//     nested: info the origin
	//     nested: version 1
	//     nested: tomare 2000-01-01T00:00:00Z
	//
	// top key: differentSink
	//     nested: label nested diff
	//     nested: info nested info
	//     nested: unversion next version
	//     nested: doki 2000-01-01T00:00:00Z
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
		if vt, ok := v.(time.Time); ok {
			fmt.Printf("maptags[%s]: %s\n", k, vt.Format(time.RFC3339))
		} else {
			fmt.Printf("maptags[%s]: %v\n", k, v)

		}
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
	// maptags[tomare]: 2000-01-01T00:00:00Z
	// {source the origin  0001-01-01 00:00:00 +0000 UTC}
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
		Label   string     `json:"label"`
		Time    time.Time  `json:"definedTime"`
		PtrTime *time.Time `json:"ptrTime"`
	}
	now := time.Now()
	obj := timeMap{Label: "test", Time: now, PtrTime: &now}
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
	if !objTarget.PtrTime.Equal(*(obj.PtrTime)) {
		t.Errorf("Error value pointer time conversion: %s not equal with %s",
			objTarget.PtrTime.Format(time.RFC3339),
			obj.PtrTime.Format(time.RFC3339),
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

func ExampleMapTagsFlatten() {
	type (
		Last struct {
			Final string `json:"final"`
		}
		Lv3 struct {
			Lv3Str string `json:"lv3str"`
			*Last
		}
		Lv2 struct {
			Lv2Str string `json:"lv2str"`
			Lv3
		}
		Lv1 struct {
			Lv2
			Lv1Str string `json:"lv1str"`
		}
	)

	obj := Lv1{
		Lv1Str: "level 1 string",
		Lv2: Lv2{
			Lv2Str: "level 2 string",
			Lv3: Lv3{
				Lv3Str: "level 3 string",
				Last: &Last{
					Final: "destination",
				},
			},
		},
	}

	for k, v := range MapTagsFlatten(&obj, "json") {
		fmt.Printf("key: %s, value: %v\n", k, v)
	}
	// Unordered Output:
	// key: final, value: destination
	// key: lv1str, value: level 1 string
	// key: lv2str, value: level 2 string
	// key: lv3str, value: level 3 string
}

type dummyValues struct {
	Int     int
	Int8    int8
	Int16   int16
	Int32   int32
	Int64   int64
	Uint    uint
	Uint8   uint8
	Uint16  uint16
	Uint32  uint32
	Uint64  uint64
	Float32 float32
	Float64 float64
	Bool    bool
	String  string
	Bytes   []byte
	sql.NullBool
	sql.NullFloat64
	sql.NullInt32
	sql.NullInt64
	sql.NullString
	sql.NullTime
}

type dummyRow struct {
	Values dummyValues
}

func (dr *dummyRow) Scan(dest ...interface{}) error {
	for i, x := range dest {
		switch x.(type) {
		case *int:
			dest[i] = &dr.Values.Int
		case *int8:
			dest[i] = &dr.Values.Int8
		case *int16:
			dest[i] = &dr.Values.Int16
		case *int32:
			dest[i] = &dr.Values.Int32
		case *int64:
			dest[i] = &dr.Values.Int64
		case *uint:
			dest[i] = &dr.Values.Uint
		case *uint8:
			dest[i] = &dr.Values.Uint8
		case *uint16:
			dest[i] = &dr.Values.Uint16
		case *uint32:
			dest[i] = &dr.Values.Uint32
		case *uint64:
			dest[i] = &dr.Values.Uint64
		case *float32:
			dest[i] = &dr.Values.Float32
		case *float64:
			dest[i] = &dr.Values.Float64
		case *string:
			dest[i] = &dr.Values.String
		case *[]byte:
			dest[i] = &dr.Values.Bytes
		case *bool:
			dest[i] = &dr.Values.Bool
		case *sql.NullBool:
			dest[i] = &dr.Values.NullBool
		case *sql.NullFloat64:
			dest[i] = &dr.Values.NullFloat64
		case *sql.NullInt32:
			dest[i] = &dr.Values.NullInt32
		case *sql.NullInt64:
			dest[i] = &dr.Values.NullInt64
		case *sql.NullString:
			dest[i] = &dr.Values.NullString
		case *sql.NullTime:
			dest[i] = &dr.Values.NullTime
		}
	}
	return nil
}

func createDummyRow(destTime time.Time) *dummyRow {
	return &dummyRow{
		Values: dummyValues{
			Int:         -5,
			Int8:        -4,
			Int16:       -3,
			Int32:       -2,
			Int64:       -1,
			Uint:        1,
			Uint8:       2,
			Uint16:      3,
			Uint32:      4,
			Uint64:      5,
			Float32:     42.1,
			Float64:     42.2,
			Bool:        true,
			String:      "hello 異世界",
			Bytes:       []byte("hello 異世界"),
			NullBool:    sql.NullBool{Bool: true, Valid: true},
			NullFloat64: sql.NullFloat64{Float64: 42.2, Valid: true},
			NullInt32:   sql.NullInt32{Int32: 421, Valid: true},
			NullInt64:   sql.NullInt64{Int64: 422, Valid: true},
			NullString:  sql.NullString{String: "hello 異世界", Valid: true},
			NullTime:    sql.NullTime{Time: destTime, Valid: true},
		},
	}
}

func ExampleSQLScan_suppliedFields() {
	currtime := time.Now()
	dr := createDummyRow(currtime)
	result := dummyValues{}
	if err := SQLScan(dr, &result,
		"", /* This is the tag, since we don't have so put it empty
		to match the field name */
		/* Below arguments are variadic and we only take several
		   fields from all available dummyValues */
		"Int32", "Uint64", "Bool", "Bytes",
		"NullString", "NullTime"); err != nil {
		fmt.Println("Error happened!")
		return
	}
	fmt.Printf("NullString is Valid? %t\n", result.NullString.Valid)
	fmt.Printf("NullTime is Valid? %t\n", result.NullTime.Valid)
	fmt.Printf("result.NullTime.Time.Equal(dr.Values.NullTime.Time)? %t\n",
		result.NullTime.Time.Equal(dr.Values.NullTime.Time))
	fmt.Printf("result.Uint64 == %d\n", result.Uint64)

	// output:
	// NullString is Valid? true
	// NullTime is Valid? true
	// result.NullTime.Time.Equal(dr.Values.NullTime.Time)? true
	// result.Uint64 == 5
}
