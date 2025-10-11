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
	"errors"
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

var (
	buildConfigFromFlags  = clientcmd.BuildConfigFromFlags
	inClusterConfig       = rest.InClusterConfig
	newClientsetForConfig = kubernetes.NewForConfig
)

// Options represents the configuration options for a kubernetes client.
type Options struct {
	configFilePath      string
	allowedPathPrefixes []string
	deniedPathPrefixes  []string
}

// SetPathPrefixAllowList sets the allow list for kubeconfig path prefixes.
func (o *Options) SetPathPrefixAllowList(prefixes []string) {
	o.allowedPathPrefixes = clonePrefixes(prefixes)
}

// SetPathPrefixDenyList sets the deny list for kubeconfig path prefixes.
func (o *Options) SetPathPrefixDenyList(prefixes []string) {
	o.deniedPathPrefixes = clonePrefixes(prefixes)
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

	absPath, err := filepath.Abs(path)
	if err != nil {
		return handleImplicitPathError(path, source, err)
	}

	resolvedPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) && !source.explicit() {
			return "", kubeconfigSourceNone, nil
		}
		return handleImplicitPathError(path, source, err)
	}

	resolvedPath = filepath.Clean(resolvedPath)

	if strings.Contains(resolvedPath, "..") {
		return handleImplicitPathError(path, source, fmt.Errorf("path traversal detected"))
	}

	info, err := os.Stat(resolvedPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) && !source.explicit() {
			return "", kubeconfigSourceNone, nil
		}
		return handleImplicitPathError(path, source, err)
	}

	if !info.Mode().IsRegular() {
		return handleImplicitPathError(path, source, fmt.Errorf("kubeconfig path %q must be a regular file", resolvedPath))
	}

	if !o.isAllowedPath(resolvedPath) {
		return handleImplicitPathError(
			path,
			source,
			fmt.Errorf("kubeconfig path %q is not under an allowed prefix", resolvedPath),
		)
	}

	if o.isDeniedPath(resolvedPath) {
		return handleImplicitPathError(path, source, fmt.Errorf("kubeconfig path %q is under a denied prefix", resolvedPath))
	}

	return resolvedPath, source, nil
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

func handleImplicitPathError(
	originalPath string,
	source kubeconfigSource,
	err error,
) (string, kubeconfigSource, error) {
	if source.explicit() {
		return "", source, fmt.Errorf("kubeconfig path %q from %s failed validation: %w", originalPath, source, err)
	}

	return "", kubeconfigSourceNone, nil
}

func (o *Options) allowedPrefixes() []string {
	if len(o.allowedPathPrefixes) == 0 {
		return nil
	}

	return o.allowedPathPrefixes
}

func (o *Options) deniedPrefixes() []string {
	if len(o.deniedPathPrefixes) == 0 {
		return defaultDeniedPathPrefixes
	}

	return o.deniedPathPrefixes
}

func (o *Options) isAllowedPath(path string) bool {
	prefixes := o.allowedPrefixes()
	if len(prefixes) == 0 {
		return true
	}

	for _, prefix := range prefixes {
		if hasPathPrefix(path, prefix) {
			return true
		}
	}

	return false
}

func (o *Options) isDeniedPath(path string) bool {
	for _, prefix := range o.deniedPrefixes() {
		if hasPathPrefix(path, prefix) {
			return true
		}
	}

	return false
}

func hasPathPrefix(path, prefix string) bool {
	if prefix == "" {
		return false
	}

	cleanedPath := filepath.Clean(path)
	cleanedPrefix := filepath.Clean(prefix)

	if cleanedPath == cleanedPrefix {
		return true
	}

	sep := string(os.PathSeparator)
	if !strings.HasSuffix(cleanedPrefix, sep) {
		cleanedPrefix += sep
	}

	return strings.HasPrefix(cleanedPath, cleanedPrefix)
}

func clonePrefixes(prefixes []string) []string {
	if len(prefixes) == 0 {
		return nil
	}

	cloned := make([]string, len(prefixes))
	copy(cloned, prefixes)
	return cloned
}

var defaultDeniedPathPrefixes = []string{"/proc", "/sys"}

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
		config, buildErr := buildConfigFromFlags("", kubeconfig)
		if buildErr != nil {
			return nil, fmt.Errorf("failed to load kubeconfig from %s: %w", source, buildErr)
		}
		return config, nil
	case kubeconfigSourceDefault:
		if kubeconfig != "" {
			if config, buildErr := buildConfigFromFlags("", kubeconfig); buildErr == nil {
				return config, nil
			}
		}
	}

	return inClusterConfig()
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

	client, err := newClientsetForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create a config: %w", err)
	}

	return client, config, err
}
