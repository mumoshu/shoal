package shoal

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

func (a *App) TempDir(source string) (string, error) {
	g := a.git
	if g == nil {
		g = &NativeGit{}
	}

	tempRemote, err := ioutil.TempDir(os.TempDir(), "shoal-remote")
	if err != nil {
		return "", err
	}

	if err := g.InitBare(tempRemote); err != nil {
		return "", err
	}

	tempLocal, err := ioutil.TempDir(os.TempDir(), "shoal-local")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tempLocal)

	// go-git fails on cloning an empty repository. So init/add remote instead.
	//if err := g.Clone(tempRemote, tempLocal); err != nil {
	//	return "", err
	//}

	if err := g.Init(tempLocal); err != nil {
		return "", err
	}

	if err := g.AddRemote(tempLocal, "origin", tempRemote); err != nil {
		return "", err
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

		if err := g.Add(tempLocal, rel); err != nil {
			return err
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
		if err := g.Config(tempLocal, k, v); err != nil {
			return "", err
		}
	}

	if err := g.Commit(tempLocal, "first commit"); err != nil {
		return "", err
	}

	if err := g.Push(tempLocal, "origin", "master"); err != nil {
		return "", err
	}

	return tempRemote, nil
}
