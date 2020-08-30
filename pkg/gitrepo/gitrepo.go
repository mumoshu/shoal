package gitrepo

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
)

func TempDir(source string) (string, error) {
	tempRemote, err := ioutil.TempDir(os.TempDir(), "shoal-remote")
	if err != nil {
		return "", err
	}

	gitInit := exec.Command("git", "init", "--bare", tempRemote)
	if err := gitInit.Run(); err != nil {
		return "", err
	}

	tempLocal, err := ioutil.TempDir(os.TempDir(), "shoal-local")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tempLocal)

	cmd := exec.Command("git", "clone", tempRemote, tempLocal)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("running git-clone: %w", err)
	}

	sourceAbs, err := filepath.Abs(source)
	if err != nil {
		return "", err
	}

	if err := filepath.Walk(sourceAbs, func(path string, info os.FileInfo, err error) error {
		rel, err := filepath.Rel(sourceAbs, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(tempLocal, rel)

		if info.IsDir() {
			err := os.MkdirAll(dstPath, 0755)
			return err
		}

		src, err := os.Open(path)
		if err != nil {
			return err
		}
		defer src.Close()

		dst, err := os.Create(dstPath)
		if err != nil {
			return err
		}
		defer dst.Close()

		_, err = io.Copy(dst, src)
		if err != nil {
			return err
		}

		_ = dst.Close()
		_ = src.Close()

		cmd := exec.Command("git", "add", rel)
		cmd.Dir = tempLocal
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("running git-add: %w", err)
		}

		return nil
	}); err != nil {
		return "", err
	}

	config := map[string]string{
		"user.email": "user@example.com",
		"user.name":  "user",
	}

	for k, v := range config {
		gitConfig := exec.Command("git", "config", k, v)
		gitConfig.Dir = tempLocal
		if out, err := gitConfig.CombinedOutput(); err != nil {
			return "", fmt.Errorf("running git-config: %w\n\nCOMBINED OUTPUT:\n%s", err, string(out))
		}
	}

	gitCommit := exec.Command("git", "commit", "-m", "first commit")
	gitCommit.Dir = tempLocal
	if out, err := gitCommit.CombinedOutput(); err != nil {
		return "", fmt.Errorf("running git-commit: %w\n\nCOMBINED OUTPUT:\n%s", err, string(out))
	}

	gitPush := exec.Command("git", "push", "origin", "master")
	gitPush.Dir = tempLocal
	if err := gitPush.Run(); err != nil {
		return "", fmt.Errorf("running git-push: %w", err)
	}

	return tempRemote, nil
}
