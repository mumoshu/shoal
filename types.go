package shoal

type Config struct {
	Rig string `yaml:"rig"`

	Foods Foods `yaml:"foods"`
	Helm  Helm  `yaml:"helm"`
}

type Foods struct {
	Helmfile string `yaml:"helmfile"`
	Helm     string `yaml:"helm"`
	Kubectl  string `yaml:"kubectl"`
	Eksctl   string `yaml:"eksctl"`
}

type Helm struct {
	Plugins HelmPlugins `yaml:"plugins"`
}

type HelmPlugins struct {
	Diff string `yaml:"diff"`
}
