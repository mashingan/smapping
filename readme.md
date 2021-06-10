[![license](https://img.shields.io/github/license/mashape/apistatus.svg?style=plastic)](./LICENSE)
[![CircleCI](https://circleci.com/gh/mashingan/smapping.svg?style=svg)](https://circleci.com/gh/mashingan/smapping)
[![GoDoc](https://godoc.org/github.com/mashingan/smapping?status.svg)](https://godoc.org/github.com/mashingan/smapping)
[![Go Report Card](https://goreportcard.com/badge/github.com/mashingan/smapping)](https://goreportcard.com/report/github.com/mashingan/smapping)

# smapping
Golang structs generic mapping.

### Version Limit
To support nesting object conversion, the lowest Golang version supported is `1.12.0`.  
To support `smapping.SQLScan`, the lowest Golang version supported is `1.13.0`.

# Table of Contents
1. [Motivation At Glimpse](#at-glimpse).
2. [Motivation Length](#motivation).
3. [Install](#install).
4. [Examples](#examples).
	* [Basic usage examples](#basic-usage-examples)
	* [Nested object example](#nested-object-example)
	* [SQLScan usage example](#sqlscan-usage-example)
5. [License](#license).

# At Glimpse
## What?
A library to provide a mapped structure generically/dynamically.

## Who?
Anyone who has to work with large structure.

## Why?
Scalability and Flexibility.

## When?
At the runtime.

## Where?
In users code.

## How?
By converting into `smapping.Mapped` which alias for `map[string]interface{}`,
users can iterate the struct arbitarily with `reflect` package.

# Motivation
Working with between ``struct``, and ``json`` with **Golang** has various
degree of difficulty.
The thing that makes difficult is that sometimes we get arbitrary ``json``
or have to make json with arbitrary fields. Sometime we also need to have
a different field names, extracting specific fields, working with same
structure with different domain fields name etc.

In order to answer those flexibility, we map the object struct to the
more general data structure as table/map.

Table/Map is the data structure which ubiquitous after list,
which in turn table/map can be represented as
list of pair values (In Golang we can't have it because there's no tuple
data type, tuple is limited as return values).

Object can be represented as table/map dynamically just like in
JavaScript/EcmaScript which object is behaving like table and in
Lua with its metatable. By some extent we can represent the JSON
as table too.

In this library, we provide the mechanism to smoothly map the object
representation back-and-forth without having the boilerplate of
type-checking one by one by hand. Type-checking by hand is certainly
*seems* easier when the domain set is small, but it soon becomes
unbearable as the structure and/or architecure dynamically changed
because of newer insight and information. Hence in [`Who section`](#who)
mentioned this library is for anyone who has to work with large
domain set.

Except for type `smapping.Mapped` as alias, we don't provide others
type struct currently as each operation doesn't need to keep the
internal state so each operation is transparent and *almost* functional
(*almost functional* because we modify the struct fields values instead of
returning the new struct itself, but this is only trade-off because Golang
doesn't have type-parameter which known as generic).

Since `v0.1.10`, we added the [`MapEncoder`](smapping.go#L21) and
[`MapDecoder`](smapping.go#L27) interfaces for users to have custom conversion
for custom and self-defined struct.

## Install
```
go get github.com/mashingan/smapping
```

## Examples

### Basic usage examples
Below example are basic representation how we can work with `smapping`.
Several examples are converged into single runnable example for the ease
of reusing the same structure definition and its various tags.
Refer this example to get a glimpse of how to do things. Afterward,
users can creatively use to accomplish what they're wanting to do
with the provided flexibility.

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

### Nested object example
This example illustrates how we map back-and-forth even with deep
nested object structure. The ability to map nested objects is to
creatively change its representation whether to flatten all tagged
field name even though the inner struct representation is nested.
Regardless of the usage (`whether to flatten the representation`) or
just simply fetching and remapping into different domain name set,
the ability to map the nested object is necessary.

```go

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

func main() {
	// since we're targeting the same MadNest, both of functions will yield
	// same result hence this unified example/test.
	var madnestObj MadNest
	var err error
	testByTags := true
	if testByTags {
		madnestMap := smapping.MapTags(&madnestStruct, "json")
		err = smapping.FillStructByTags(&madnestObj, madnestMap, "json")
	} else {
		madnestMap := smapping.MapFields(&madnestStruct)
		err = smapping.FillStruct(&madnestObj)
	}
	if err != nil {
		fmt.Printf("%s", err.Error())
		return
	}
	// the result should yield as intented value.
	if madnestObj.TopLayer.Level1.Level2.RefLevel3.What != "matryoska" {
		fmt.Printf("Error: expected \"matroska\" got \"%s\"", madnestObj.Level1.Level2.RefLevel3.What)
	}
}
```

### SQLScan usage example
This example, we're using `sqlite3` as the database, we add a convenience
feature for any struct/type that implements `Scan` method as `smapping.SQLScanner`.
Keep in mind this is quite different with `sql.Scanner` that's also requiring
the type/struct to implement `Scan` method. The difference here, `smapping.SQLScanner`
receiving variable arguments of `interface{}` as values' placeholder while `sql.Scanner`
is only receive a single `interface{}` argument as source. `smapping.SQLScan` is
working for `Scan` literally after we've gotten the `*sql.Row` or `*sql.Rows`.

```go
package main

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/mashingan/smapping"
	_ "github.com/mattn/go-sqlite3"
)

type book struct {
	Author author `json:"author"`
}

type author struct {
	Num  int            `json:"num"`
	ID   sql.NullString `json:"id"`
	Name sql.NullString `json:"name"`
}

func (a author) MarshalJSON() ([]byte, error) {
	mapres := map[string]interface{}{}
	if !a.ID.Valid {
		//if a.ID == nil || !a.ID.Valid {
		mapres["id"] = nil
	} else {
		mapres["id"] = a.ID.String
	}
	//if a.Name == nil || !a.Name.Valid {
	if !a.Name.Valid {
		mapres["name"] = nil
	} else {
		mapres["name"] = a.Name.String
	}
	mapres["num"] = a.Num
	return json.Marshal(mapres)
}

func getAuthor(db *sql.DB, id string) author {
	res := author{}
	err := db.QueryRow("select * from author where id = ?", id).
		Scan(&res.Num, &res.ID, &res.Name)
	if err != nil {
		panic(err)
	}
	return res
}

func getAuthor12(db *sql.DB, id string) author {
	result := author{}
	fields := []string{"num", "id", "name"}
	err := smapping.SQLScan(
		db.QueryRow("select * from author where id = ?", id),
		&result,
		"json",
		fields...)
	if err != nil {
		panic(err)
	}
	return result
}

func getAuthor13(db *sql.DB, id string) author {
	result := author{}
	fields := []string{"num", "name"}
	err := smapping.SQLScan(
		db.QueryRow("select num, name from author where id = ?", id),
		&result,
		"json",
		fields...)
	if err != nil {
		panic(err)
	}
	return result
}

func getAllAuthor(db *sql.DB) []author {
	result := []author{}
	rows, err := db.Query("select * from author")
	if err != nil {
		panic(err)
	}
	for rows.Next() {
		res := author{}
		if err := smapping.SQLScan(rows, &res, "json"); err != nil {
			fmt.Println("error scan:", err)
			break
		}
		result = append(result, res)
	}
	return result
}

func main() {
	db, err := sql.Open("sqlite3", "./dummy.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()
	_, err = db.Exec(`
drop table if exists author;
create table author(num integer primary key autoincrement, id text, name text);
insert into author(id, name) values
('id1', 'name1'),
('this-nil', null);`)
	if err != nil {
		panic(err)
	}
	//auth1 := author{ID: &sql.NullString{String: "id1"}}
	auth1 := author{ID: sql.NullString{String: "id1"}}
	auth1 = getAuthor(db, auth1.ID.String)
	fmt.Println("auth1:", auth1)
	jsonbyte, _ := json.Marshal(auth1)
	fmt.Println("json auth1:", string(jsonbyte))
	b1 := book{Author: auth1}
	fmt.Println(b1)
	jbook1, _ := json.Marshal(b1)
	fmt.Println("json book1:", string(jbook1))
	auth2 := getAuthor(db, "this-nil")
	fmt.Println("auth2:", auth2)
	jbyte, _ := json.Marshal(auth2)
	fmt.Println("json auth2:", string(jbyte))
	b2 := book{Author: auth2}
	fmt.Println("book2:", b2)
	jbook2, _ := json.Marshal(b2)
	fmt.Println("json book2:", string(jbook2))
	fmt.Println("author12:", getAuthor12(db, auth1.ID.String))
	fmt.Println("author13:", getAuthor13(db, auth1.ID.String))
	fmt.Println("all author1:", getAllAuthor(db))
}

```

## LICENSE
MIT
