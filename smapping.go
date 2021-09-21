/*
mapping
Golang mapping structure
*/

package smapping

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	s "strings"
	"time"
)

// Mapped simply an alias
type Mapped map[string]interface{}

type MapEncoder interface {
	MapEncode() (interface{}, error)
}

var mapEncoderI = reflect.TypeOf((*MapEncoder)(nil)).Elem()

type MapDecoder interface {
	MapDecode(interface{}) error
}

var mapDecoderI = reflect.TypeOf((*MapDecoder)(nil)).Elem()

func extractValue(x interface{}) reflect.Value {
	var result reflect.Value
	switch v := x.(type) {
	case reflect.Value:
		result = v
	default:
		result = reflect.ValueOf(x).Elem()
	}
	return result
}

/*
MapFields maps between struct to mapped interfaces{}.
The argument must be pointer struct or else it will throw panic error.
Now it's implemented as MapTags with empty tag "".

Only map the exported fields.
*/
func MapFields(x interface{}) Mapped {
	return MapTags(x, "")
}

func tagHead(tag string) string {
	return s.Split(tag, ",")[0]
}

func isValueNil(v reflect.Value) bool {
	for _, kind := range []reflect.Kind{
		reflect.Ptr, reflect.Slice, reflect.Map,
		reflect.Chan, reflect.Interface, reflect.Func,
	} {
		if v.Kind() == kind && v.IsNil() {
			return true
		}

	}
	return false
}

func getValTag(fieldval reflect.Value, tag string) interface{} {
	var resval interface{}
	if isValueNil(fieldval) {
		return nil
	}
	if fieldval.Type().Name() == "Time" ||
		reflect.Indirect(fieldval).Type().Name() == "Time" {
		resval = fieldval.Interface()
	} else if typof := fieldval.Type(); typof.Implements(mapEncoderI) ||
		reflect.PtrTo(typof).Implements(mapEncoderI) {
		valx, ok := fieldval.Interface().(MapEncoder)
		if !ok {
			return nil
		}
		val, err := valx.MapEncode()
		if err != nil {
			val = nil
		}
		resval = val
	} else {
		switch fieldval.Kind() {
		case reflect.Struct:
			resval = MapTags(fieldval, tag)
		case reflect.Ptr:
			indirect := reflect.Indirect(fieldval)
			if indirect.Kind() < reflect.Array || indirect.Kind() == reflect.String {
				resval = indirect.Interface()
			} else {
				resval = MapTags(fieldval.Elem(), tag)
			}
		case reflect.Slice:
			placeholder := make([]interface{}, fieldval.Len())
			for i := 0; i < fieldval.Len(); i++ {
				fieldvalidx := fieldval.Index(i)
				theval := getValTag(fieldvalidx, tag)
				placeholder[i] = theval
			}
			resval = placeholder
		default:
			resval = fieldval.Interface()
		}

	}
	return resval
}

/*
MapTags maps the tag value of defined field tag name. This enable
various field extraction that will be mapped to mapped interfaces{}.
*/
func MapTags(x interface{}, tag string) Mapped {
	result := make(Mapped)
	value := extractValue(x)
	if !value.IsValid() {
		return nil
	}
	xtype := value.Type()
	for i := 0; i < value.NumField(); i++ {
		field := xtype.Field(i)
		if field.PkgPath != "" {
			continue
		}
		fieldval := value.Field(i)
		if tag == "" {
			result[field.Name] = getValTag(fieldval, tag)
		} else if tagvalue, ok := field.Tag.Lookup(tag); ok {
			result[tagHead(tagvalue)] = getValTag(fieldval, tag)
		}
	}
	return result
}

