/*
mapping
Golang mapping structure
*/

package smapping

import (
	"fmt"
	"reflect"
)

type Mapped map[string]interface{}

/*
This function maps between struct to mapped interfaces{}.
The argument must be pointer struct or else it will throw panic error.

Only map the exported fields.
*/
func MapFields(x interface{}) Mapped {
	result := make(Mapped)
	argvalue := reflect.ValueOf(x).Elem()
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

/*
This function maps the tag value of defined field tag name. This enable
various field extraction that will be mapped to mapped interfaces{}.
*/
func MapTags(x interface{}, tag string) Mapped {
	result := make(Mapped)
	value := reflect.ValueOf(x).Elem()
	xtype := value.Type()
	for i := 0; i < value.NumField(); i++ {
		field := xtype.Field(i)
		if field.PkgPath != "" {
			continue
		}
		if tagvalue, ok := field.Tag.Lookup(tag); ok {
			result[tagvalue] = value.Field(i).Interface()
		}
	}
	return result
}

func setField(obj interface{}, name string, value interface{}) (bool, error) {
	sval := reflect.ValueOf(obj).Elem()
	sfval := sval.FieldByName(name)
	if !sfval.IsValid() {
		return false, nil
	}
	if !sfval.CanSet() {
		return false, fmt.Errorf("Cannot set field %s in object", name)
	}
	sftype := sfval.Type()
	val := reflect.ValueOf(value)
	if sftype != val.Type() {
		return false, fmt.Errorf("Provided value type not match object field type")
	}
	sfval.Set(val)
	return true, nil
}

func setFieldFromTag(obj interface{}, tagname, tagvalue string, value interface{}) (bool, error) {
	sval := reflect.ValueOf(obj).Elem()
	stype := sval.Type()
	for i := 0; i < sval.NumField(); i++ {
		field := stype.Field(i)
		if field.PkgPath != "" {
			continue
		}
		if tag, ok := field.Tag.Lookup(tagname); ok {
			vfield := sval.Field(i)
			if !vfield.IsValid() || !vfield.CanSet() {
				return false, nil
			} else if tag != tagvalue {
				continue
			} else {
				val := reflect.ValueOf(value)
				if field.Type != val.Type() {
					return false, fmt.Errorf("Provided value type not match field object")
				}
				vfield.Set(val)
				return true, nil
			}
		}
	}
	return false, nil
}

func FillStruct(obj interface{}, mapped Mapped) error {
	for k, v := range mapped {
		exists, err := SetField(obj, k, v)
		if err != nil {
			return err
		}
		if !exists {
			continue
		}
	}
	return nil
}

func FillStructByTags(obj interface{}, mapped Mapped, tagname string) error {
	for k, v := range mapped {
		exists, err := SetFieldFromTag(obj, tagname, k, v)
		if err != nil {
			return err
		}
		if !exists {
			continue
		}
	}
	return nil
}
