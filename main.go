package main

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	core "k8s.io/api/core/v1"
	// Credential providers
	_ "k8s.io/kubernetes/pkg/credentialprovider/aws"
	_ "k8s.io/kubernetes/pkg/credentialprovider/azure"
	_ "k8s.io/kubernetes/pkg/credentialprovider/gcp"
	"github.com/appscode/go/log"
	manifestV1 "github.com/docker/distribution/manifest/schema1"
	manifestV2 "github.com/docker/distribution/manifest/schema2"
	"github.com/golang/glog"
	reg "github.com/heroku/docker-registry-client/registry"
	"github.com/pkg/errors"
	"k8s.io/api/core/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/kubernetes/pkg/credentialprovider"
	"k8s.io/kubernetes/pkg/util/parsers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	secrets, err := kc.CoreV1().Secrets(metav1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		log.Fatalln(err)
	}

	var pullSecrets []v1.Secret
	for _, sec := range secrets.Items {
		if sec.Type == core.SecretTypeDockerConfigJson || sec.Type == core.SecretTypeDockercfg {
			pullSecrets = append(pullSecrets, sec)
		}
	}

	mf1, err := PullImage("nginx", pullSecrets)
	if err != nil {
		log.Fatalln(err)
	}
	switch manifest := mf1.(type) {
	case *manifestV2.DeserializedManifest:
		fmt.Println("nginx", manifest.Config.Digest)
	case *manifestV1.SignedManifest:
		fmt.Println("nginx", manifest.Name)
	}

	mf2, err := PullImage("k8s.gcr.io/kube-proxy-amd64:v1.10.0", pullSecrets)
	switch manifest := mf2.(type) {
	case *manifestV2.DeserializedManifest:
		fmt.Println("k8s.gcr.io/kube-proxy-amd64:v1.10.0", manifest.Config.Digest)
	case *manifestV1.SignedManifest:
		fmt.Println("k8s.gcr.io/kube-proxy-amd64:v1.10.0", manifest.Name)
	}
}

// ref: https://github.com/kubernetes/kubernetes/blob/release-1.9/pkg/kubelet/kuberuntime/kuberuntime_image.go#L29

// PullImage pulls an image from the network to local storage using the supplied secrets if necessary.
func PullImage(img string, pullSecrets []v1.Secret) (interface{}, error) {
	repoToPull, tag, _, err := parsers.ParseImageName(img)
	if err != nil {
		return nil, err
	}

	parts := strings.SplitN(repoToPull, "/", 2)
	regURL := parts[0]
	repo := parts[1]
	fmt.Println(regURL, repo, tag)

	keyring, err := credentialprovider.MakeDockerKeyring(pullSecrets, credentialprovider.NewDockerKeyring())
	if err != nil {
		return nil, err
	}

	creds, withCredentials := keyring.Lookup(repoToPull)
	if !withCredentials {
		glog.V(3).Infof("Pulling image %q without credentials", img)
		return PullManifest(repo, tag, &AuthConfig{})
	}

	var pullErrs []error
	for _, currentCreds := range creds {
		authConfig := credentialprovider.LazyProvide(currentCreds)
		auth := &AuthConfig{
			Username:      authConfig.Username,
			Password:      authConfig.Password,
			Auth:          authConfig.Auth,
			ServerAddress: authConfig.ServerAddress,
		}

		mf, err := PullManifest(repo, tag, auth)
		if err == nil {
			return mf, nil
		}
		pullErrs = append(pullErrs, err)
	}
	return nil, utilerrors.NewAggregate(pullErrs)
}

func PullManifest(repo, tag string, auth *AuthConfig) (interface{}, error) {
	if auth.ServerAddress == "" {
		auth.ServerAddress = "https://registry-1.docker.io"
	}

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
}
