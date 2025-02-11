package modules

import (
	"context"
	"goparse/defs"
)

func init() {
	defs.MustRegisterTask(&SetFact{})
}

type SetFact struct {
	Config defs.Config
}

func (task *SetFact) Name() string {
	return "set_fact"
}

func (task *SetFact) Init(yamlElement *defs.YamlElement) error {
	task.Config = make(defs.Config)
	return yamlElement.ReadTaskConfig(&task.Config)
}

func (task *SetFact) Run(ctx context.Context, executor defs.PlaybookExecutor) (defs.Output, error) {
	err := executor.ApplyConfig(task.Config)
	return nil, err
}
