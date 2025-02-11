package modules

import (
	"context"
	"goparse/defs"
	"os"

	"github.com/noirbizarre/gonja"
)

func init() {
	defs.MustRegisterTask(&Template{})
}

type Template struct {
	Src  string `json:"src"`
	Dest string `json:"dest"`
	Mod  uint32 `json:"mode"`
}

func (task *Template) Name() string {
	return "template"
}

func (task *Template) Init(yamlElement *defs.YamlElement) error {
	return yamlElement.ReadTaskConfig(task)
}

func (task *Template) Run(ctx context.Context, executor defs.PlaybookExecutor) (defs.Output, error) {
	tpl, err := gonja.FromFile(task.Src)
	if err != nil {
		return nil, err
	}
	output, err := tpl.Execute(executor.CurrentConfig())
	if err != nil {
		return nil, err
	}
	err = os.WriteFile(task.Dest, []byte(output), os.FileMode(task.Mod))
	if err != nil {
		panic(err)
	}
	return string(output), nil
}
