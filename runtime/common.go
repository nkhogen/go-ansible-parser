package runtime

import (
	"encoding/json"
	"fmt"
	"goparse/defs"
)

type PlaybookConfig struct {
	YamlDir   string                 `json:"yaml_dir"`
	ExtraVars map[string]interface{} `json:"extra_vars"`
}

func resolveVars[V any](input V, values defs.Config) (V, error) {
	return defs.ResolveVars[V](input, defs.DefaultTemplateResolver(values))
}

func resolveLoop(yamlLoop *defs.YamlLoop, values defs.Config) ([]any, error) {
	if yamlLoop.Var != nil {
		str, err := resolveVars[string](*yamlLoop.Var, values)
		if err != nil {
			return nil, err
		}
		loop := []any{}
		err = json.Unmarshal([]byte(str), &loop)
		if err != nil {
			return nil, err
		}
		return loop, nil
	}
	if yamlLoop.Items != nil {
		slice, err := resolveVars[[]any](yamlLoop.Items, values)
		if err != nil {
			return nil, err
		}
		return slice, nil
	}
	return nil, fmt.Errorf("Unsupported loop type %+v", yamlLoop)
}
