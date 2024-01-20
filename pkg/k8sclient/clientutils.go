/*
MIT License

Copyright (c) 2019 Norihiro Seto

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package k8sclient

import (
	"flag"
	"fmt"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"path/filepath"
)

// getConfigFilePath retrieves the kubeconfig file path.
// It checks if the flag is parsed using flag.Parsed() and parses it if not.
// If the HOME environment variable is set, it returns the kubeconfig path as filepath.Join(home, ".kube", "config").
// If the KUBECONFIG environment variable is set, it returns the value of the variable.
// If neither HOME nor KUBECONFIG are set, it returns an empty string.
// Returns a pointer to the kubeconfig path string.
func getConfigFilePath() *string {
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

// NewRESTConfig retrieves the REST configuration for communicating with a Kubernetes cluster.
// It first tries to get the kubeconfig file path by calling getConfigFilePath function.
// If the kubeconfig file path is provided, it builds the config using clientcmd.BuildConfigFromFlags function.
// If the config is nil or an error occurs, it falls back to using rest.InClusterConfig to get the config.
// Returns the config and an error, if any.
func NewRESTConfig() (config *rest.Config, err error) {
	kubeconfig := getConfigFilePath()

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

// NewClientset creates a new instance of *kubernetes.Clientset.
// It calls the function NewClientsetWithRestConfig to get the *kubernetes.Clientset and error.
// Returns the *kubernetes.Clientset and error from NewClientsetWithRestConfig.
func NewClientset() (*kubernetes.Clientset, error) {
	clnt, _, err := NewClientsetWithRestConfig()

	return clnt, err
}

// NewClientsetWithRestConfig initializes a Kubernetes clientset and REST config.
// It first calls NewRESTConfig to retrieve the REST config.
// If an error occurs during the creation of the config, nil values and the error are returned.
// Otherwise, it creates a clientset using the config.
// If an error occurs during the creation of the clientset, nil values and the error are returned.
// Otherwise, it returns the clientset, the config, and any error that occurred during config creation.
// Example usage:
//
//	clnt, config, err := NewClientsetWithRestConfig()
func NewClientsetWithRestConfig() (*kubernetes.Clientset, *rest.Config, error) {
	config, err := NewRESTConfig()

	if err != nil {
		return nil, nil, fmt.Errorf("failed to create a REST config: %w", err)
	}

	clnt, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create a config: %w", err)
	}

	return clnt, config, err
}
