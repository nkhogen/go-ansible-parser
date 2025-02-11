package defs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"gopkg.in/yaml.v3"
)

// Config is a generic key-value pairs.
type Config map[string]any

// Configs is a slice of Configs.
type Configs []Config

type StrConfig map[string]string

// Output is a generic output.
type Output any

// TemplateResolver converts a template string to a resolved string.
type TemplateResolver func(string) (string, error)

// YamlElement represents the configuration of a unit in the YAML file.
type YamlElement struct {
	Block        YamlElements `json:"block"`
	Name         *string      `json:"name"`
	When         []string     `json:"when"`
	Register     *string      `json:"register"`
	Environ      StrConfig    `json:"environment"`
	IgnoreErrors bool         `json:"ignore_errors"`
	Loop         *YamlLoop    `json:"loop"`
	Parent       *YamlElement
	Task         *YamlTask
}

// YamlTask represents the configuration of a task in the YAML file.
type YamlTask struct {
	Name   string `json:"name"`
	Config Config `json:"config"`
}

type YamlLoop struct {
	Var   *string `json:"var"`
	Items []any   `json:"items"`
}

var (
	registeredTaskTypes = map[string]reflect.Type{}
)

type YamlElements []*YamlElement

type Task interface {
	Name() string
	Init(*YamlElement) error
	Run(context.Context, PlaybookExecutor) (Output, error)
}

type TaskRunner interface {
	Run(context.Context, PlaybookExecutor) (Output, error)
}

type PlaybookExecutor interface {
	ExecuteFile(context.Context, string) error
	ApplyConfig(Config) error
	CurrentConfig() Config
}
type taskRunner struct {
	yamlElement *YamlElement
	task        Task
}

// MustRegisterTask registers the task.
func MustRegisterTask(task Task) {
	value := reflect.Indirect(reflect.ValueOf(task)).Interface()
	if _, ok := registeredTaskTypes[task.Name()]; ok {
		panic(fmt.Sprintf("Task %s is already registered", task.Name()))
	}
	registeredTaskTypes[task.Name()] = reflect.TypeOf(value)
}

func (runner *taskRunner) Run(ctx context.Context, executor PlaybookExecutor) (Output, error) {
	name := runner.task.Name()
	if runner.yamlElement.Name != nil {
		name = *runner.yamlElement.Name
	}
	fmt.Printf("\nRunning task %s\n", name)
	err := runner.task.Init(runner.yamlElement)
	if err != nil {
		return nil, fmt.Errorf("Init failed for task %s", string(runner.task.Name()))
	}
	output, err := runner.task.Run(ctx, executor)
	if err != nil && runner.yamlElement.IgnoreErrors {
		err = nil
	}
	return output, err
}

func (yamlTask *YamlTask) validate() error {
	if yamlTask.Config == nil {
		return errors.New("Task config is nil")
	}
	return nil
}

func (yamlLoop *YamlLoop) validate() error {
	if yamlLoop.Var == nil && len(yamlLoop.Items) == 0 {
		return errors.New("Either var or items must be set")
	}
	if yamlLoop.Var != nil && len(yamlLoop.Items) > 0 {
		return errors.New("Loop var and items are both set")
	}
	return nil
}

// MakeTask instantiates and initializes the task from the config.
func (yamlElement *YamlElement) MakeTask() (TaskRunner, error) {
	yamlTask := yamlElement.Task
	if yamlTask == nil {
		return nil, errors.New("Task is nil")
	}
	task, ok := reflect.New(registeredTaskTypes[yamlTask.Name]).Interface().(Task)
	if !ok {
		return nil, fmt.Errorf("Cannot instantiate task %s", string(yamlTask.Name))
	}
	return &taskRunner{task: task, yamlElement: yamlElement}, nil
}

func (yamlElements *YamlElements) UnmarshalYAML(value *yaml.Node) error {
	if value == nil {
		return nil
	}
	if value.Kind != yaml.SequenceNode {
		return fmt.Errorf("Sequence node is expected, but found %v", value.Kind)
	}
	for _, node := range value.Content {
		yamlElement := &YamlElement{}
		err := node.Decode(yamlElement)
		if err != nil {
			return err
		}
		*yamlElements = append(*yamlElements, yamlElement)
	}
	return nil
}

func (yamlElement *YamlElement) UnmarshalYAML(value *yaml.Node) error {
	if value == nil {
		return nil
	}
	if value.Kind != yaml.MappingNode {
		return fmt.Errorf("Mapping node is expected, but found %v", value.Kind)
	}
	for i := 0; i < len(value.Content); i += 2 {
		key := value.Content[i]
		val := value.Content[i+1]
		var strKey string
		err := key.Decode(&strKey)
		if err != nil {
			return err
		}
		if fieldParser, ok := yamlElementFieldParsers[strKey]; ok {
			err := fieldParser(strKey, val, yamlElement)
			if err != nil {
				return err
			}
		} else if _, ok := registeredTaskTypes[strKey]; ok {
			if yamlElement.Task != nil {
				return fmt.Errorf("Task is already configured for %s", yamlElement.Task.Name)
			}
			taskConfig := Config{}
			err = val.Decode(&taskConfig)
			if err != nil {
				return err
			}
			yamlElement.Task = &YamlTask{Name: strKey, Config: taskConfig}
		} else {
			return fmt.Errorf("Unknown field %s", strKey)
		}
	}
	return yamlElement.validate()
}

func (yamlElement *YamlElement) ReadTaskConfig(receiver any) error {
	if reflect.TypeOf(receiver).Kind() != reflect.Ptr {
		return errors.New("Receiver must be a pointer")
	}
	if yamlElement.Task == nil {
		return errors.New("Task is nil")
	}
	configJson, err := json.Marshal(yamlElement.Task.Config)
	if err != nil {
		return err
	}
	return json.Unmarshal(configJson, receiver)
}

func (yamlElement *YamlElement) Environment() StrConfig {
	env := StrConfig{}
	yamlElement.environment(&env)
	return env
}

func (yamlElement *YamlElement) environment(env *StrConfig) {
	if yamlElement.Parent != nil {
		yamlElement.Parent.environment(env)
	}
	for key, val := range yamlElement.Environ {
		(*env)[key] = val
	}
}

// TODO add more checks.
func (yamlElement *YamlElement) validate() error {
	if len(yamlElement.Block) > 0 {
		if yamlElement.Register != nil {
			return errors.New("Block cannot have register")
		}
		if yamlElement.Task != nil {
			return errors.New("Block cannot have task")
		}
	} else if yamlElement.Task != nil {
		if len(yamlElement.Block) > 0 {
			return errors.New("Task cannot have block")
		}
		return yamlElement.Task.validate()
	}
	if yamlElement.Loop != nil {
		err := yamlElement.Loop.validate()
		if err != nil {
			return err
		}
	}
	return nil
}
