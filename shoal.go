package shoal

import (
	"crypto/sha1"
	"fmt"
	"github.com/Masterminds/semver"
	"github.com/fishworks/gofish"
	"github.com/fishworks/gofish/pkg/home"
	"github.com/fishworks/gofish/pkg/ohai"
	"github.com/fishworks/gofish/pkg/rig/installer"
	"github.com/yuin/gluamapper"
	"github.com/yuin/gopher-lua"
	"golang.org/x/xerrors"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"time"
)

var DefaultRootDir = ".shoal"

var Version string

type Option func(*App)

func LogOutput(w io.Writer) Option {
	return func(app *App) {
		app.logOutput = w
	}
}

func New(opts ...Option) (*App, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	rootDir := filepath.Join(wd, DefaultRootDir)

	app := &App{
		RootDir: rootDir,
		fetched: map[string]bool{},
	}

	for _, o := range opts {
		o(app)
	}

	if app.logOutput == nil {
		app.logOutput = os.Stderr
	}

	app.logger = log.New(app.logOutput, "", log.Lshortfile)

	return app, nil
}

type App struct {
	git GitClient

	RootDir string

	fetchedMutex sync.Mutex
	fetched      map[string]bool

	logOutput io.Writer
	logger    *log.Logger
}

type versionedFood struct {
	foodCommitID string
	description  string
	food         gofish.Food
}

func (a *App) setEnv() {
	GofishRoot := a.RootDir

	os.Setenv("GOFISH_HOME", GofishRoot)
	os.Setenv("GOFISH_BINPATH", filepath.Join(GofishRoot, "bin"))
}

func (a *App) Init() error {
	a.setEnv()

	cands := []string{
		home.String(),
		home.Barrel(),
		home.Rigs(),
		home.BinPath(),
		home.Cache(),
	}

	var dirs []string

	for _, d := range cands {
		if r, _ := os.Stat(d); r == nil {
			dirs = append(dirs, d)
		}
	}

	if len(dirs) > 0 {
		a.logger.Printf("The following new directories will be created:\n")
		a.logger.Println(strings.Join(dirs, "\n"))

		for _, d := range dirs {
			if err := os.MkdirAll(d, 0755); err != nil {
				return err
			}
		}
	}

	return nil
}