/*
MapTagsWithDefault maps the tag with optional fallback tags. This to enable
tag differences when there are only few difference with the default ``json``
tag.
*/
func MapTagsWithDefault(x interface{}, tag string, defs ...string) Mapped {
	result := make(Mapped)
	value := extractValue(x)
	xtype := value.Type()
	for i := 0; i < value.NumField(); i++ {
		field := xtype.Field(i)
		if field.PkgPath != "" {
			continue
		}
		var (
			tagval string
			ok     bool
		)
		if tagval, ok = field.Tag.Lookup(tag); ok {
			result[tagHead(tagval)] = getValTag(value.Field(i), tag)
		} else {
			for _, deftag := range defs {
				if tagval, ok = field.Tag.Lookup(deftag); ok {
					result[tagHead(tagval)] = getValTag(value.Field(i), deftag)
					break // break from looping the defs
				}
			}
		}
	}
	return result
}

// MapTagsFlatten is to flatten mapped object with specific tag. The limitation
// of this flattening that it can't have duplicate tag name and it will give
// incorrect result because the older value will be written with newer map field value.
func MapTagsFlatten(x interface{}, tag string) Mapped {
	result := make(Mapped)
	value := extractValue(x)
	xtype := value.Type()
	for i := 0; i < value.NumField(); i++ {
		field := xtype.Field(i)
		if field.PkgPath != "" {
			continue
		}
		fieldval := value.Field(i)
		if tagvalue, ok := field.Tag.Lookup(tag); ok {
			key := tagHead(tagvalue)
			result[key] = fieldval.Interface()
			continue
		}
		fieldval = reflect.Indirect(fieldval)
		if fieldval.Type().Kind() != reflect.Struct {
			continue
		}
		nests := MapTagsFlatten(fieldval, tag)
		for k, v := range nests {
			result[k] = v
		}
	}
	return result
}

func isTime(typ reflect.Type) bool {
	return typ.Name() == "Time" || typ.String() == "*time.Time"
}
func handleTime(layout, format string, typ reflect.Type) (reflect.Value, error) {
	t, err := time.Parse(layout, format)
	var resval reflect.Value
	if err != nil {
		return resval, fmt.Errorf("time conversion: %s", err.Error())
	}
	if typ.Kind() == reflect.Ptr {
		resval = reflect.New(typ).Elem()
		resval.Set(reflect.ValueOf(&t))
	} else {
		resval = reflect.ValueOf(&t).Elem()

	}
	return resval, err
}

func isSlicedObj(val, res reflect.Value) bool {
	return val.Type().Kind() == reflect.Slice &&
		res.Kind() == reflect.Slice
}

func fillMapIter(vfield, res reflect.Value, val *reflect.Value, tagname string) error {
	iter := val.MapRange()
	m := Mapped{}
	for iter.Next() {
		m[iter.Key().String()] = iter.Value().Interface()
	}
	if vfield.Kind() == reflect.Ptr {
		vval := vfield.Type().Elem()
		ptrres := reflect.New(vval).Elem()
		for k, v := range m {
			_, err := setFieldFromTag(ptrres, tagname, k, v)
			if err != nil {
				return fmt.Errorf("ptr nested error: %s", err.Error())
			}
		}
		*val = ptrres.Addr()
	} else {
		if err := FillStructByTags(res, m, tagname); err != nil {
			return fmt.Errorf("nested error: %s", err.Error())
		}
		*val = res
	}
	return nil
}

func fillTime(vfield reflect.Value, val *reflect.Value) error {
	if (*val).Type().Name() == "string" {
		newval, err := handleTime(time.RFC3339, val.String(), vfield.Type())
		if err != nil {
			return fmt.Errorf("smapping Time conversion: %s", err.Error())
		}
		*val = newval
	} else if val.Type().Name() == "Time" {
		*val = reflect.Indirect(*val)
	}
	return nil
}

func scalarType(val reflect.Value) bool {
	if val.Kind() != reflect.Interface {
		return false
	}
	switch val.Interface().(type) {
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64, string, []byte:
		return true

	}
	return false
}

