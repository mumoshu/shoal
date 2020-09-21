package shoal

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestGit(t *testing.T) {
	c := &NativeGit{}

	testClient(t, c)

	gc := &GoGit{}

	testClient(t, gc)
}

func testClient(t *testing.T, c GitClient) {
	t.Helper()

	RepoURL := "git@github.com:fishworks/fish-food.git"

	d, err := ioutil.TempDir("", "gittest")
	if err != nil {
		t.Fatalf("creating tempdir: %v", err)
	}

	defer func() {
		err := os.RemoveAll(d)
		if err != nil {
			t.Logf("removing all %s: %v", d, err)
		}
	}()

	if err := c.Clone(RepoURL, d); err != nil {
		t.Fatalf("cloning repo: %v", err)
	}

	b, err := c.ShowOriginHeadBranch(d)
	if err != nil {
		t.Fatalf("showing origin head branch: %v", err)
	}

	if err := c.ForceCheckout(d, b); err != nil {
		t.Fatalf("force checking out %s: %v", b, err)
	}
}
