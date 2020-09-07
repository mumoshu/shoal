package shoal

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"os/exec"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
)

type GitClient interface {
	Fetch(string) error
	Clone(repo string, dir string) error
	Log(string, string) (string, error)
	Show(string, string, string) (string, error)
	InitBare(dir string) error
	Add(local string, rel string) error
	Config(dir string, string, v string) error
	Commit(dir string, msg string) error
	Push(local string, remote string, branch string) error
	Init(dir string) error
	AddRemote(dir string, name string, url string) error
}

type NativeGit struct {
}

func (n *NativeGit) Init(dir string) error {
	gitInit := exec.Command("git", "init", dir)
	if err := gitInit.Run(); err != nil {
		return err
	}
	return nil
}

func (n *NativeGit) AddRemote(dir string, name string, url string) error {
	gitRemoteAdd := exec.Command("git", "remote", "add", name, url)
	gitRemoteAdd.Dir = dir
	if out, err := gitRemoteAdd.CombinedOutput(); err != nil {
		return fmt.Errorf("running git-remote add: %w\n\nCOMBINED OUTPUT:\n%s", err, out)
	}
	return nil
}

func (n *NativeGit) Push(local string, remote string, branch string) error {
	gitPush := exec.Command("git", "push", remote, branch)
	gitPush.Dir = local
	if err := gitPush.Run(); err != nil {
		return fmt.Errorf("running git-push: %w", err)
	}
	return nil
}

func (n *NativeGit) Commit(dir string, msg string) error {
	gitCommit := exec.Command("git", "commit", "-m", msg)
	gitCommit.Dir = dir
	if out, err := gitCommit.CombinedOutput(); err != nil {
		return fmt.Errorf("running git-commit: %w\n\nCOMBINED OUTPUT:\n%s", err, string(out))
	}

	return nil
}

func (n *NativeGit) Config(tempLocal string, k string, v string) error {
	gitConfig := exec.Command("git", "config", k, v)
	gitConfig.Dir = tempLocal
	if out, err := gitConfig.CombinedOutput(); err != nil {
		return fmt.Errorf("running git-config: %w\n\nCOMBINED OUTPUT:\n%s", err, string(out))
	}
	return nil
}

func (n *NativeGit) Add(tempLocal string, rel string) error {
	cmd := exec.Command("git", "add", rel)
	cmd.Dir = tempLocal
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running git-add: %w", err)
	}
	return nil
}

func (n *NativeGit) InitBare(tempRemote string) error {
	gitInit := exec.Command("git", "init", "--bare", tempRemote)
	if err := gitInit.Run(); err != nil {
		return err
	}
	return nil
}

var _ GitClient = &NativeGit{}

func (n *NativeGit) Fetch(workspaceDir string) error {
	gitFetch := exec.Command("git", "fetch", "origin")
	gitFetch.Dir = workspaceDir
	if trace, err := gitFetch.CombinedOutput(); err != nil {
		return fmt.Errorf("running git-fetch: %w\n\nCOMBINED OUTPUT:\n%s", err, trace)
	}
	return nil
}

func (n *NativeGit) Clone(rig, workspaceDir string) error {
	gitClone := exec.Command("git", "clone", rig, workspaceDir)
	if trace, err := gitClone.CombinedOutput(); err != nil {
		return fmt.Errorf("running git-clone: %w\n\nCOMBINED OUTPUT:\n%s", err, trace)
	}

	return nil
}

func (n *NativeGit) Log(workspaceDir, filePath string) (string, error) {
	var gitLogStdout, gitLogStderr bytes.Buffer

	gitLog := exec.Command("git", "log", "--oneline", "--no-color", "--", filePath)
	gitLog.Dir = workspaceDir
	gitLog.Stdout = &gitLogStdout
	gitLog.Stderr = &gitLogStderr
	if err := gitLog.Run(); err != nil {
		return "", fmt.Errorf("running git-log: %w\n\nSTDERR:\n%s", err, gitLogStderr.String())
	}
	gitLogOutput := gitLogStdout.String()

	return gitLogOutput, nil
}

func (n *NativeGit) Show(workspaceDir, commitID, filePath string) (string, error) {
	var gitShowStdout bytes.Buffer

	gitShow := exec.Command("git", "show", fmt.Sprintf("%s:%s", commitID, filePath))
	gitShow.Dir = workspaceDir
	gitShow.Stdout = &gitShowStdout
	if err := gitShow.Run(); err != nil {
		return "", fmt.Errorf("running git-show: %w", err)
	}

	luaScript := gitShowStdout.String()

	return luaScript, nil
}

type GoGit struct {
}

func (n *GoGit) Init(dir string) error {
	if _, err := git.PlainInit(dir, false); err != nil {
		return fmt.Errorf("go-get init %q: %w", dir, err)
	}
	return nil
}

