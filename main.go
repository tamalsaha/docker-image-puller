package main

import (
	"fmt"
	"path/filepath"

	manifestV1 "github.com/docker/distribution/manifest/schema1"
	manifestV2 "github.com/docker/distribution/manifest/schema2"
	reg "github.com/heroku/docker-registry-client/registry"
	// Credential providers
	_ "k8s.io/kubernetes/pkg/credentialprovider/aws"
	_ "k8s.io/kubernetes/pkg/credentialprovider/azure"
	_ "k8s.io/kubernetes/pkg/credentialprovider/gcp"
	// _ "k8s.io/kubernetes/pkg/credentialprovider/rancher"
	"net/http"
	"strings"

	"github.com/appscode/go/log"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/tamalsaha/go-oneliners"
	"k8s.io/api/core/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/kubernetes/pkg/credentialprovider"
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

// ref: https://github.com/kubernetes/kubernetes/blob/release-1.9/pkg/kubelet/kuberuntime/kuberuntime_image.go#L29

// PullImage pulls an image from the network to local storage using the supplied secrets if necessary.
func PullImage(img string, pullSecrets []v1.Secret) (string, error) {
	repoToPull, tag, _, err := parsers.ParseImageName(img)
	if err != nil {
		return "", err
	}

	parts := strings.SplitN(repoToPull, "/", 2)
	regURL := parts[0]
	repo := parts[1]
	fmt.Println(regURL, repo, tag)

	keyring, err := credentialprovider.MakeDockerKeyring(pullSecrets, credentialprovider.NewDockerKeyring())
	if err != nil {
		return "", err
	}

	creds, withCredentials := keyring.Lookup(repoToPull)
	if !withCredentials {
		glog.V(3).Infof("Pulling image %q without credentials", img)

		fmt.Printf("Pull image %q auth %v", img, nil)
		mf, err := PullManifest(repo, tag, nil)
		fmt.Println(mf, err)
		return "imageRef", nil
	}

	var pullErrs []error
	for _, currentCreds := range creds {
		authConfig := credentialprovider.LazyProvide(currentCreds)
		auth := &AuthConfig{
			Username:      authConfig.Username,
			Password:      authConfig.Password,
			Auth:          authConfig.Auth,
			ServerAddress: authConfig.ServerAddress,
			IdentityToken: authConfig.IdentityToken,
			RegistryToken: authConfig.RegistryToken,
		}

		mf, err := PullManifest(repo, tag, auth)
		fmt.Println(mf, err)

		// pullErrs = append(pullErrs, err)
	}

	return "", utilerrors.NewAggregate(pullErrs)
}

func PullManifest(repo, tag string, auth *AuthConfig) (interface{}, error) {
	hub := &reg.Registry{
		URL: auth.ServerAddress,
		Client: &http.Client{
			Transport: reg.WrapTransport(http.DefaultTransport, auth.ServerAddress, auth.Username, auth.Password),
		},
		Logf: reg.Log,
	}
	mx, err := hub.ManifestVx(repo, tag)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to retrieve manifest for image %s:%s", repo, tag)
	}

	switch manifest := mx.(type) {
	case *manifestV2.DeserializedManifest:
		fmt.Println(manifest)
	case *manifestV1.SignedManifest:
		fmt.Println(manifest)
	}
	return nil, errors.New("unknown manifest type")
}

// AuthConfig contains authorization information for connecting to a registry.
type AuthConfig struct {
	Username      string
	Password      string
	Auth          string
	ServerAddress string
	// IdentityToken is used to authenticate the user and get
	// an access token for the registry.
	IdentityToken string
	// RegistryToken is a bearer token to be sent to a registry
	RegistryToken string
}
