# Shoal

![image](https://user-images.githubusercontent.com/22009/90324679-b1e27a00-dfac-11ea-8e63-8ac00f56b9a7.png)

`Shoal` is a declarative tool that installs sets of [GoFish](https://github.com/fishworks/gofish/) foods.

# Usage

- CLI
- Go library

## CLI

Create a `shoal.yaml` and run `shoal sync [-f shoal.yaml]`:

```yaml
rig: https://github.com/fishworks/fish-food

foods:
  helmfile: ">= 0.125.0"
  helm: ">= 3.3.0"
  kubectl: ">= 1.18.0"
  eksctl: ">= 0.23.0"

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
		Rig: rig,
		Foods: shoal.Foods{
			Helmfile: ">= 0.125.0",
			Helm:     ">= 3.3.0",
			Kubectl:  ">= 1.18.0",
			Eksctl:   ">= 0.23.0",
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
