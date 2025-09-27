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

package client

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Options represents the configuration options for a kubernetes client.
type Options struct {
	configFilePath string
}

type kubeconfigSource int

const (
	kubeconfigSourceNone kubeconfigSource = iota
	kubeconfigSourceFlag
	kubeconfigSourceEnv
	kubeconfigSourceDefault
)

func (s kubeconfigSource) String() string {
	switch s {
	case kubeconfigSourceFlag:
		return "--kubeconfig flag"
	case kubeconfigSourceEnv:
		return "KUBECONFIG environment variable"
	case kubeconfigSourceDefault:
		return "default kubeconfig location"
	default:
		return ""
	}
}

func (s kubeconfigSource) explicit() bool {
	return s == kubeconfigSourceFlag || s == kubeconfigSourceEnv
}

// BindFlags adds the "kubeconfig" flag to the given FlagSet.
// It binds the value of the flag to the configFilePath field of the Options struct.
// The flag is used to specify the absolute path to the kubeconfig file.
func (o *Options) BindFlags(fs *flag.FlagSet) {
	fs.StringVar(&o.configFilePath, "kubeconfig", "", "absolute path to the kubeconfig file")
}

// BindPFlags adds the "kubeconfig" flag to the given FlagSet.
// It binds the value of the flag to the configFilePath field of the Options struct.
// The flag is used to specify the absolute path to the kubeconfig file.
func (o *Options) BindPFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.configFilePath, "kubeconfig", "", "absolute path to the kubeconfig file")
	_ = fs.MarkHidden("kubeconfig")
}

// GetConfigFilePath retrieves the kubeconfig file path with security validation.
// It returns the sanitized path, the source of the configuration, and an error if validation fails.
func (o *Options) GetConfigFilePath() (string, kubeconfigSource, error) {
	var (
		path   string
		source kubeconfigSource
	)

	switch {
	case o.configFilePath != "":
		path = o.configFilePath
		source = kubeconfigSourceFlag
	case os.Getenv("KUBECONFIG") != "":
		path = os.Getenv("KUBECONFIG")
		source = kubeconfigSourceEnv
	case os.Getenv("HOME") != "":
		path = filepath.Join(os.Getenv("HOME"), ".kube", "config")
		source = kubeconfigSourceDefault
	default:
		return "", kubeconfigSourceNone, nil
	}

	cleanPath := filepath.Clean(path)

	if strings.Contains(cleanPath, "..") || strings.HasPrefix(cleanPath, "/proc") ||
		strings.HasPrefix(cleanPath, "/sys") {
		if source.explicit() {
			return "", source, fmt.Errorf("kubeconfig path %q from %s failed validation", path, source)
		}

		return "", kubeconfigSourceNone, nil
	}

	return cleanPath, source, nil
}

type contextKey struct{}

// FromContext retrieves the *Options value from the given context.
// If the value exists and is of type *Options, it is returned.
// Otherwise, a new empty *Options is returned.
func FromContext(ctx context.Context) *Options {
	if v, ok := ctx.Value(contextKey{}).(*Options); ok {
		return v
	}

	return &Options{}
}

// WithContext sets the value of the options in the given context.
// It returns a new context with the updated value.
func WithContext(ctx context.Context, opts *Options) context.Context {
	return context.WithValue(ctx, contextKey{}, opts)
}

// NewRESTConfig creates a new Kubernetes REST config based on the provided options.
// It takes an `opts` pointer to an `Options` struct which contains the path to the kubeconfig file.
// If the `opts` contains a non-empty kubeconfig file path,
// it uses `clientcmd.BuildConfigFromFlags` to build the config.
// If the config is not specified or there is an error building it, it falls back to using `rest.InClusterConfig`.
// The function returns the created REST config and an error if there was a failure.
func NewRESTConfig(opts *Options) (*rest.Config, error) {
	kubeconfig, source, err := opts.GetConfigFilePath()
	if err != nil {
		return nil, err
	}

	switch source {
	case kubeconfigSourceFlag, kubeconfigSourceEnv:
		config, buildErr := clientcmd.BuildConfigFromFlags("", kubeconfig)
		if buildErr != nil {
			return nil, fmt.Errorf("failed to load kubeconfig from %s: %w", source, buildErr)
		}
		return config, nil
	case kubeconfigSourceDefault:
		if kubeconfig != "" {
			if config, buildErr := clientcmd.BuildConfigFromFlags("", kubeconfig); buildErr == nil {
				return config, nil
			}
		}
	}

	return rest.InClusterConfig()
}

// NewClientset creates a new Kubernetes clientset.
// It takes an `opts` pointer to an `Options` struct which contains the path to the kubeconfig file.
// It returns a `*kubernetes.Clientset` and an `error` if there was a failure.
// It utilizes the `NewClientsetWithRestConfig` function to create the clientset.
// If there was an error creating the clientset, an error is returned along with `nil` for the clientset.
func NewClientset(opts *Options) (*kubernetes.Clientset, error) {
	clnt, _, err := NewClientsetWithRestConfig(opts)

	return clnt, err
}

// NewClientsetWithRestConfig creates a new Kubernetes clientset and REST config.
// It takes an `opts` pointer to an `Options` struct which contains the path to the kubeconfig file.
// It returns a `*kubernetes.Clientset`, `*rest.Config`, and an `error` if there was a failure.
// It utilizes the `NewRESTConfig` function to create the REST config,
// then uses the REST config to create the clientset.
// If there was an error creating the REST config or the clientset,
// an error is returned along with `nil` for the clientset and config.
func NewClientsetWithRestConfig(opts *Options) (*kubernetes.Clientset, *rest.Config, error) {
	config, err := NewRESTConfig(opts)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to create a REST config: %w", err)
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create a config: %w", err)
	}

	return client, config, err
}
