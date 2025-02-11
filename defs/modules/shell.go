package modules

import (
	"context"
	"fmt"
	"goparse/defs"
	"os/exec"
)

func init() {
	defs.MustRegisterTask(&Shell{})
}

type Shell struct {
	Command string `json:"cmd"`
}

func (task *Shell) Name() string {
	return "shell"
}

func (task *Shell) Init(yamlElement *defs.YamlElement) error {
	fmt.Printf("\nEnvironment: %+v\n", yamlElement.Environment())
	return yamlElement.ReadTaskConfig(task)
}

func (task *Shell) Run(ctx context.Context, executor defs.PlaybookExecutor) (defs.Output, error) {
	cmd := exec.Command("/bin/bash", "-c", task.Command)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	fmt.Println(string(output))
	return string(output), nil
}