func (a *App) Ensure(rig, food, constraint string) error {
	a.setEnv()

	g := a.git

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
		a.logger.Println("Listing versions")

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

		a.logger.Printf("Reading workspace cache dir at %s", workspaceCacheDir)

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

			a.logger.Printf("reading rig ID file at %s", rigIDFile)

			bs, err := ioutil.ReadFile(rigIDFile)
			if err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf("broken shoal cache: missing RIG file: please remove %s and try again", d)
				}
				return fmt.Errorf("reading RIG file: %w", err)
			}

			rigID := string(bs)

			if rigID == rig {
				workspaceDir = d
				break
			}
		}

		if workspaceDir != "" {
			a.logger.Printf("locking workspace dir at %s", workspaceDir)
			a.fetchedMutex.Lock()
			defer func() {
				a.logger.Printf("unlocking workspace dir at %s", workspaceDir)
				a.fetchedMutex.Unlock()
			}()

			if fetched := a.fetched[workspaceDir]; !fetched {
				a.logger.Printf("getting origin head branch in %s", workspaceDir)

				b, err := g.ShowOriginHeadBranch(workspaceDir)
				if err != nil {
					return err
				}

				a.logger.Printf("fetching remote changes in %s", workspaceDir)

				if err := g.Fetch(workspaceDir, b); err != nil {
					return err
				}

				a.logger.Printf("force-checking-out remote changes in %s", workspaceDir)

				if err := g.ForceCheckout(workspaceDir, b); err != nil {
					return err
				}

				a.logger.Printf("writing rig ID file in %s", workspaceDir)

				// Force check-out using go-git seems to remove all the uncommitted changes to the worktree so
				// the RIG file.
				// We have to recreate it otherwise shoal is unable to detect if this workspace dir is that of this rig
				if err := ioutil.WriteFile(filepath.Join(workspaceDir, "RIG"), []byte(rig), 0644); err != nil {
					return fmt.Errorf("writing RIG file: %w", err)
				}

				a.fetched[workspaceDir] = true
			}
		} else {
			workspaceDir = filepath.Join(workspaceCacheDir, fmt.Sprintf("%d", len(fileInfoList)))

			a.logger.Printf("cloning rig %q into %q", rig, workspaceDir)

			if err := g.Clone(rig, workspaceDir); err != nil {
				return err
			}

			a.logger.Printf("creating RIG ID file in %s", workspaceDir)

			if err := ioutil.WriteFile(filepath.Join(workspaceDir, "RIG"), []byte(rig), 0644); err != nil {
				return fmt.Errorf("writing RIG file: %w", err)
			}
		}

		filePath := filepath.Join("Food", fmt.Sprintf("%s.lua", food))

		a.logger.Printf("running git-log in %s for path %s", workspaceDir, filePath)

		gitLogOutput, err := g.Log(workspaceDir, filePath)
		if err != nil {
			return err
		}

		for _, l := range strings.Split(gitLogOutput, "\n") {
			items := strings.SplitN(l, " ", 2)

			if len(items) != 2 {
				continue
			}

			commitID := items[0]
			description := items[1]

			luaScript, err := g.Show(workspaceDir, commitID, filePath)
			if err != nil {
				return err
			}

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

		if len(versions) > 0 {
			a.logger.Printf("Fetched %d versions for food %q", len(versions), versions[0].food.Name)
			for i, v := range versions {
				a.logger.Printf("%3d: %s %s", i, v.foodCommitID[:8], v.food.Version)
			}
		}
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

	a.logger.Printf("installing %s %s...", version.food.Name, version.food.Version)

	if err := version.food.Install(); err != nil {
		return fmt.Errorf("installing %s %s: %w", version.food.Name, version.food.Version, err)
	}

	a.logger.Printf("installed %s %s.", version.food.Name, version.food.Version)

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

func (a *App) InitGitProvider(config Config) error {
	var g GitClient
	switch p := config.Git.Provider; p {
	case "go-git":
		g = &GoGit{}
	case "":
		g = &NativeGit{}
	default:
		return fmt.Errorf("invalid git.provider: %s", p)
	}

	a.git = g

	return nil
}

func (a *App) Sync(config Config) (finalErr error) {
	defer func() {
		if err := recover(); err != nil {
			finalErr = xerrors.Errorf("sync failed due to panic: %w\nSTACK TRACE:\n%s", err, debug.Stack())
		}
	}()

	rig := config.Rig

	if v := config.Foods.Helm; v != "" {
		if err := a.Ensure(rig, "helm", v); err != nil {
			return err
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

	if v := config.Helm.Plugins.Diff; v != "" {
		pluginInstall := exec.Command(filepath.Join(a.BinPath(), "helm"), "plugin", "install", "https://github.com/databus23/helm-diff", "--version", v)

		var homeSet bool

		helmPluginsDir := filepath.Join(a.RootDir, "Library")
		helmPluginsHomeEnv := "XDG_DATA_HOME"

		for _, e := range os.Environ() {
			nameValue := strings.Split(e, "=")
			name := nameValue[0]

			if len(nameValue) > 1 {
				value := nameValue[1]

				if name == helmPluginsHomeEnv {
					value = helmPluginsDir
					homeSet = true
				}

				pluginInstall.Env = append(pluginInstall.Env, name+"="+value)
			} else {
				pluginInstall.Env = append(pluginInstall.Env, name)
			}
		}

		if !homeSet {
			pluginInstall.Env = append(pluginInstall.Env, helmPluginsHomeEnv+"="+helmPluginsDir)
		}

		if o, err := pluginInstall.CombinedOutput(); err != nil {
			var out string
			if o != nil {
				out = string(o)
			}

			// TODO Upgrade the plugin if it's already installed
			if !strings.HasPrefix(out, "Error: plugin already exists") {
				return fmt.Errorf("installing helm-diff: %w\nCOMBINED OUTPUT:\n%s", err, out)
			}
		}
	}

	return nil
}

func (a *App) TempRig(source string) (string, error) {
	return a.TempDir(source)
}
