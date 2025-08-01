/*
MIT License

Copyright (c) 2024 Norihiro Seto

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
	"os"
	"path/filepath"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var backup *flag.FlagSet

func init() {
	backup = flag.CommandLine
}

func TestGetKubeconfig(t *testing.T) {
	tests := []struct {
		name     string
		setup    func()
		teardown func()
		expected string
	}{
		{
			name: "HOME path exists",
			setup: func() {
				_ = os.Setenv("HOME", "/home/mock")
			},
			teardown: func() {
				_ = os.Unsetenv("HOME")
			},
			expected: "/home/mock/.kube/config",
		},
		{
			name: "KUBECONFIG set",
			setup: func() {
				_ = os.Unsetenv("HOME")
				_ = os.Setenv("KUBECONFIG", "/home/mock/.kube/config2")
			},
			teardown: func() {
				_ = os.Unsetenv("KUBECONFIG")
			},
			expected: "/home/mock/.kube/config2",
		},
		{
			name: "Neither HOME, nor KUBECONFIG are set",
			setup: func() {
				_ = os.Unsetenv("HOME")
				_ = os.Unsetenv("KUBECONFIG")
			},
			teardown: func() {
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &Options{}
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
			opts.BindFlags(flag.CommandLine)
			flag.Parse()
			tt.setup()
			defer func() {
				tt.teardown()
				flag.CommandLine = backup
			}()
			result := opts.GetConfigFilePath()
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestFromContext(t *testing.T) {
	// Test when context contains Options value
	expectedOptions := &Options{} // Replace with the actual initialization of Options struct
	ctx := context.WithValue(context.Background(), contextKey{}, expectedOptions)
	options := FromContext(ctx)
	if options != expectedOptions {
		t.Errorf("Expected options to be %v, but got %v", expectedOptions, options)
	}

	// Test when context does not contain Options value
	ctx = context.Background()
	options = FromContext(ctx)
	if options == nil {
		t.Errorf("Expected options to be non-nil, but got nil")
	}
}

func TestWithContext(t *testing.T) {
	// Positive test case
	opts := &Options{} // Fill in the necessary options
	ctx := context.Background()
	ctx = WithContext(ctx, opts)

	// Verify that the options are correctly set in the context
	if value := ctx.Value(contextKey{}); value != opts {
		t.Errorf("Expected options %v, but got %v", opts, value)
	}

	// Negative test case
	// Verify that the options are not set in the context if nil is passed
	ctx = context.Background()
	ctx = WithContext(ctx, nil)
	if value := ctx.Value(contextKey{}); value != (*Options)(nil) {
		t.Errorf("Expected nil options, but got %v", value)
	}
}

func TestNewRESTConfig(t *testing.T) {
	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "config")
	kubeconfigContent := `apiVersion: v1
kind: Config
clusters:
- name: test
  cluster:
    server: https://127.0.0.1
    insecure-skip-tls-verify: true
users:
- name: test
  user:
    token: dummy
contexts:
- name: test
  context:
    cluster: test
    user: test
current-context: test`
	err := os.WriteFile(kubeconfigPath, []byte(kubeconfigContent), 0600)
	assert.NoError(t, err)

	tests := []struct {
		name      string
		path      string
		setup     func()
		teardown  func()
		wantHost  string
		expectErr bool
	}{
		{
			name:     "valid path",
			path:     kubeconfigPath,
			setup:    func() {},
			teardown: func() {},
			wantHost: "https://127.0.0.1",
		},
		{
			name:      "invalid path",
			path:      filepath.Join(tmpDir, "noexist"),
			setup:     func() {},
			teardown:  func() {},
			expectErr: true,
		},
		{
			name: "fallback to incluster",
			path: filepath.Join(tmpDir, "missing"),
			setup: func() {
				t.Setenv("KUBERNETES_SERVICE_HOST", "10.0.0.1")
				t.Setenv("KUBERNETES_SERVICE_PORT", "6443")
				saDir := "/var/run/secrets/kubernetes.io/serviceaccount"
				_ = os.MkdirAll(saDir, 0755)
				_ = os.WriteFile(filepath.Join(saDir, "token"), []byte("dummy"), 0644)
				_ = os.WriteFile(filepath.Join(saDir, "ca.crt"), []byte("dummy"), 0644)
			},
			teardown: func() {
				saDir := "/var/run/secrets/kubernetes.io/serviceaccount"
				_ = os.Remove(filepath.Join(saDir, "token"))
				_ = os.Remove(filepath.Join(saDir, "ca.crt"))
			},
			wantHost: "https://10.0.0.1:6443",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			defer tt.teardown()
			opts := &Options{configFilePath: tt.path}
			cfg, err := NewRESTConfig(opts)
			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, cfg)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, cfg)
			assert.Equal(t, tt.wantHost, cfg.Host)
		})
	}
}

func TestNewClientsetWithRestConfig(t *testing.T) {
	dummyClient := &kubernetes.Clientset{}
	dummyCfg := &rest.Config{Host: "dummy"}

	tests := []struct {
		name      string
		cfgErr    error
		clientErr error
		wantErr   bool
	}{
		{
			name:    "success",
			wantErr: false,
		},
		{
			name:    "config error",
			cfgErr:  assert.AnError,
			wantErr: true,
		},
		{
			name:      "client error",
			clientErr: assert.AnError,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.NewPatches()
			defer patches.Reset()

			patches.ApplyFunc(NewRESTConfig, func(*Options) (*rest.Config, error) {
				if tt.cfgErr != nil {
					return nil, tt.cfgErr
				}
				return dummyCfg, nil
			})
			patches.ApplyFunc(kubernetes.NewForConfig, func(c *rest.Config) (*kubernetes.Clientset, error) {
				if tt.clientErr != nil {
					return nil, tt.clientErr
				}
				assert.Equal(t, dummyCfg, c)
				return dummyClient, nil
			})

			cl, cfg, err := NewClientsetWithRestConfig(&Options{})
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, cl)
				assert.Nil(t, cfg)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, dummyClient, cl)
			assert.Equal(t, dummyCfg, cfg)
		})
	}
}

func TestNewClientsetDelegation(t *testing.T) {
	patches := gomonkey.ApplyFunc(NewClientsetWithRestConfig, func(opts *Options) (*kubernetes.Clientset, *rest.Config, error) {
		return &kubernetes.Clientset{}, &rest.Config{}, nil
	})
	defer patches.Reset()

	cl, err := NewClientset(&Options{})
	assert.NoError(t, err)
	assert.NotNil(t, cl)
}

func TestNewRESTConfigCases(t *testing.T) {
	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "kube")
	kubeconfigContent := `apiVersion: v1
kind: Config
clusters:
- name: test
  cluster:
    server: https://127.0.0.1
    insecure-skip-tls-verify: true
users:
- name: test
  user:
    token: dummy
contexts:
- name: test
  context:
    cluster: test
    user: test
current-context: test`
	err := os.WriteFile(kubeconfigPath, []byte(kubeconfigContent), 0600)
	assert.NoError(t, err)

	saDir := "/var/run/secrets/kubernetes.io/serviceaccount"
	_ = os.MkdirAll(saDir, 0755)
	_ = os.WriteFile(filepath.Join(saDir, "token"), []byte("d"), 0644)
	_ = os.WriteFile(filepath.Join(saDir, "ca.crt"), []byte("d"), 0644)

	tests := []struct {
		name      string
		path      string
		setup     func()
		wantHost  string
		expectErr bool
	}{
		{
			name:     "valid",
			path:     kubeconfigPath,
			setup:    func() {},
			wantHost: "",
		},
		{
			name:      "invalid",
			path:      filepath.Join(tmpDir, "dummy"),
			setup:     func() {},
			expectErr: true,
		},
		{
			name: "incluster",
			path: "",
			setup: func() {
				t.Setenv("KUBERNETES_SERVICE_HOST", "127.0.0.2")
				t.Setenv("KUBERNETES_SERVICE_PORT", "6443")
			},
			wantHost: "https://127.0.0.2:6443",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			opts := &Options{configFilePath: tt.path}
			cfg, err := NewRESTConfig(opts)
			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, cfg)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, cfg)
			if tt.wantHost != "" {
				assert.Equal(t, tt.wantHost, cfg.Host)
			}
		})
	}
}

func TestNewClientsetWithRestConfigMock(t *testing.T) {
	dummyClient := &kubernetes.Clientset{}
	dummyCfg := &rest.Config{}

	tests := []struct {
		name      string
		cfgErr    error
		clientErr error
		wantErr   bool
	}{
		{name: "ok"},
		{name: "cfg", cfgErr: assert.AnError, wantErr: true},
		{name: "client", clientErr: assert.AnError, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.NewPatches()
			defer patches.Reset()

			patches.ApplyFunc(NewRESTConfig, func(*Options) (*rest.Config, error) {
				if tt.cfgErr != nil {
					return nil, tt.cfgErr
				}
				return dummyCfg, nil
			})
			patches.ApplyFunc(kubernetes.NewForConfig, func(c *rest.Config) (*kubernetes.Clientset, error) {
				if tt.clientErr != nil {
					return nil, tt.clientErr
				}
				return dummyClient, nil
			})

			cl, cfg, err := NewClientsetWithRestConfig(&Options{})
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, cl)
				assert.Nil(t, cfg)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, dummyClient, cl)
			assert.Equal(t, dummyCfg, cfg)
		})
	}
}

func TestNewClientsetCallsWrapper(t *testing.T) {
	patches := gomonkey.ApplyFunc(NewClientsetWithRestConfig, func(*Options) (*kubernetes.Clientset, *rest.Config, error) {
		return &kubernetes.Clientset{}, &rest.Config{}, nil
	})
	defer patches.Reset()

	cl, err := NewClientset(&Options{})
	assert.NoError(t, err)
	assert.NotNil(t, cl)
}
