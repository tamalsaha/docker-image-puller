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
	"k8s.io/api/core/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/kubernetes/pkg/credentialprovider"
	"k8s.io/kubernetes/pkg/util/parsers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/url"
	"flag"
)

// k8s.gcr.io/kube-proxy-amd64:v1.10.0
// nginx
// gcloud docker -- pull gcr.io/tigerworks-kube/glusterd:3.7-3
func main() {
	img := flag.String("image", "", "Name of docker image as used in a Kubernetes container")
	masterURL := flag.String("master", "", "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	kubeconfigPath := flag.String("kubeconfig", filepath.Join(homedir.HomeDir(), ".kube/config"), "Path to kubeconfig file")
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags(*masterURL, *kubeconfigPath)
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

	mf2, err := PullImage(*img, pullSecrets)
	if err != nil {
		log.Fatalln(err)
	}
	switch manifest := mf2.(type) {
	case *manifestV2.DeserializedManifest:
		data, _ := manifest.MarshalJSON()
		fmt.Println("V2 Manifest:", string(data))
	case *manifestV1.SignedManifest:
		data, _ := manifest.MarshalJSON()
		fmt.Println("V1 Manifest:", string(data))
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

	if regURL == "docker.io" {
		regURL = "https://registry-1.docker.io"
	} else {
		if u2, e2 := url.Parse(regURL); e2 == nil {
			if u2.Scheme == "" {
				u2.Scheme = "https"
			}
			regURL = u2.String()
		}
	}

	keyring, err := credentialprovider.MakeDockerKeyring(pullSecrets, credentialprovider.NewDockerKeyring())
	if err != nil {
		return nil, err
	}

	creds, withCredentials := keyring.Lookup(repoToPull)
	if !withCredentials {
		glog.V(3).Infof("Pulling image %q without credentials", img)
		return PullManifest(repo, tag, &AuthConfig{ServerAddress: regURL})
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
		if auth.ServerAddress == "" {
			auth.ServerAddress = regURL
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
	return hub.ManifestVx(repo, tag)
}

// AuthConfig contains authorization information for connecting to a registry.
type AuthConfig struct {
	Username      string
	Password      string
	Auth          string
	ServerAddress string
}