func ptrExtract(vval, rval reflect.Value) (reflect.Value, bool) {
	acttype := rval.Type().Elem()
	newrval := reflect.New(acttype).Elem()
	gotval := false
	if newrval.Kind() < reflect.Array {
		gotval = true
		ival := vval.Interface()
		if newrval.Kind() > reflect.Bool && newrval.Kind() < reflect.Uint {
			nval := reflect.ValueOf(ival).Int()
			newrval.SetInt(nval)
		} else if newrval.Kind() > reflect.Uintptr &&
			newrval.Kind() < reflect.Complex64 {
			fval := reflect.ValueOf(ival).Float()
			newrval.SetFloat(fval)
		} else {
			newrval.Set(reflect.ValueOf(ival))
		}
	}
	return newrval, gotval
}

func fillSlice(res reflect.Value, val *reflect.Value, tagname string) error {
	for i := 0; i < val.Len(); i++ {
		vval := val.Index(i)
		rval := reflect.New(res.Type().Elem()).Elem()
		if vval.Kind() < reflect.Array {
			rval.Set(vval)
			res = reflect.Append(res, rval)
			continue
		} else if scalarType(vval) {
			if rval.Kind() == reflect.Ptr {
				if newrval, ok := ptrExtract(vval, rval); ok {
					res = reflect.Append(res, newrval.Addr())
					continue
				}
			}
			rval.Set(reflect.ValueOf(vval.Interface()))
			res = reflect.Append(res, rval)
			continue
		} else if vval.IsNil() {
			res = reflect.Append(res, reflect.Zero(rval.Type()))
			continue
		}
		newrval := rval
		if rval.Kind() == reflect.Ptr {
			var ok bool
			if newrval, ok = ptrExtract(vval, rval); ok {
				res = reflect.Append(res, newrval.Addr())
				continue
			}
		}
		m, ok := vval.Interface().(Mapped)
		if !ok && newrval.Kind() >= reflect.Array {
			m = MapTags(vval.Interface(), tagname)
		}
		err := FillStructByTags(newrval, m, tagname)
		if err != nil {
			return fmt.Errorf("cannot set an element slice")
		}
		if rval.Kind() == reflect.Ptr {
			res = reflect.Append(res, newrval.Addr())
		} else {
			res = reflect.Append(res, newrval)
		}
	}
	*val = res
	return nil
}

func setFieldFromTag(obj interface{}, tagname, tagvalue string, value interface{}) (bool, error) {
	sval := extractValue(obj)
	stype := sval.Type()
	for i := 0; i < sval.NumField(); i++ {
		field := stype.Field(i)
		if field.PkgPath != "" {
			continue
		}
		vfield := sval.Field(i)
		var (
			tag string
			ok  bool
		)
		if tagname == "" && (vfield.IsValid() || vfield.CanSet()) &&
			field.Name == tagvalue {
			ok = true
		} else if tag, ok = field.Tag.Lookup(tagname); ok {
			if !vfield.IsValid() || !vfield.CanSet() {
				return false, nil
			} else if tagHead(tag) != tagvalue {
				continue
			}
		}
		if !ok {
			continue
		}
		val := reflect.ValueOf(value)
		if !val.IsValid() {
			continue
		}
		res := reflect.New(vfield.Type()).Elem()
		if typof := vfield.Type(); typof.Implements(mapDecoderI) ||
			reflect.PtrTo(typof).Implements(mapDecoderI) {
			isPtr := typof.Kind() == reflect.Ptr
			var mapval reflect.Value
			if isPtr {
				mapval = reflect.New(typof.Elem())
			} else {
				mapval = reflect.New(typof)
			}
			mapdecoder, ok := mapval.Interface().(MapDecoder)
			if !ok {
				return false, nil
			}
			if err := mapdecoder.MapDecode(value); err != nil {
				return false, err
			}
			if isPtr {
				val = reflect.ValueOf(mapdecoder)
			} else {
				val = reflect.Indirect(reflect.ValueOf(mapdecoder))
			}
		} else if isTime(vfield.Type()) {
			if err := fillTime(vfield, &val); err != nil {
				return false, err
			}
		} else if res.IsValid() && val.Type().Name() == "Mapped" {
			if err := fillMapIter(vfield, res, &val, tagname); err != nil {
				return false, err
			}
		} else if isSlicedObj(val, res) {
			if err := fillSlice(res, &val, tagname); err != nil {
				return false, err
			}
		} else if vfield.Kind() == reflect.Ptr {
			vfv := vfield.Type().Elem()
			if vfv != val.Type() {
				return false, fmt.Errorf(
					"provided value (%v) pointer type not match field tag '%s' of tagname '%s' from object",
					value, tagname, tagvalue)
			}
			nval := reflect.New(vfv).Elem()
			nval.Set(val)
			val = nval.Addr()
		} else if field.Type != val.Type() {
			return false, fmt.Errorf("provided value (%v) type not match field tag '%s' of tagname '%s' from object",
				value, tagname, tagvalue)
		}
		vfield.Set(val)
		return true, nil
	}
	return false, nil
}

