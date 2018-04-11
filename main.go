package main

import (
	"github.com/kubernetes/kubernetes/pkg/util/parsers"
	"fmt"
)

func main() {
	x, y, z, err := parsers.ParseImageName("k8s.gcr.io/kube-proxy-amd64:v1.10.0")
	fmt.Println("x = ", x)
	fmt.Println("y = ", y)
	fmt.Println("z = ", z)
	fmt.Println("err = ", err)
}
