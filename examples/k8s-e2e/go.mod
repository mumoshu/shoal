module github.com/mumoshu/shoal/examples/k8s-e2e

go 1.14

require (
	github.com/mumoshu/shoal v0.0.0-20200816053351-ee2ec69f44d2
	k8s.io/apimachinery v0.18.6
	k8s.io/client-go v0.18.6
	sigs.k8s.io/controller-runtime v0.6.2
)

replace (
	github.com/fishworks/gofish => github.com/mumoshu/gofish v0.13.1-0.20200908033248-ab2d494fb15c
	github.com/mumoshu/shoal => ../../
)
