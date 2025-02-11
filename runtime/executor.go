package runtime

import (
	"context"
	"goparse/defs"

	_ "goparse/defs/modules"

	fp "path/filepath"
)

type PlaybookExecutor struct {
	inputConfig   *PlaybookConfig
	currentConfig defs.Config
}

func NewPlaybookExecutor(config *PlaybookConfig) *PlaybookExecutor {
	pe := &PlaybookExecutor{}
	pe.inputConfig = config
	pe.currentConfig = make(defs.Config)
	for key, value := range config.ExtraVars {
		pe.currentConfig[key] = value
	}
	return pe
}

func (pe *PlaybookExecutor) ApplyConfig(config defs.Config) error {
	for key, value := range config {
		pe.currentConfig[key] = value
	}
	return nil
}

func (pe *PlaybookExecutor) CurrentConfig() defs.Config {
	return pe.currentConfig
}

func (pe *PlaybookExecutor) shouldExecute(yamlElement *defs.YamlElement) bool {
	if len(yamlElement.When) == 0 {
		return true
	}
	for _, cond := range yamlElement.When {
		output, err := resolveVars[string](cond, pe.CurrentConfig())
		if err != nil {
			return false
		}
		//	fmt.Printf("\nCondition %s evaluated to %s with %+v\n", cond, output, pe.CurrentConfig())
		if output == "false" {
			return false
		}
	}
	return true
}

func (pe *PlaybookExecutor) executeLoop(ctx context.Context, yamlElement *defs.YamlElement) error {
	loop, err := resolveLoop(yamlElement.Loop, pe.CurrentConfig())
	if err != nil {
		return err
	}
	for _, item := range loop {
		pe.currentConfig["item"] = item
		err = pe.executeBlockOrTask(ctx, yamlElement)
		if err != nil {
			return err
		}
		delete(pe.currentConfig, "item")
	}
	return nil
}

func (pe *PlaybookExecutor) executeSingleTask(ctx context.Context, yamlElement *defs.YamlElement) error {
	element, err := resolveVars[*defs.YamlElement](yamlElement, pe.CurrentConfig())
	if err != nil {
		return err
	}
	task, err := element.MakeTask()
	if err != nil {
		return err
	}
	//raw, _ := json.Marshal(yamlElement.Loop)
	//str, _ := json.Marshal(task)
	//fmt.Printf("\nRunning task: %+v with config %+v -> %+v\n", string(str), pe.CurrentConfig(), string(raw))
	out, err := task.Run(ctx, pe)
	if err != nil {
		return err
	}
	if out != nil && element.Register != nil {
		pe.currentConfig[*element.Register] = out
	}
	return nil
}

func (pe *PlaybookExecutor) executeBlockOrTask(ctx context.Context, yamlElement *defs.YamlElement) error {
	if len(yamlElement.Block) == 0 {
		return pe.executeSingleTask(ctx, yamlElement)
	}
	for _, innerConfig := range yamlElement.Block {
		err := pe.execute(ctx, innerConfig)
		if err != nil {
			return err
		}
	}
	return nil
}

func (pe *PlaybookExecutor) execute(ctx context.Context, yamlElement *defs.YamlElement) error {
	if !pe.shouldExecute(yamlElement) {
		return nil
	}
	if yamlElement.Loop == nil {
		return pe.executeBlockOrTask(ctx, yamlElement)
	}
	return pe.executeLoop(ctx, yamlElement)
}

func (pe *PlaybookExecutor) ExecuteFile(ctx context.Context, filepath string) error {
	if yes := fp.IsAbs(filepath); !yes {
		filepath = fp.Join(pe.inputConfig.YamlDir, filepath)
	}
	processor := defs.NewProcessor()
	err := processor.ParseYaml(filepath)
	if err != nil {
		return err
	}
	for _, config := range processor.YamlConfigs() {
		err = pe.execute(ctx, config)
		if err != nil {
			return err
		}
	}
	return nil
}
