package shoal

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"github.com/Masterminds/semver"
	"github.com/fishworks/gofish"
	"github.com/fishworks/gofish/pkg/home"
	"github.com/fishworks/gofish/pkg/ohai"
	"github.com/fishworks/gofish/pkg/rig/installer"
	"github.com/mumoshu/shoal/pkg/gitrepo"
	"github.com/yuin/gluamapper"
	"github.com/yuin/gopher-lua"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"
)

var DefaultRootDir = ".shoal"

var Version string

func New() (*App, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	rootDir := filepath.Join(wd, DefaultRootDir)

	app := &App{
		RootDir: rootDir,
		fetched: map[string]bool{},
	}

	if err := app.init(); err != nil {
		return nil, err
	}

	return app, nil
}

type App struct {
	RootDir string

	fetchedMutex sync.Mutex
	fetched      map[string]bool
}

type versionedFood struct {
	foodCommitID string
	description  string
	food         gofish.Food
}

func (a *App) init() error {
	GofishRoot := a.RootDir

	os.Setenv("GOFISH_HOME", GofishRoot)
	os.Setenv("GOFISH_BINPATH", filepath.Join(GofishRoot, "bin"))
	os.Setenv("HOME", GofishRoot)

	if err := os.RemoveAll(GofishRoot); err != nil {
		return err
	}

	dirs := []string{
		home.String(),
		home.Barrel(),
		home.Rigs(),
		home.BinPath(),
		home.Cache(),
	}

	fmt.Printf("The following new directories will be created:\n")
	fmt.Println(strings.Join(dirs, "\n"))

	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return err
		}
	}

	return nil
}

