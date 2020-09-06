# Shoal

![image](https://user-images.githubusercontent.com/22009/90324679-b1e27a00-dfac-11ea-8e63-8ac00f56b9a7.png)

`Shoal` is a declarative tool that installs sets of [GoFish](https://github.com/fishworks/gofish/) foods.

# Usage

- CLI
- Go library

## CLI

Create a `shoal.yaml` and run `shoal [-f shoal.yaml] sync`:

```yaml
rig: &rig https://github.com/fishworks/fish-food

dependencies:
- rig: *rig
  food: helmfile
  version: ">= 0.125.0"
- rig: *rig
  food: helm
  version: ">= 3.3.0"
- rig: *rig
  food: kubectl
  version: ">= 1.18.0"
- rig: *rig
  food: eksctl
  version: ">= 0.23.0"

  # Additionally, you can declare whatever food found in
  #   https://github.com/fishworks/fish-food/tree/main/Food

helm:
  plugins:
    diff: ">= 3.1.3"
```

The installed binaries are linked under `$PWD/.shoal/bin`.

## Go library

Create a `shoal.Config` and run `shoal/App.Sync` on it.
The installation path can be obtained via `shoal/App.BinPath`:

```go
import "github.com/fishworks/gofish/shoal"

func example() {
	rig := "https://github.com/fishworks/fish-food"

	app, err := shoal.New()
	if err != nil {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := app.Sync(shoal.Config{
		Dependencies: []shoal.Dependency{
			{
				Rig:     rig,
				Food:    "helmfile",
				Version: ">= 0.125.0",
			},
			{
				Rig:     rig,
				Food:    "helm",
				Version: ">= 3.3.0",
			},
			{
				Rig:     rig,
				Food:    "kubectl",
				Version: ">= 1.18.0",
			},
			{
				Rig:     rig,
				Food:    "eksctl",
				Version: ">= 0.23.0",
			},
		},
	}); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	binPath := app.BinPath()

	cmd := exec.Command(filepath.Join(binPath, "helm"), "version", "-c")
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: running helm: %v", err)
		os.Exit(1)
	}
}
```

# go-git integration

`shoal` has two implementations of the `provider`:

- `native` (default)
- `go-git`

The `go-git` provider uses [go-git](https://github.com/go-git/go-git) instead of the `git` command installed on your system,
for whatever git operation it needs to run, like `clone`, `log`, etc.

To enable the `go-git` provider, add the `git` section to your config, and set `provider` to `go-git` in it:

```yaml
git:
  provider: go-git
```

# History and Context

My initial goal was to build a cross-platform package manager that can be embedded into my [terraform-provider-eksctl](https://github.com/mumoshu/terraform-provider-eksctl) and [terraform-provider-helmfile](https://github.com/mumoshu/terraform-provider-helmfile),
so that those tf providers are able to install and upgrade extra binaries
like `eksctl` and `helmfile` on demand,
while allowing users to manage versions of those binaries declaratively.

At the time of writing the first version of `shoal`, there was already a robust cross-platform package manager written in Go, called [gofish](https://github.com/fishworks/gofish), authored by @bacongobbler.
I've conducted some manual testing, code reading, made [one change](https://github.com/fishworks/gofish/pull/174) to `gofish`, and successfully built the initial version of `shoal` on top of `gofish`.

`shoal` was implemented by mostly reorganizing `gofish init` and `gofish install` code, then adding a feature to fetch historical revisions of rigs(repos) and foods(packages) from Git, and a declarative config syntax.
The reorganization and additions were needed because `gofish init` doesn't support sudo-less mode and `gofish install` was unable to specify the specific version of food(package) to install.

Although the original use-case was to embed it into terraform providers, I got to think that something like `brew-bundle` for Homebrew can be easily built around it, so I built the command-line interface to it. That's `shoal sync`.
