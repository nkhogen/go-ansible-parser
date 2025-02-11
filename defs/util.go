package defs

import (
	"reflect"

	"github.com/noirbizarre/gonja"
)

func DefaultTemplateResolver(values Config) TemplateResolver {
	return TemplateResolver(func(str string) (string, error) {
		tpl, err := gonja.FromString(str)
		if err != nil {
			return "", err
		}
		return tpl.Execute(values)
	})
}

func ResolveVars[V any](input V, resolver TemplateResolver) (V, error) {
	var output V
	visited := map[uintptr]struct{}{}
	i, err := resolveVars(input, resolver, visited)
	if err != nil {
		return output, err
	}
	if i != nil {
		output = i.(V)
	}
	return output, nil
}

// This resolves the variables in the input structure using the provided resolver function.
func resolveVars(input any, resolver TemplateResolver, visited map[uintptr]struct{}) (any, error) {
	value := reflect.ValueOf(input)
	targetValue := reflect.Indirect(value)
	if !targetValue.IsValid() {
		return nil, nil
	}
	if targetValue.CanAddr() {
		address := targetValue.Addr().Pointer()
		if _, ok := visited[address]; ok {
			return input, nil
		}
		visited[address] = struct{}{}
	}
	var output any
	targetValueType := targetValue.Type()
	switch targetValueType.Kind() {
	case reflect.Struct:
		clonePtr := reflect.New(targetValueType)
		clone := clonePtr.Elem()
		for i := 0; i < targetValue.NumField(); i++ {
			name := targetValueType.Field(i).Name
			field := targetValue.FieldByName(name)
			cloneField := clone.FieldByName(name)
			if field.IsValid() {
				resolvedValue, err := resolveVars(field.Interface(), resolver, visited)
				if err != nil {
					return nil, err
				}
				if resolvedValue != nil {
					cloneField.Set(reflect.ValueOf(resolvedValue))
				}
			}
		}
		if value.Type().Kind() == reflect.Pointer {
			output = clonePtr.Interface()
		} else {
			output = clone.Interface()
		}
	case reflect.Slice:
		clone := reflect.MakeSlice(targetValueType, targetValue.Len(), targetValue.Len())
		for i := 0; i < targetValue.Len(); i++ {
			element := targetValue.Index(i)
			cloneElement := clone.Index(i)
			resolvedValue, err := resolveVars(element.Interface(), resolver, visited)
			if err != nil {
				return nil, err
			}
			if resolvedValue != nil {
				cloneElement.Set(reflect.ValueOf(resolvedValue))
			}
		}
		if value.Kind() == reflect.Pointer {
			output = clone.Addr().Interface()
		} else {
			output = clone.Interface()
		}
	case reflect.Map:
		clone := reflect.MakeMap(targetValueType)
		for _, key := range targetValue.MapKeys() {
			mValue := targetValue.MapIndex(key)
			if mValue.IsValid() {
				resolvedValue, err := resolveVars(mValue.Interface(), resolver, visited)
				if err != nil {
					return nil, err
				}
				if resolvedValue != nil {
					clone.SetMapIndex(key, reflect.ValueOf(resolvedValue))
				}
			}
			if value.Kind() == reflect.Pointer {
				output = clone.Addr().Interface()
			} else {
				output = clone.Interface()
			}
		}
	case reflect.String:
		strValue, ok := targetValue.Interface().(string)
		if ok {
			resolvedValue, err := resolver(strValue)
			if err != nil {
				return nil, err
			}
			if value.Type().Kind() == reflect.Pointer {
				clone := reflect.New(targetValueType).Elem()
				clone.SetString(resolvedValue)
				output = clone.Addr().Interface()
			} else {
				output = resolvedValue
			}
		}
	default:
		output = value.Interface()
	}
	return output, nil
}
