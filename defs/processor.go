package defs

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	yamlElementFieldParsers = map[string]YamlConfigFieldParser{}
)

func init() {
	registerYamlElementFieldParsers()
}

type YamlConfigFieldParser func(name string, node *yaml.Node, yamlElement *YamlElement) error

func registerYamlElementFieldParsers() {
	yamlElementFieldParsers["name"] = func(name string, node *yaml.Node, yamlElement *YamlElement) error {
		var v string
		err := node.Decode(&v)
		if err != nil {
			return err
		}
		yamlElement.Name = &v
		return nil
	}
	yamlElementFieldParsers["register"] = func(name string, node *yaml.Node, yamlElement *YamlElement) error {
		var v string
		err := node.Decode(&v)
		if err != nil {
			return err
		}
		yamlElement.Register = &v
		return nil
	}
	yamlElementFieldParsers["when"] = func(name string, node *yaml.Node, yamlElement *YamlElement) error {
		fn := func(cond string) string {
			return fmt.Sprintf("{%% if %s %%}true{%% else %%}false{%% endif %%}", cond)
		}
		switch node.Kind {
		case yaml.SequenceNode:
			v := []string{}
			err := node.Decode(&v)
			if err != nil {
				return err
			}
			yamlElement.When = make([]string, len(v))
			for i, cond := range v {
				yamlElement.When[i] = fn(cond)
			}
		case yaml.ScalarNode:
			var v string
			err := node.Decode(&v)
			if err != nil {
				return err
			}
			yamlElement.When = []string{
				fn(v),
			}
		default:
			return fmt.Errorf("Unsupported node kind %v", node.Kind)
		}
		return nil
	}
	yamlElementFieldParsers["environment"] = func(name string, node *yaml.Node, yamlElement *YamlElement) error {
		v := StrConfig{}
		err := node.Decode(&v)
		if err != nil {
			return err
		}
		yamlElement.Environ = v
		return nil
	}
	yamlElementFieldParsers["loop"] = func(name string, node *yaml.Node, yamlElement *YamlElement) error {
		yamlLoop := YamlLoop{}
		err := node.Decode(&yamlLoop)
		if err != nil {
			return err
		}
		yamlElement.Loop = &yamlLoop
		return nil
	}
	yamlElementFieldParsers["ignore_errors"] = func(name string, node *yaml.Node, yamlElement *YamlElement) error {
		var v bool
		err := node.Decode(&v)
		if err != nil {
			return err
		}
		yamlElement.IgnoreErrors = v
		return nil
	}
	yamlElementFieldParsers["block"] = func(name string, node *yaml.Node, yamlElement *YamlElement) error {
		yamlElements := YamlElements{}
		err := node.Decode(&yamlElements)
		if err != nil {
			return err
		}
		for _, child := range yamlElements {
			child.Parent = yamlElement
		}
		yamlElement.Block = yamlElements
		return nil
	}
}

type Processor struct {
	yamlElements YamlElements
}

func NewProcessor() *Processor {
	return &Processor{yamlElements: YamlElements{}}
}

func (processor *Processor) YamlConfigs() YamlElements {
	return processor.yamlElements
}

func (processor *Processor) ParseYaml(filepath string) error {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return err
	}
	err = processor.validateYamlElementFieldParsers()
	if err != nil {
		return err
	}
	yamlElements := YamlElements{}
	err = yaml.Unmarshal([]byte(data), &yamlElements)
	if err != nil {
		return err
	}
	processor.yamlElements = yamlElements
	return nil
}

// This ensures field parsers are registered for all the JSON tagged fields of YamlElement.
func (processor *Processor) validateYamlElementFieldParsers() error {
	sType := reflect.TypeOf(YamlElement{})
	for i := 0; i < sType.NumField(); i++ {
		if jsonTags, ok := sType.Field(i).Tag.Lookup("json"); ok {
			jsonTag := strings.Split(jsonTags, ",")[0]
			if _, ok := yamlElementFieldParsers[jsonTag]; !ok {
				return fmt.Errorf("No parser found for field %s", jsonTag)
			}
		}
	}
	return nil
}
