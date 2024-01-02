package k8sutils

import (
	"flag"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"path/filepath"
)

// getKubeconfig retrieves the path to the kubeconfig file.
func getKubeconfig() *string {
	if !flag.Parsed() {
		flag.Parse()
	}

	if home := os.Getenv("HOME"); home != "" {
		return flag.String("kubeconfig", filepath.Join(home, ".kube", "config"),
			"(optional) absolute path to the kubeconfig file")
	} else if envVar := os.Getenv("KUBECONFIG"); envVar != "" {
		return &envVar
	}
	return flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
}

// NewRESTConfig returns REST client configuration for Kubernetes
func NewRESTConfig() (config *rest.Config, err error) {
	kubeconfig := getKubeconfig()

	if !flag.Parsed() {
		flag.Parse()
	}

	if *kubeconfig != "" {
		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
	}

	if config == nil || err != nil {
		config, err = rest.InClusterConfig()
	}
	return
}

// NewClientset returns current Clientset
func NewClientset() (*kubernetes.Clientset, error) {
	clnt, _, err := NewClientsetWithRestConfig()

	return clnt, err
}

// NewClientsetWithRestConfig returns current Clientset and Rest configuration
func NewClientsetWithRestConfig() (*kubernetes.Clientset, *rest.Config, error) {
	config, err := NewRESTConfig()

	if err != nil {
		return nil, nil, err
	}

	clnt, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, err
	}

	return clnt, config, err
}
