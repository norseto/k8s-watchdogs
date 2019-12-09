package k8sutils

// Common client package.

import (
	"flag"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"path/filepath"
)

// NewClientset returns current Clientset
func NewClientset() (clientset *kubernetes.Clientset, err error) {
	var kubeconfig *string
	var config *rest.Config

	if home := os.Getenv("HOME"); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"),
			"(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	if *kubeconfig != "" {
		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
		if err != nil {
			// Use in-cluster configuration
			config, err = rest.InClusterConfig()
		}
	}

	if err == nil {
		// Create clientset from configuration
		clientset, err = kubernetes.NewForConfig(config)
	}
	return
}