/*
FillStruct acts just like ``json.Unmarshal`` but works with ``Mapped``
instead of bytes of char that made from ``json``.
*/
func FillStruct(obj interface{}, mapped Mapped) error {
	errmsg := ""
	for k, v := range mapped {
		if v == nil {
			continue
		}
		_, err := setFieldFromTag(obj, "", k, v)
		if err != nil {
			if errmsg != "" {
				errmsg += ","
			}
			errmsg += err.Error()
		}
	}
	if errmsg != "" {
		return fmt.Errorf(errmsg)
	}
	return nil
}

/*
FillStructByTags fills the field that has tagname and tagvalue
instead of Mapped key name.
*/
func FillStructByTags(obj interface{}, mapped Mapped, tagname string) error {
	errmsg := ""
	for k, v := range mapped {
		if v == nil {
			continue
		}
		_, err := setFieldFromTag(obj, tagname, k, v)
		if err != nil {
			if errmsg != "" {
				errmsg += ","
			}
			errmsg += err.Error()
		}
	}
	if errmsg != "" {
		return fmt.Errorf(errmsg)
	}
	return nil
}

func assignScanner(mapvals []interface{}, tagFields map[string]reflect.StructField,
	tag string, index int, key string, obj, value interface{}) {
	switch value.(type) {
	case int:
		mapvals[index] = new(int)
	case int8:
		mapvals[index] = new(int8)
	case int16:
		mapvals[index] = new(int16)
	case int32:
		mapvals[index] = new(int32)
	case int64:
		mapvals[index] = new(int64)
	case uint:
		mapvals[index] = new(uint)
	case uint8:
		mapvals[index] = new(uint8)
	case uint16:
		mapvals[index] = new(uint16)
	case uint32:
		mapvals[index] = new(uint32)
	case uint64:
		mapvals[index] = new(uint64)
	case string:
		mapvals[index] = new(string)
	case float32:
		mapvals[index] = new(float32)
	case float64:
		mapvals[index] = new(float64)
	case bool:
		mapvals[index] = new(bool)
	case []byte:
		mapvals[index] = new([]byte)
	case sql.Scanner, driver.Valuer, Mapped:
		mapvals[index] = new(interface{})
		typof := reflect.TypeOf(obj).Elem()
		if tag == "" {
			strufield, ok := typof.FieldByName(key)
			if !ok {
				return
			}
			typof = strufield.Type
		} else if strufield, ok := tagFields[key]; ok {
			typof = strufield.Type
		} else {
			for i := 0; i < typof.NumField(); i++ {
				strufield := typof.Field(i)
				if tagval, ok := strufield.Tag.Lookup(tag); ok {
					tagFields[key] = strufield
					if tagHead(tagval) == key {
						typof = strufield.Type
						break
					}
				}
			}
		}

		scannerI := reflect.TypeOf((*sql.Scanner)(nil)).Elem()
		if typof.Implements(scannerI) || reflect.PtrTo(typof).Implements(scannerI) {
			valx := reflect.New(typof).Elem()
			mapvals[index] = valx.Addr().Interface()
		}
	default:
	}

}

