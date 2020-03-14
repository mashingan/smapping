/*
mapping
Golang mapping structure
*/

package smapping

import (
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

func getValTag(fieldval reflect.Value, tag string) interface{} {
	var resval interface{}
	switch fieldval.Kind() {
	case reflect.Struct:
		resval = MapTags(fieldval, tag)
	case reflect.Ptr:
		resval = MapTags(fieldval.Elem(), tag)
	default:
		resval = fieldval.Interface()
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
		val, err = handleTime(time.RFC3339, val.String(), sftype)
		if err != nil {
			return false, fmt.Errorf("smapping Time conversion: %s", err.Error())
		}

	} else if sftype != val.Type() {
		return false, fmt.Errorf("Provided value type not match object field type")
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
		if tag, ok := field.Tag.Lookup(tagname); ok {
			var err error
			vfield := sval.Field(i)
			if !vfield.IsValid() || !vfield.CanSet() {
				return false, nil
			} else if tag != tagvalue {
				continue
			} else {
				val := reflect.ValueOf(value)
				gotptr := false
				if vfield.Kind() == reflect.Ptr {
					gotptr = true
				}
				res := reflect.New(vfield.Type()).Elem()
				if isTime(vfield.Type()) {
					val, err = handleTime(time.RFC3339, val.String(), vfield.Type())
					if err != nil {
						return false, fmt.Errorf("smapping Time conversion: %s", err.Error())
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
					return false, fmt.Errorf("Provided value type not match field object")
				}
				vfield.Set(val)
				return true, nil
			}
		}
	}
	//}
	return false, nil
}

/*
FillStruct acts just like ``json.Unmarshal`` but works with ``Mapped``
instead of bytes of char that made from ``json``.
*/
func FillStruct(obj interface{}, mapped Mapped) error {
	for k, v := range mapped {
		exists, err := setField(obj, k, v)
		if err != nil {
			return err
		}
		if !exists {
			continue
		}
	}
	return nil
}

/*
FillStructByTags fills the field that has tagname and tagvalue
instead of Mapped key name.
*/
func FillStructByTags(obj interface{}, mapped Mapped, tagname string) error {
	for k, v := range mapped {
		exists, err := setFieldFromTag(obj, tagname, k, v)
		if err != nil {
			return err
		}
		if !exists {
			continue
		}
	}
	return nil
}