func (a *App) Ensure(rig, food, constraint string) error {
	var constraints *semver.Constraints

	if constraint != "" {
		var err error

		constraints, err = semver.NewConstraint(constraint)
		if err != nil {
			return fmt.Errorf("parsing semver constraint from %q: %w", constraint, err)
		}
	}

	GofishRoot := a.RootDir

	var versions []versionedFood

	listVersions := true
	if listVersions {
		h := sha1.New()
		h.Write([]byte(rig))
		hash := fmt.Sprintf("%x", h.Sum(nil))

		workspaceCacheKey := rig
		workspaceCacheKey = strings.TrimPrefix(workspaceCacheKey, "https://")
		workspaceCacheKey = strings.TrimPrefix(workspaceCacheKey, "http://")
		workspaceCacheKey = strings.TrimPrefix(workspaceCacheKey, "git@")
		workspaceCacheKey = strings.ReplaceAll(workspaceCacheKey, string(os.PathSeparator), "-")
		workspaceCacheKey += "-" + hash

		workspaceCacheDir := filepath.Join(GofishRoot, "workspaces", workspaceCacheKey)

		if _, err := os.Lstat(workspaceCacheDir); os.IsNotExist(err) {
			if err := os.MkdirAll(workspaceCacheDir, 0755); err != nil {
				return fmt.Errorf("creating workspaces cache dir: %w", err)
			}
		}

		fileInfoList, err := ioutil.ReadDir(workspaceCacheDir)
		if err != nil {
			return err
		}

		var workspaceDir string

		for _, info := range fileInfoList {
			if !info.IsDir() {
				continue
			}

			d := filepath.Join(workspaceCacheDir, info.Name())

			rigIDFile := filepath.Join(d, "RIG")

			bs, err := ioutil.ReadFile(rigIDFile)
			if err != nil {
				return fmt.Errorf("reading RIG file: %w", err)
			}

			rigID := string(bs)

			if rigID == rig {
				workspaceDir = d
				break
			}
		}

		if workspaceDir != "" {
			a.fetchedMutex.Lock()
			defer a.fetchedMutex.Unlock()

			if fetched := a.fetched[workspaceDir]; !fetched {
				gitFetch := exec.Command("git", "fetch", "origin")
				gitFetch.Dir = workspaceDir
				if trace, err := gitFetch.CombinedOutput(); err != nil {
					return fmt.Errorf("running git-fetch: %w\n\nCOMBINED OUTPUT:\n%s", err, trace)
				}

				a.fetched[workspaceDir] = true
			}
		} else {
			workspaceDir = filepath.Join(workspaceCacheDir, fmt.Sprintf("%d", len(fileInfoList)))

			gitClone := exec.Command("git", "clone", rig, workspaceDir)
			if trace, err := gitClone.CombinedOutput(); err != nil {
				return fmt.Errorf("running git-clone: %w\n\nCOMBINED OUTPUT:\n%s", err, trace)
			}

			if err := ioutil.WriteFile(filepath.Join(workspaceDir, "RIG"), []byte(rig), 0644); err != nil {
				return fmt.Errorf("writing RIG file: %w", err)
			}
		}

		filePath := filepath.Join("Food", fmt.Sprintf("%s.lua", food))

		var gitLogStdout, gitLogStderr bytes.Buffer

		gitLog := exec.Command("git", "log", "--oneline", "--no-color", "--", filePath)
		gitLog.Dir = workspaceDir
		gitLog.Stdout = &gitLogStdout
		gitLog.Stderr = &gitLogStderr
		if err := gitLog.Run(); err != nil {
			return fmt.Errorf("running git-log: %w\n\nSTDERR:\n%s", err, gitLogStderr.String())
		}
		gitLogOutput := gitLogStdout.String()

		for _, l := range strings.Split(gitLogOutput, "\n") {
			items := strings.SplitN(l, " ", 2)

			if len(items) != 2 {
				continue
			}

			commitID := items[0]
			description := items[1]

			var gitShowStdout bytes.Buffer

			gitShow := exec.Command("git", "show", fmt.Sprintf("%s:%s", commitID, filePath))
			gitShow.Dir = workspaceDir
			gitShow.Stdout = &gitShowStdout
			if err := gitShow.Run(); err != nil {
				return fmt.Errorf("running git-show: %w", err)
			}

			luaScript := gitShowStdout.String()

			var food gofish.Food

			if ok, err := func() (bool, error) {
				l := lua.NewState()
				defer l.Close()
				if err := l.DoString(luaScript); err != nil {
					if strings.Contains(err.Error(), "syntax error") {
						return false, nil
					}
					return false, fmt.Errorf("executing lua: %w\n\nSCRIPT:\n%s", err, luaScript)
				}

				if err := gluamapper.Map(l.GetGlobal(strings.ToLower(reflect.TypeOf(food).Name())).(*lua.LTable), &food); err != nil {
					return false, fmt.Errorf("reading lua execution result: %w", err)
				}

				return true, nil
			}(); err != nil {
				return err
			} else if !ok {
				ohai.Ohaif("Ignored rotten fish %q from commit %q", food, commitID)
				continue
			}

			versions = append(versions, versionedFood{
				foodCommitID: commitID,
				description:  description,
				food:         food,
			})
		}

		ohai.Ohaif("Fetched versions: %v", versions)
	}

	var version versionedFood

	if constraints == nil {
		version = versions[0]
	} else {
		var found bool

		verToFood := map[string][]versionedFood{}

		for _, v := range versions {
			verToFood[v.food.Version] = append(verToFood[v.food.Version], v)
		}

		var vers semver.Collection

		for k := range verToFood {
			v, err := semver.NewVersion(k)
			if err != nil {
				return fmt.Errorf("parsing %q as semver: %w", k, err)
			}

			vers = append(vers, v)
		}

		sort.Sort(vers)

		for _, v := range vers {
			if constraints.Check(v) {
				found = true
				vStr := v.String()
				version = verToFood[vStr][0]
				break
			}
		}

		if !found {
			return fmt.Errorf(
				"finding food: no food matching the semver constraint %q found out of %d food versions",
				constraint,
				len(versions),
			)
		}
	}

	if err := version.food.Install(); err != nil {
		return fmt.Errorf("installing %s %s: %w", version.food.Name, version.food.Version, err)
	}

	installDefaultFishFood := false
	if installDefaultFishFood {
		ohai.Ohailn("Installing default fish food...")

		i, err := installer.New(rig, "", "")
		if err != nil {
			return err
		}

		start := time.Now()
		if err := installer.Install(i); err != nil {
			return err
		}

		t := time.Now()

		ohai.Successf("rig constructed in %s\n", t.Sub(start).String())
	}

	return nil
}

func (a *App) BinPath() string {
	return home.BinPath()
}

func (a *App) Sync(config Config) error {
	rig := config.Rig

	if v := config.Foods.Helm; v != "" {
		if err := a.Ensure(rig, "helm", v); err != nil {
			return err
		}

		if v := config.Helm.Plugins.Diff; v != "" {
			pluginInstall := exec.Command(filepath.Join(a.BinPath(), "helm"), "plugin", "install", "https://github.com/databus23/helm-diff", "--version", v)
			if err := pluginInstall.Run(); err != nil {
				return fmt.Errorf("installing helm-diff: %w", err)
			}
		}
	}

	if v := config.Foods.Kubectl; v != "" {
		if err := a.Ensure(rig, "kubectl", v); err != nil {
			return err
		}
	}

	if v := config.Foods.Helmfile; v != "" {
		if err := a.Ensure(rig, "helmfile", v); err != nil {
			return err
		}
	}

	if v := config.Foods.Eksctl; v != "" {
		if err := a.Ensure(rig, "eksctl", v); err != nil {
			return err
		}
	}

	for food, version := range config.Foods.Others {
		if err := a.Ensure(rig, food, version); err != nil {
			return err
		}
	}

	for _, d := range config.Dependencies {
		if err := a.Ensure(d.Rig, d.Food, d.Version); err != nil {
			return err
		}
	}

	return nil
}

func (a *App) TempRig(source string) (string, error) {
	return gitrepo.TempDir(source)
}