func assignValuer(mapres Mapped, tagFields map[string]reflect.StructField,
	tag, key string, obj, value interface{}) {
	switch v := value.(type) {
	case *int8:
		mapres[key] = *v
	case *int16:
		mapres[key] = *v
	case *int32:
		mapres[key] = *v
	case *int64:
		mapres[key] = *v
	case *int:
		mapres[key] = *v
	case *uint8:
		mapres[key] = *v
	case *uint16:
		mapres[key] = *v
	case *uint32:
		mapres[key] = *v
	case *uint64:
		mapres[key] = *v
	case *uint:
		mapres[key] = *v
	case *string:
		mapres[key] = *v
	case *bool:
		mapres[key] = *v
	case *float32:
		mapres[key] = *v
	case *float64:
		mapres[key] = *v
	case *[]byte:
		mapres[key] = *v
	case *driver.Valuer:
	default:
		typof := reflect.TypeOf(obj).Elem()
		if tag == "" {
			strufield, ok := typof.FieldByName(key)
			if !ok {
				return
			}
			typof = strufield.Type
		} else if strufield, ok := tagFields[key]; ok {
			typof = strufield.Type
		} else {
			for i := 0; i < typof.NumField(); i++ {
				strufield := typof.Field(i)
				if tagval, ok := strufield.Tag.Lookup(tag); ok {
					if tagHead(tagval) == key {
						typof = strufield.Type
						break
					}
				}
			}
		}
		valuerI := reflect.TypeOf((*driver.Valuer)(nil)).Elem()
		if typof.Implements(valuerI) || reflect.PtrTo(typof).Implements(valuerI) {
			valx := reflect.New(typof).Elem()
			valv := reflect.Indirect(reflect.ValueOf(value))
			valx.Set(valv)
			mapres[key] = valx.Interface()
		}
		// ignore if it's not recognized
	}
}

// SQLScanner is the interface that dictate
// any type that implement Scan method to
// be compatible with sql.Row Scan method.
type SQLScanner interface {
	Scan(dest ...interface{}) error
}

/*
SQLScan is the function that will map scanning object based on provided
field name or field tagged string. The tags can receive the empty string
"" and then it will map the field name by default.
*/
func SQLScan(row SQLScanner, obj interface{}, tag string, x ...string) error {
	var mapres Mapped
	if tag == "" {
		mapres = MapFields(obj)
	} else {
		mapres = MapTags(obj, tag)
	}
	fieldsName := x
	length := len(x)
	if length == 0 || (length == 1 && x[0] == "*") {
		typof := reflect.TypeOf(obj).Elem()
		newfields := make([]string, typof.NumField())
		length = typof.NumField()
		for i := 0; i < length; i++ {
			field := typof.Field(i)
			if tag == "" {
				newfields[i] = field.Name
			} else {
				if tagval, ok := field.Tag.Lookup(tag); ok {
					newfields[i] = tagHead(tagval)
				}
			}
		}
		fieldsName = newfields
	}
	mapvals := make([]interface{}, length)
	tagFields := make(map[string]reflect.StructField)
	for i, k := range fieldsName {
		assignScanner(mapvals, tagFields, tag, i, k, obj, mapres[k])
	}
	if err := row.Scan(mapvals...); err != nil {
		return err
	}
	for i, k := range fieldsName {
		assignValuer(mapres, tagFields, tag, k, obj, mapvals[i])
	}
	var err error
	if tag == "" {
		err = FillStruct(obj, mapres)
	} else {
		err = FillStructByTags(obj, mapres, tag)
	}
	return err
}
