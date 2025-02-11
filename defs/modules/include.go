package modules

import (
	"context"
	"goparse/defs"
)

func init() {
	defs.MustRegisterTask(&Include{})
}

type Include struct {
	Files []string `json:"files"`
}

func (task *Include) Name() string {
	return "include"
}

func (task *Include) Init(yamlElement *defs.YamlElement) error {
	return yamlElement.ReadTaskConfig(task)
}

func (task *Include) Run(ctx context.Context, executor defs.PlaybookExecutor) (defs.Output, error) {
	for _, file := range task.Files {
		err := executor.ExecuteFile(ctx, file)
		if err != nil {
			return nil, err
		}
	}
	return nil, nil
}
