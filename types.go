package shoal

type Config struct {
	Rig string `yaml:"rig"`

	Foods Foods `yaml:"foods"`
}

type Foods struct {
	Helmfile string `yaml:"helmfile"`
	Helm     string `yaml:"helm"`
	HelmDiff string `yaml:"helmDiff"`
	Kubectl  string `yaml:"kubectl"`
	Eksctl   string `yaml:"eksctl"`
}

