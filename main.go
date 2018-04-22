package main

import (
	"fmt"
	"path/filepath"

	"github.com/appscode/go/log"
	"github.com/golang/glog"
	"github.com/tamalsaha/go-oneliners"
	"k8s.io/api/core/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/kubernetes/pkg/credentialprovider"
	runtimeapi "k8s.io/kubernetes/pkg/kubelet/apis/cri/v1alpha1/runtime"
	"k8s.io/kubernetes/pkg/util/parsers"
)

func main() {
	//x, y, z, err := parsers.ParseImageName("nginx")
	//fmt.Println("x = ", x)
	//fmt.Println("y = ", y)
	//fmt.Println("z = ", z)
	//fmt.Println("err = ", err)

	x, y, z, err := parsers.ParseImageName("k8s.gcr.io/kube-proxy-amd64:v1.10.0")
	fmt.Println("x = ", x)
	fmt.Println("y = ", y)
	fmt.Println("z = ", z)
	fmt.Println("err = ", err)

	masterURL := ""
	kubeconfigPath := filepath.Join(homedir.HomeDir(), ".kube/config")

	config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfigPath)
	if err != nil {
		log.Fatalf("Could not get Kubernetes config: %s", err)
	}

	kc := kubernetes.NewForConfigOrDie(config)
	oneliners.FILE(kc.CoreV1().Nodes())
}

// PullImage pulls an image from the network to local storage using the supplied secrets if necessary.
func PullImage(img string, pullSecrets []v1.Secret) (string, error) {
	repoToPull, _, _, err := parsers.ParseImageName(img)
	if err != nil {
		return "", err
	}

	keyring, err := credentialprovider.MakeDockerKeyring(pullSecrets, credentialprovider.NewDockerKeyring())
	if err != nil {
		return "", err
	}

	imgSpec := &runtimeapi.ImageSpec{Image: img}
	creds, withCredentials := keyring.Lookup(repoToPull)
	if !withCredentials {
		glog.V(3).Infof("Pulling image %q without credentials", img)

		var auth *runtimeapi.AuthConfig = nil
		fmt.Printf("Pull image %q auth %v", img, auth)

		return "imageRef", nil
	}

	var pullErrs []error
	for _, currentCreds := range creds {
		authConfig := credentialprovider.LazyProvide(currentCreds)
		auth := &runtimeapi.AuthConfig{
			Username:      authConfig.Username,
			Password:      authConfig.Password,
			Auth:          authConfig.Auth,
			ServerAddress: authConfig.ServerAddress,
			IdentityToken: authConfig.IdentityToken,
			RegistryToken: authConfig.RegistryToken,
		}

		fmt.Println(imgSpec, auth)
		// pullErrs = append(pullErrs, err)
	}

	return "", utilerrors.NewAggregate(pullErrs)
}
