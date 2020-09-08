package main

import (
	"flag"
	"fmt"
	"github.com/mumoshu/shoal"
	"gopkg.in/yaml.v2"
	"os"
)

func main() {
	var configFile string

	flag.StringVar(&configFile, "f", "shoal.yaml", "Path to the config file")

	flag.Parse()

	if flag.Arg(0) == "version" {
		fmt.Fprintf(os.Stdout, "%s\n", shoal.Version)
		os.Exit(0)
	}

	f, err := os.Open(configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening %s: %v\n", configFile, err)
		os.Exit(1)
	}

	var config shoal.Config

	d := yaml.NewDecoder(f)

	if err := d.Decode(&config); err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding yaml file %q: %v\n", configFile, err)
		os.Exit(1)
	}

	app, err := shoal.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error %v\n", err)
		os.Exit(1)
	}

	if err := app.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := app.Sync(config); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
