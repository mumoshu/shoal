package main

import (
	"fmt"
	"github.com/fishworks/gofish/pkg/ohai"
	"github.com/mumoshu/shoal"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
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

	if err := app.Ensure(rig, "helm", ">= 3.3.0"); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	if err := app.Ensure(rig, "kubectl", "> 1.18"); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	if err := app.Ensure(rig, "helmfile", "> 0.125.0"); err != nil {
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

	ohai.Ohaif("TEST OUTPUT:\n%s", string(out))
}
