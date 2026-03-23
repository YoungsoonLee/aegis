package policy

type PolicyFile struct {
	Policies []Policy `yaml:"policies"`
}

type Policy struct {
	Name   string `yaml:"name"`
	Rules  []Rule `yaml:"rules"`
	Labels map[string]string `yaml:"labels,omitempty"`
}

type Rule struct {
	Guard    string            `yaml:"guard"`
	Action   string            `yaml:"action"`
	Severity string            `yaml:"severity,omitempty"`
	Params   map[string]any    `yaml:"params,omitempty"`
}
