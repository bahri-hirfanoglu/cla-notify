package config

import (
	"fmt"
	"reflect"
	"strings"
)

type kind int

const (
	kindUnsupported kind = iota
	kindString
	kindBool
	kindNumber
	kindMap
	kindStruct
)

func classify(t reflect.Type) kind {
	switch t.Kind() {
	case reflect.String:
		return kindString
	case reflect.Bool:
		return kindBool
	case reflect.Int, reflect.Float64:
		return kindNumber
	case reflect.Map:
		return kindMap
	case reflect.Struct:
		return kindStruct
	default:
		return kindUnsupported
	}
}

// fieldTypeByTag finds the field of struct type t whose json tag matches name.
func fieldTypeByTag(t reflect.Type, name string) (reflect.Type, bool) {
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tag := strings.Split(f.Tag.Get("json"), ",")[0]
		if tag == name {
			return f.Type, true
		}
	}
	return nil, false
}

// lookupType walks the Config schema following dot-path parts; a map's next segment is a free key.
// It also returns the canonical path, where map keys are wildcarded as "*" for value rule lookup.
func lookupType(parts []string) (reflect.Type, string, error) {
	t := reflect.TypeOf(Config{})
	canonical := make([]string, 0, len(parts))
	for _, part := range parts {
		switch classify(t) {
		case kindStruct:
			ft, ok := fieldTypeByTag(t, part)
			if !ok {
				return nil, "", fmt.Errorf("unknown key")
			}
			t = ft
			canonical = append(canonical, part)
		case kindMap:
			t = t.Elem()
			canonical = append(canonical, "*")
		default:
			return nil, "", fmt.Errorf("unknown key")
		}
	}
	return t, strings.Join(canonical, "."), nil
}
