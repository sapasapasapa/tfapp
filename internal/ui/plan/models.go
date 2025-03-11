package plan

type TerraformPlan struct {
	FormatVersion    string           `json:"format_version"`
	TerraformVersion string           `json:"terraform_version"`
	ResourceChanges  []ResourceChange `json:"resource_changes"`
	PlannedValues    PlannedValues    `json:"planned_values"`
}

type PlannedValues struct {
	RootModule RootModule `json:"root_module"`
}

type RootModule struct {
	Resources    []Resource    `json:"resources"`
	ChildModules []ChildModule `json:"child_modules"`
}

type ChildModule struct {
	Address   string     `json:"address"`
	Resources []Resource `json:"resources"`
}

type Resource struct {
	Address         string                 `json:"address"`
	Type            string                 `json:"type"`
	Name            string                 `json:"name"`
	Values          map[string]interface{} `json:"values"`
	SensitiveValues map[string]interface{} `json:"sensitive_values"`
}

type ResourceChange struct {
	Address       string     `json:"address"`
	ModuleAddress string     `json:"module_address"`
	Mode          string     `json:"mode"`
	Type          string     `json:"type"`
	Name          string     `json:"name"`
	ProviderName  string     `json:"provider_name"`
	Change        ChangeData `json:"change"`
}

type ChangeData struct {
	Actions      []string               `json:"actions"`
	Before       interface{}            `json:"before"`
	After        map[string]interface{} `json:"after"`
	AfterUnknown map[string]interface{} `json:"after_unknown"`
}
