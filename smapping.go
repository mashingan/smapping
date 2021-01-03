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

Only map the exported fields.
*/
func MapFields(x interface{}) Mapped {
	result := make(Mapped)
	argvalue := extractValue(x)
	argtype := argvalue.Type()
	for i := 0; i < argvalue.NumField(); i++ {
		field := argtype.Field(i)
		if field.PkgPath != "" {
			continue
		}
		result[field.Name] = argvalue.Field(i).Interface()
	}
	return result
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
	if fieldval.Type().Name() == "Time" {
		resval = fieldval.Interface()
	} else {
		switch fieldval.Kind() {
		case reflect.Struct:
			resval = MapTags(fieldval, tag)
		case reflect.Ptr:
			resval = MapTags(fieldval.Elem(), tag)
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
		if tagvalue, ok := field.Tag.Lookup(tag); ok {
			fieldval := value.Field(i)
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
		fkind := fieldval.Kind()
		if fkind == reflect.Ptr {
			fieldval = fieldval.Elem()
		}
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

func setField(obj interface{}, name string, value interface{}) (bool, error) {
	sval := extractValue(obj)
	sfval := sval.FieldByName(name)
	if !sfval.IsValid() {
		return false, nil
	}
	if !sfval.CanSet() {
		return false, fmt.Errorf("Cannot set field %s in object", name)
	}
	sftype := sfval.Type()
	val := reflect.ValueOf(value)
	if isTime(sftype) {
		var err error
		if val.Type().Name() == "string" {
			val, err = handleTime(time.RFC3339, val.String(), sftype)
			if err != nil {
				return false, fmt.Errorf("smapping Time conversion: %s", err.Error())
			}
		}
	} else if sftype != val.Type() {
		return false, fmt.Errorf("Provided value (%v) type not match object field '%s' type",
			value, name)
	}
	sfval.Set(val)
	return true, nil
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
			err error
		)
		if tag, ok = field.Tag.Lookup(tagname); ok {
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
		gotptr := false
		if vfield.Kind() == reflect.Ptr {
			gotptr = true
		}
		res := reflect.New(vfield.Type()).Elem()
		if isTime(vfield.Type()) {
			if val.Type().Name() == "string" {
				val, err = handleTime(time.RFC3339, val.String(), vfield.Type())
				if err != nil {
					return false, fmt.Errorf("smapping Time conversion: %s", err.Error())
				}
			}
		} else if res.IsValid() && val.Type().Name() == "Mapped" {
			iter := val.MapRange()
			m := Mapped{}
			for iter.Next() {
				m[iter.Key().String()] = iter.Value().Interface()
			}
			if gotptr {
				vval := vfield.Type().Elem()
				ptrres := reflect.New(vval).Elem()
				for k, v := range m {
					success, err := setFieldFromTag(ptrres, tagname, k, v)
					if err != nil {
						return false, fmt.Errorf("Ptr nested error: %s", err.Error())
					}
					if !success {
						continue
					}
				}
				val = ptrres.Addr()
			} else {
				if err := FillStructByTags(res, m, tagname); err != nil {
					return false, fmt.Errorf("Nested error: %s", err.Error())
				}
				val = res
			}
		} else if field.Type != val.Type() {
			return false, fmt.Errorf("Provided value (%v) type not match field tag '%s' of tagname '%s' from object",
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
		exists, err := setField(obj, k, v)
		if err != nil {
			if errmsg != "" {
				errmsg += ","
			}
			errmsg += err.Error()
		}
		if !exists {
			continue
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
		exists, err := setFieldFromTag(obj, tagname, k, v)
		if err != nil {
			if errmsg != "" {
				errmsg += ","
			}
			errmsg += err.Error()
		}
		if !exists {
			continue
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
	switch value.(type) {
	case *int8:
		mapres[key] = *(value.(*int8))
	case *int16:
		mapres[key] = *(value.(*int16))
	case *int32:
		mapres[key] = *(value.(*int32))
	case *int64:
		mapres[key] = *(value.(*int64))
	case *int:
		mapres[key] = *(value.(*int))
	case *uint8:
		mapres[key] = *(value.(*uint8))
	case *uint16:
		mapres[key] = *(value.(*uint16))
	case *uint32:
		mapres[key] = *(value.(*uint32))
	case *uint64:
		mapres[key] = *(value.(*uint64))
	case *uint:
		mapres[key] = *(value.(*uint))
	case *string:
		mapres[key] = *(value.(*string))
	case *bool:
		mapres[key] = *(value.(*bool))
	case *float32:
		mapres[key] = *(value.(*float32))
	case *float64:
		mapres[key] = *(value.(*float64))
	case *[]byte:
		mapres[key] = *(value.(*[]byte))
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
		lookupAssgn:
			for i := 0; i < typof.NumField(); i++ {
				strufield := typof.Field(i)
				if tagval, ok := strufield.Tag.Lookup(tag); ok {
					if tagHead(tagval) == key {
						typof = strufield.Type
						break lookupAssgn
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
