package main

import (
	"bytes"
	"fmt"
	"github.com/mumoshu/shoal"
	"io"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/util/json"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"os"
	"os/exec"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"time"
)

var testEnv *envtest.Environment

func main() {
	//wd, err := os.Getwd()
	//if err != nil {
	//	panic(err)
	//}
	//
	//GofishRoot := filepath.Join(wd, shoal.DefaultRootDir)
	//
	//if err := os.RemoveAll(GofishRoot); err != nil {
	//	panic(err)
	//}

	s, err := shoal.New()
	if err != nil {
		panic(err)
	}

	if err := s.Init(); err != nil {
		panic(err)
	}

	if err := s.InitGitProvider(shoal.Config{
		Git: shoal.Git{
			//Provider: "go-git",
			Provider: "",
		},
	}); err != nil {
		panic(err)
	}

	rig, err := s.TempRig("./rig")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(rig)

	conf := shoal.Config{
		Git: shoal.Git{
			Provider: "go-git",
		},
		Helm: shoal.Helm{},
		Dependencies: []shoal.Dependency{
			{
				Rig:     rig,
				Food:    "kubebuilder",
				Version: "2.3.1",
			},
			{
				Rig:     "https://github.com/fishworks/fish-food",
				Food:    "helm",
				Version: "3.3.0",
			},
		},
	}

	if err := s.InitGitProvider(conf); err != nil {
		panic(err)
	}

	if err := s.Sync(conf); err != nil {
		panic(err)
	}

	apiserverPath := filepath.Join(s.BinPath(), "kube-apiserver")
	etcdPath := filepath.Join(s.BinPath(), "etcd")

	os.Setenv("TEST_ASSET_KUBE_APISERVER", apiserverPath)
	os.Setenv("TEST_ASSET_ETCD", etcdPath)

	testEnv = &envtest.Environment{
		// Settings pathes here doesn't work sa they are overriden by envvars
		//ControlPlane: envtest.ControlPlane{
		//	APIServer: &envtest.APIServer{
		//		Path: apiserverPath,
		//	},
		//	Etcd: &envtest.Etcd{
		//		Path: etcdPath,
		//	},
		//},
		//CRDDirectoryPaths: []string{filepath.Join("..", "config", "crd", "bases")},
	}

	_, err = testEnv.Start()
	if err != nil {
		panic(err)
	}

	apiServerURL := testEnv.ControlPlane.APIURL().Host
	config := &clientcmdapi.Config{}
	config.Clusters = make(map[string]*clientcmdapi.Cluster)
	cluster := "local"
	config.Clusters[cluster] = &clientcmdapi.Cluster{
		Server: apiServerURL,
	}
	context := "default"
	config.Contexts = make(map[string]*clientcmdapi.Context)
	config.Contexts[context] = &clientcmdapi.Context{
		Cluster: cluster,
	}
	config.CurrentContext = context

	bs, err := json.Marshal(config)
	if err != nil {
		panic(err)
	}

	kcFile, err := ioutil.TempFile(os.TempDir(), "shoal-kubeconfig")
	if err != nil {
		panic(err)
	}
	defer func() {
		kcFile.Close()
		_ = os.Remove(kcFile.Name())
	}()

	_, err = io.Copy(kcFile, bytes.NewBuffer(bs))
	if err != nil {
		panic(err)
	}

	helmVersion := exec.Command(filepath.Join(s.BinPath(), "helm"), "version")
	helmVersion.Env = append(helmVersion.Env, os.Environ()...)
	helmVersion.Env = append(helmVersion.Env, "KUBECONFIG="+kcFile.Name())
	if out, err := helmVersion.CombinedOutput(); err != nil {
		panic(fmt.Errorf("running helm-version: %w\n\nCOMBINED OUTPUT:\n%s", err, string(out)))
	} else {
		fmt.Printf("helm-version output: %s\n", string(out))
	}

	defer func() {
		testEnv.Stop()
	}()

	// Your test scenario follows...

	time.Sleep(60 * time.Second)
}