func (n *GoGit) AddRemote(dir string, name string, url string) error {
	r, err := git.PlainOpen(dir)
	if err != nil {
		return err
	}

	if _, err := r.CreateRemote(&config.RemoteConfig{
		Name: name,
		URLs: []string{url},
	}); err != nil {
		return fmt.Errorf("go-git remote add: %w", err)
	}

	return nil
}

func (n *GoGit) InitBare(dir string) error {
	if _, err := git.PlainInit(dir, true); err != nil {
		return fmt.Errorf("go-get init %q: %w", dir, err)
	}
	return nil
}

func (n *GoGit) Add(local string, rel string) error {
	r, err := git.PlainOpen(local)
	if err != nil {
		return err
	}

	w, err := r.Worktree()
	if err != nil {
		return err
	}

	_, err = w.Add(rel)
	if err != nil {
		return nil
	}
	return nil
}

func (n *GoGit) Config(dir string, string, v string) error {
	return nil
}

func (n *GoGit) Commit(dir string, msg string) error {
	r, err := git.PlainOpen(dir)
	if err != nil {
		return err
	}

	w, err := r.Worktree()
	if err != nil {
		return err
	}

	_, err = w.Commit(msg, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "user",
			Email: "user@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("go-git commit: %w", err)
	}
	return nil
}

func (n *GoGit) Push(local string, remote string, branch string) error {
	r, err := git.PlainOpen(local)
	if err != nil {
		return fmt.Errorf("go-git opening %q: %w", local, err)
	}

	if err := r.Push(&git.PushOptions{
		RemoteName: remote,
		RefSpecs: []config.RefSpec{
			config.RefSpec(branch + ":" + branch),
		},
	}); err != nil {
		return fmt.Errorf("go-git pushing %q %q to %q: %w", local, branch, remote, err)
	}

	return nil
}

var _ GitClient = &GoGit{}

func (n *GoGit) Fetch(workspaceDir string) error {
	r, err := git.PlainOpen(workspaceDir)
	if err != nil {
		return fmt.Errorf("go-git opening %q: %w", workspaceDir, err)
	}

	if err := r.Fetch(&git.FetchOptions{
		RemoteName: "origin",
	}); err != nil && err.Error() != "already up-to-date" {
		return fmt.Errorf("go-git fetching %q: %w", workspaceDir, err)
	}

	return nil
}

func (n *GoGit) Clone(rig, workspaceDir string) error {
	_, err := git.PlainClone(workspaceDir, false, &git.CloneOptions{
		URL: rig,
	})
	if err != nil {
		return fmt.Errorf("go-git cloning %q into %q: %w", rig, workspaceDir, err)
	}

	return nil
}

func (n *GoGit) Log(workspaceDir, filePath string) (string, error) {
	r, err := git.PlainOpen(workspaceDir)
	if err != nil {
		return "", fmt.Errorf("go-git opening %q: %w", workspaceDir, err)
	}

	c, err := r.Log(&git.LogOptions{
		FileName: &filePath,
	})
	if err != nil {
		return "", fmt.Errorf("go-git log %q %q: %w", workspaceDir, filePath, err)
	}

	var gitLogOutput bytes.Buffer

	skip := errors.New("skip remaining")

	if err := c.ForEach(func(commit *object.Commit) error {
		lines := strings.Split(commit.Message, "\n")
		oneline := strings.TrimSpace(lines[0])
		if _, err := commit.File(filePath); err != nil {
			// `go-git log -- PATH` seems to return commits that doesn't have any object at PATH.
			return skip
		}
		if _, err := gitLogOutput.Write([]byte(fmt.Sprintf("%s %s\n", commit.ID(), oneline))); err != nil {
			return fmt.Errorf("proessing commit %q: %w", commit.ID(), err)
		}
		return nil
	}); err != nil && err != skip {
		return "", fmt.Errorf("go-git interating log entry: %w", err)
	}

	return gitLogOutput.String(), nil
}

func (n *GoGit) Show(workspaceDir, commitID, filePath string) (string, error) {
	r, err := git.PlainOpen(workspaceDir)
	if err != nil {
		return "", fmt.Errorf("go-git opening %q: %w", workspaceDir, err)
	}

	c, err := r.CommitObject(plumbing.NewHash(commitID))
	if err != nil {
		return "", fmt.Errorf("go-git showing %q: %w", commitID, err)
	}

	f, err := c.File(filePath)
	if err != nil {
		return "", fmt.Errorf("go-git file %q: %w", filePath, err)
	}

	contents, err := f.Contents()
	if err != nil {
		return "", fmt.Errorf("go-git gettting contents of %q: %w", filePath, err)
	}

	return contents, nil
}
