package smapping

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
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
	t.Logf("madnestObj %#v\n", madnestObj)
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
		_ = json.Unmarshal(jsbyte, &mapp)
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
			Final       string `json:"final"`
			Destination string
		}
		Lv3 struct {
			Lv3Str string `json:"lv3str"`
			*Last
			Lv3Dummy string
		}
		Lv2 struct {
			Lv2Str string `json:"lv2str"`
			Lv3
			Lv2Dummy string
		}
		Lv1 struct {
			Lv2
			Lv1Str   string `json:"lv1str"`
			Lv1Dummy string
		}
	)

	obj := Lv1{
		Lv1Str:   "level 1 string",
		Lv1Dummy: "baka",
		Lv2: Lv2{
			Lv2Dummy: "bakabaka",
			Lv2Str:   "level 2 string",
			Lv3: Lv3{
				Lv3Dummy: "bakabakka",
				Lv3Str:   "level 3 string",
				Last: &Last{
					Final:       "destination",
					Destination: "overloop",
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

func ExampleSQLScan_allFields() {
	currtime := time.Now()
	dr := createDummyRow(currtime)
	result := dummyValues{}
	if err := SQLScan(dr, &result, ""); err != nil {
		fmt.Println("Error happened!")
		return
	}
	fmt.Printf("NullString is Valid? %t\n", result.NullString.Valid)
	fmt.Printf("result.NullString is %s\n", result.NullString.String)
	fmt.Printf("NullTime is Valid? %t\n", result.NullTime.Valid)
	fmt.Printf("result.NullTime.Time.Equal(dr.Values.NullTime.Time)? %t\n",
		result.NullTime.Time.Equal(dr.Values.NullTime.Time))
	fmt.Printf("result.Uint64 == %d\n", result.Uint64)

	// output:
	// NullString is Valid? true
	// result.NullString is hello 異世界
	// NullTime is Valid? true
	// result.NullTime.Time.Equal(dr.Values.NullTime.Time)? true
	// result.Uint64 == 5
}

func notin(s string, pool ...string) bool {
	for _, sp := range pool {
		if s == sp {
			return false
		}
	}
	return true
}

func compareErrorReports(t *testing.T, msgfmt string, msgs, errval, errfield []string) {
	for _, msg := range msgs {
		reader := strings.NewReader(msg)
		var (
			v1, v2, v3, v4 string
			field          string
		)
		n, err := fmt.Fscanf(reader, msgfmt, &v1, &v2, &v3, &v4, &field)
		v4 = v4[:len(v4)-1]
		field = field[:len(field)-1]
		if n != 5 {
			t.Errorf("Scanned values should 5 but got %d", n)
			continue
		}
		if err != nil {
			t.Errorf(err.Error())
			continue
		}
		value := strings.Join([]string{v1, v2, v3, v4}, " ")
		if notin(value, errval...) {
			t.Errorf("value '%s' not found", value)
		}
		if notin(field, errfield...) {
			t.Errorf("field '%s' not found", field)
		}
	}
}

func TestBetterErrorReporting(t *testing.T) {
	type SomeStruct struct {
		Field1 int      `errtag:"fieldint"`
		Field2 bool     `errtag:"fieldbol"`
		Field3 string   `errtag:"fieldstr"`
		Field4 float64  `errtag:"fieldflo"`
		Field5 struct{} `errtag:"fieldsru"`
	}
	field1 := "this should be int"
	field2 := "this should be boolean"
	field3 := "this is succesfully converted"
	field4 := "this should be float64"
	field5 := "this should be struct"
	ssmap := Mapped{
		"Field1": field1,
		"Field2": field2,
		"Field3": field3,
		"Field4": field4,
		"Field5": field5,
	}
	ss := SomeStruct{}
	err := FillStruct(&ss, ssmap)
	if err == nil {
		t.Errorf("Error should not nil")
	}
	if ss.Field3 != field3 {
		t.Errorf("ss.Field3 expected '%s' but got '%s'", field3, ss.Field3)
	}
	errmsg := err.Error()
	msgs := strings.Split(errmsg, ",")
	if len(msgs) == 0 {
		t.Errorf("Error message should report more than one field, got 0 report")
	}
	msgfmt := "provided value (%s %s %s %s type not match object field '%s type"
	errval := []string{field1, field2, field4, field5}
	errfield := []string{"Field1", "Field2", "Field4", "Field5"}
	compareErrorReports(t, msgfmt, msgs, errval, errfield)

	ssmaptag := Mapped{
		"fieldint": field1,
		"fieldbol": field2,
		"fieldstr": field3,
		"fieldflo": field4,
		"fieldsru": field5,
	}
	ss = SomeStruct{}
	err = FillStructByTags(&ss, ssmaptag, "errtag")
	if err == nil {
		t.Errorf("Error should not nil")
	}
	if ss.Field3 != field3 {
		t.Errorf("ss.Field3 expected '%s' but got '%s'", field3, ss.Field3)
	}
	errmsg = err.Error()
	msgs = strings.Split(errmsg, ",")
	if len(msgs) == 0 {
		t.Errorf("Error message should report more than one field, got 0 report")
	}
	msgfmt = "provided value (%s %s %s %s type not match field tag 'errtag' of tagname '%s from object"
	errfield = []string{"fieldint", "fieldbol", "fieldflo", "fieldsru"}
	compareErrorReports(t, msgfmt, msgs, errval, errfield)
}

type (
	embedObj struct {
		FieldInt   int     `json:"fieldInt"`
		FieldStr   string  `json:"fieldStr"`
		FieldFloat float64 `json:"fieldFloat"`
	}
	embedEmbed struct {
		Embed1 embedObj  `json:"embed1"`
		Embed2 *embedObj `json:"embed2"`
	}
	embedObjs struct {
		Objs []*embedObj `json:"embeds"`
	}
)

func TestNilValue(t *testing.T) {
	obj := embedEmbed{
		Embed1: embedObj{1, "one", 1.1},
	}
	objmap := MapTags(&obj, "json")
	embed2 := embedEmbed{}
	if err := FillStructByTags(&embed2, objmap, "json"); err != nil {
		t.Errorf("objmap fill fail: %v", err)
	}
	if embed2.Embed2 != nil {
		t.Errorf("Invalid nil conversion, value should be nil")
	}

	objmap = MapFields(&obj)
	embed2 = embedEmbed{}
	if err := FillStruct(&embed2, objmap); err != nil {
		t.Errorf("objmap fields fill fail: %v", err)
	}
	if embed2.Embed2 != nil {
		t.Errorf("Invalid nil conversion, value should be nil")
	}

	objsem := embedObjs{
		Objs: []*embedObj{
			{1, "one", 1.1},
			{2, "two", 2.2},
			nil,
			{4, "four", 3.3},
			{5, "five", 4.4},
		},
	}
	objsmap := MapTags(&objsem, "json")
	fillobjsem := embedObjs{}
	if err := FillStructByTags(&fillobjsem, objsmap, "json"); err != nil {
		t.Errorf("Should not fail: %v", err)
	}
	for i, obj := range fillobjsem.Objs {
		if obj == nil && i != 2 {
			t.Errorf("index %d of object value %v should not nil", i, obj)
		} else if i == 2 && obj != nil {
			t.Errorf("index 3 of object value %v should be nil", obj)
		}
	}

	objsmap = MapFields(&objsem)
	fillobjsem = embedObjs{}
	if err := FillStruct(&fillobjsem, objsmap); err != nil {
		t.Errorf("Should not fail: %v", err)
	}
	for i, obj := range fillobjsem.Objs {
		if obj == nil && i != 2 {
			t.Errorf("index %d of object value %v should not nil", i, obj)
		} else if i == 2 && obj != nil {
			t.Errorf("index 3 of object value %v should be nil", obj)
		}
	}

}

func eq(a, b *embedObj) bool {
	if a == nil || b == nil {
		return false
	}
	return a.FieldFloat == b.FieldFloat && a.FieldInt == b.FieldInt &&
		a.FieldStr == b.FieldStr
}

func arrobj(t *testing.T) {
	objsem := embedObjs{
		Objs: []*embedObj{
			{1, "one", 1.1},
			{2, "two", 2.2},
			nil,
			{4, "four", 3.3},
			{5, "five", 4.4},
		},
	}
	maptag := MapTags(&objsem, "json")

	embedstf, ok := maptag["embeds"].([]interface{})
	if !ok {
		t.Fatalf("Wrong type, %#v", maptag["embeds"])
	}
	if len(embedstf) != len(objsem.Objs) {
		t.Fatalf("len(embedstf) expected %d got %d\n", len(objsem.Objs), len(embedstf))
	}
	for i, emtf := range embedstf {
		if i == 2 && emtf != nil {
			t.Errorf("%v expected nil, got empty value\n", emtf)
			continue
		}
		if i == 2 {
			continue

		}
		emtfmap, ok := emtf.(Mapped)
		// emtf2, ok := emtf.(*embedObj)
		if !ok {
			t.Errorf("Cannot cast to Mapped %#v\n", emtf)
			continue
		}
		emtf2 := &embedObj{}
		if err := FillStructByTags(emtf2, emtfmap, "json"); err != nil {
			t.Error(err)
		}
		if !eq(emtf2, objsem.Objs[i]) && i != 2 {
			t.Errorf("embedObj (%#v) at index %d got wrong value, expect (%#v)",
				emtf2, i, objsem.Objs[i])
		}
	}

	// raw of mapped case
	rawtfobj := Mapped{
		"embeds": []Mapped{
			{"fieldInt": 1, "fieldStr": "one", "fieldFloat": 1.1},
			{"fieldInt": 2, "fieldStr": "two", "fieldFloat": 2.2},
			nil,
			{"fieldInt": 4, "fieldStr": "four", "fieldFloat": 4.4},
			{"fieldInt": 5, "fieldStr": "five", "fieldFloat": 5.5},
		},
	}
	expectedVals := []*embedObj{
		{1, "one", 1.1},
		{2, "two", 2.2},
		nil,
		{4, "four", 4.4},
		{5, "five", 5.5},
	}

	testit := func(raw Mapped) {
		newemb := embedObjs{}
		err := FillStructByTags(&newemb, rawtfobj, "json")
		if err != nil {
			t.Error(err)
		}
		t.Logf("%#v\n", newemb)
		newemblen := len(newemb.Objs)
		exptlen := len(expectedVals)
		if newemblen != exptlen {
			t.Fatalf("New len got %d, expected %d", newemblen, exptlen)
		}
		for i, ob := range newemb.Objs {
			if i == 2 && ob != nil {
				t.Errorf("%v expected nil, got empty value\n", ob)
				continue
			}
			if i != 2 && !eq(ob, expectedVals[i]) {
				t.Errorf("embedObj (%#v) at index %d got wrong value, expect (%#v)",
					ob, i, expectedVals[i])

			}
		}

	}
	testit(rawtfobj)

	// case of actual object
	rawtfobj = Mapped{
		"embeds": []*embedObj{
			{1, "one", 1.1},
			{2, "two", 2.2},
			nil,
			{4, "four", 4.4},
			{5, "five", 5.5},
		},
	}
	testit(rawtfobj)
}

func arrvalues(t *testing.T) {
	type (
		ArrInt   []int
		MyArrInt struct {
			ArrInt `json:"array_int"`
		}
		APint []*int
		MPint struct {
			APint `json:"ptarr_int"`
		}
		APfloat []*float32
		MPfloat struct {
			APfloat `json:"ptarr_float"`
		}
	)

	initobj := MyArrInt{
		ArrInt: []int{1, 2, 3, 4, 5},
	}
	minitobj := MapTags(&initobj, "json")
	arintminit, ok := minitobj["array_int"].([]interface{})
	if !ok {
		t.Errorf("failed to cast %#v\n", minitobj["array_int"])
		return
	}
	t.Logf("arrintminit %#v\n", arintminit)
	rawminit := Mapped{
		"array_int": []int{5, 4, 3, 2, 1},
	}
	var rinit MyArrInt
	if err := FillStructByTags(&rinit, rawminit, "json"); err != nil {
		t.Error(err)
	}
	t.Logf("rinit %#v\n", rinit)

	a := new(int)
	b := new(int)
	c := new(int)
	d := new(int)
	e := new(int)
	*a = 11
	*b = 22
	*c = 33
	*d = 44
	*e = 55
	pinitobj := MPint{
		APint: []*int{a, b, nil, c, d, e},
	}
	mapinit := MapTags(&pinitobj, "json")
	rawpinit, ok := mapinit["ptarr_int"].([]interface{})
	if !ok {
		t.Errorf("failed conv %#v\n", mapinit["ptrarr_int"])
	}

	t.Logf("rawpinit %#v\n", rawpinit)
	for i, rp := range rawpinit {
		if i == 2 && rp != nil {
			t.Errorf("rp should be nil, %#v\n", rp)
		}
		if i == 2 {
			continue
		}
		p, ok := rp.(int)
		if !ok {
			t.Errorf("failed cast, got %#v %T\n", rp, rp)

		}
		ptrop := pinitobj.APint[i]
		if ptrop != nil && i != 2 && *ptrop != p {
			t.Errorf("Wrong value at index %d, got %#v expected %#v",
				i, p, *ptrop)
		}
	}

	rawpinit2 := Mapped{
		"ptarr_int": []interface{}{55, 44, nil, 33, 22, 11},
	}
	var pinit2 MPint
	if err := FillStructByTags(&pinit2, rawpinit2, "json"); err != nil {
		t.Error(err)
	}
	expt2 := []*int{e, d, nil, c, b, a}
	t.Logf("pinit2 %#v\n", pinit2)
	for i, rp := range pinit2.APint {
		if i == 2 && rp != nil {
			t.Errorf("rp should be nil, %#v\n", rp)
		}
		if i == 2 {
			continue
		}

		if !ok {
			t.Errorf("failed cast, got %#v %T\n", rp, rp)

		}
		ptrop := expt2[i]
		if ptrop != nil && i != 2 && *ptrop != *rp {
			t.Errorf("Wrong value at index %d, got %#v expected %#v",
				i, *rp, *ptrop)
		}
	}

	rawfloat := Mapped{
		"ptarr_float": []interface{}{1.1, 2.2, nil, 3.3, 4.4},
	}
	var mfloat MPfloat
	if err := FillStructByTags(&mfloat, rawfloat, "json"); err != nil {
		t.Error(err)
	}
	testelemfloat := func(ap APfloat, expected []interface{}) {
		vallen := len(ap)
		expectlen := len(expected)
		if vallen != expectlen {
			t.Fatalf("Got %d length, expected %d length", vallen, expectlen)
		}
		for i, rp := range ap {
			if i == 2 && rp != nil {
				t.Errorf("rp should be nil, got %v %#v\n", *rp, rp)
			} else {
				if expected[i] == nil {
					continue
				}
				ptrop := expected[i].(float64)
				if float32(ptrop) != *rp {
					t.Errorf("Wrong value at index %d, got %#v expected %#v",
						i, *rp, ptrop)
				}
			}
		}
	}
	testelemfloat(mfloat.APfloat, rawfloat["ptarr_float"].([]interface{}))
}

func TestTagsSlice(t *testing.T) {
	arrobj(t)
	arrvalues(t)
}
