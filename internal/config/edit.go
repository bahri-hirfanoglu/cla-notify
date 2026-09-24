package config

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// Get resolves a dot path against an already-merged Config and returns its value.
func Get(cfg Config, path string) (interface{}, error) {
	parts := strings.Split(path, ".")
	v := reflect.ValueOf(cfg)
	for _, part := range parts {
		switch v.Kind() {
		case reflect.Struct:
			fv, ok := fieldByJSONTag(v, part)
			if !ok {
				return nil, fmt.Errorf("unknown key: %s", path)
			}
			v = fv
		case reflect.Map:
			mv := v.MapIndex(reflect.ValueOf(part))
			if !mv.IsValid() {
				return nil, fmt.Errorf("unknown key: %s", path)
			}
			v = mv
		default:
			return nil, fmt.Errorf("unknown key: %s", path)
		}
	}
	return v.Interface(), nil
}

func fieldByJSONTag(v reflect.Value, name string) (reflect.Value, bool) {
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tag := strings.Split(f.Tag.Get("json"), ",")[0]
		if tag == name {
			return v.Field(i), true
		}
	}
	return reflect.Value{}, false
}

// FormatValue renders a value returned by Get for CLI output.
func FormatValue(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case bool:
		return strconv.FormatBool(t)
	case int:
		return strconv.Itoa(t)
	case float64:
		return strconv.FormatFloat(t, 'g', -1, 64)
	default:
		data, err := json.Marshal(t)
		if err != nil {
			return fmt.Sprintf("%v", t)
		}
		return string(data)
	}
}

// Set parses value against the schema at path and writes it into the raw override tree.
func Set(raw map[string]interface{}, path, value string) error {
	parts := strings.Split(path, ".")
	t, canonical, err := lookupType(parts)
	if err != nil {
		return fmt.Errorf("unknown key: %s", path)
	}

	var parsed interface{}
	switch classify(t) {
	case kindBool:
		b, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid value for %s: expected a boolean", path)
		}
		parsed = b
	case kindNumber:
		n, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid value for %s: expected a number", path)
		}
		parsed = n
	case kindString:
		parsed = value
	default:
		return fmt.Errorf("%s does not name a settable value", path)
	}

	if problem := checkValue(canonical, parsed); problem != "" {
		return fmt.Errorf("invalid value for %s: %s", path, problem)
	}

	setPath(raw, parts, parsed)
	return nil
}

func setPath(raw map[string]interface{}, parts []string, value interface{}) {
	m := raw
	for i, part := range parts {
		if i == len(parts)-1 {
			m[part] = value
			return
		}
		next, ok := m[part].(map[string]interface{})
		if !ok {
			next = map[string]interface{}{}
			m[part] = next
		}
		m = next
	}
}

// Unset removes a key from the raw override tree, pruning parent maps left empty.
func Unset(raw map[string]interface{}, path string) error {
	parts := strings.Split(path, ".")
	if _, _, err := lookupType(parts); err != nil {
		return fmt.Errorf("unknown key: %s", path)
	}
	unsetPath(raw, parts)
	return nil
}

func unsetPath(m map[string]interface{}, parts []string) {
	if len(parts) == 0 {
		return
	}
	if len(parts) == 1 {
		delete(m, parts[0])
		return
	}
	next, ok := m[parts[0]].(map[string]interface{})
	if !ok {
		return
	}
	unsetPath(next, parts[1:])
	if len(next) == 0 {
		delete(m, parts[0])
	}
}

// Validate reports every unknown key and type mismatch in the raw override tree, sorted for determinism.
func Validate(raw map[string]interface{}) []string {
	var problems []string
	walkValidate(reflect.TypeOf(Config{}), raw, "", "", &problems)
	sort.Strings(problems)
	return problems
}

func walkValidate(t reflect.Type, node map[string]interface{}, prefix, canonicalPrefix string, problems *[]string) {
	for _, key := range sortedKeys(node) {
		val := node[key]
		path := key
		if prefix != "" {
			path = prefix + "." + key
		}
		canonicalPath := key
		if canonicalPrefix != "" {
			canonicalPath = canonicalPrefix + "." + key
		}
		ft, ok := fieldTypeByTag(t, key)
		if !ok {
			*problems = append(*problems, "unknown key: "+path)
			continue
		}
		switch classify(ft) {
		case kindStruct:
			sub, ok := val.(map[string]interface{})
			if !ok {
				*problems = append(*problems, "invalid value for "+path+": expected an object")
				continue
			}
			walkValidate(ft, sub, path, canonicalPath, problems)
		case kindMap:
			sub, ok := val.(map[string]interface{})
			if !ok {
				*problems = append(*problems, "invalid value for "+path+": expected an object")
				continue
			}
			for _, sk := range sortedKeys(sub) {
				subPath := path + "." + sk
				sv, ok := sub[sk].(float64)
				if !ok {
					*problems = append(*problems, fmt.Sprintf("invalid value for %s: expected a number", subPath))
					continue
				}
				if problem := checkValue(canonicalPath+".*", sv); problem != "" {
					*problems = append(*problems, fmt.Sprintf("invalid value for %s: %s", subPath, problem))
				}
			}
		case kindString:
			s, ok := val.(string)
			if !ok {
				*problems = append(*problems, "invalid value for "+path+": expected a string")
			} else if problem := checkValue(canonicalPath, s); problem != "" {
				*problems = append(*problems, fmt.Sprintf("invalid value for %s: %s", path, problem))
			}
		case kindBool:
			if _, ok := val.(bool); !ok {
				*problems = append(*problems, "invalid value for "+path+": expected a boolean")
			}
		case kindNumber:
			n, ok := val.(float64)
			if !ok {
				*problems = append(*problems, "invalid value for "+path+": expected a number")
			} else if problem := checkValue(canonicalPath, n); problem != "" {
				*problems = append(*problems, fmt.Sprintf("invalid value for %s: %s", path, problem))
			}
		default:
			*problems = append(*problems, "unknown key: "+path)
		}
	}
}

func sortedKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
