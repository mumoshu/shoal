package shoal

type Config struct {
	Git Git `yaml:"git"`

	Rig string `yaml:"rig"`

	Foods Foods `yaml:"foods"`
	Helm  Helm  `yaml:"helm"`

	Dependencies []Dependency `yaml:"dependencies"`
}

type Git struct {
	Provider string `yaml:"provider"`
}

type Dependency struct {
	Rig     string `yaml:"rig"`
	Food    string `yaml:"food"`
	Version string `yaml:"version"`
}

type Foods struct {
	Helmfile string `yaml:"helmfile"`
	Helm     string `yaml:"helm"`
	Kubectl  string `yaml:"kubectl"`
	Eksctl   string `yaml:"eksctl"`

	Others map[string]string `yaml:",inline"`
}

type Helm struct {
	Plugins HelmPlugins `yaml:"plugins"`
}

type HelmPlugins struct {
	Diff string `yaml:"diff"`
}
