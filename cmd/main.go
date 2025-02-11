package main

import (
	"context"
	"goparse/runtime"
	"path/filepath"
)

func main() {
	path, _ := filepath.Abs("config_mini.yaml")
	path = filepath.Dir(path)
	input := &runtime.PlaybookConfig{
		YamlDir: path,
		ExtraVars: map[string]interface{}{
			"yb_process_type": "master",
		},
	}
	executor := runtime.NewPlaybookExecutor(input)
	err := executor.ExecuteFile(context.TODO(), "config_mini.yaml")
	if err != nil {
		panic(err)
	}
}
