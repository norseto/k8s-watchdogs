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
	"strings"
	"testing"
)

var backup *flag.FlagSet

func init() {
	backup = flag.CommandLine
}

func TestGetKubeconfig(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T)
		expected string
		source   kubeconfigSource
	}{
		{
			name: "HOME path exists",
			setup: func(t *testing.T) {
				t.Setenv("HOME", "/home/mock")
				t.Setenv("KUBECONFIG", "")
			},
			expected: "/home/mock/.kube/config",
			source:   kubeconfigSourceDefault,
		},
		{
			name: "KUBECONFIG set",
			setup: func(t *testing.T) {
				t.Setenv("HOME", "")
				t.Setenv("KUBECONFIG", "/home/mock/.kube/config2")
			},
			expected: "/home/mock/.kube/config2",
			source:   kubeconfigSourceEnv,
		},
		{
			name: "Neither HOME, nor KUBECONFIG are set",
			setup: func(t *testing.T) {
				t.Setenv("HOME", "")
				t.Setenv("KUBECONFIG", "")
			},
			expected: "",
			source:   kubeconfigSourceNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &Options{}
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
			opts.BindFlags(flag.CommandLine)
			if err := flag.CommandLine.Parse([]string{}); err != nil {
				t.Fatalf("failed to parse flags: %v", err)
			}
			tt.setup(t)
			defer func() {
				flag.CommandLine = backup
			}()
			result, source, err := opts.GetConfigFilePath()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
			if source != tt.source {
				t.Errorf("expected source %v, got %v", tt.source, source)
			}
		})
	}
}

func TestGetKubeconfigValidationError(t *testing.T) {
	opts := &Options{configFilePath: "/proc/1/status"}

	_, _, err := opts.GetConfigFilePath()
	if err == nil {
		t.Fatalf("expected validation error, got nil")
	}
	if !strings.Contains(err.Error(), "--kubeconfig flag") {
		t.Fatalf("expected error to mention flag source, got %v", err)
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

func createTempKubeconfig(t *testing.T) string {
	content := []byte("apiVersion: v1\n" +
		"clusters:\n" +
		"- cluster:\n" +
		"    server: https://127.0.0.1\n" +
		"  name: test\n" +
		"contexts:\n" +
		"- context:\n" +
		"    cluster: test\n" +
		"    user: test\n" +
		"  name: test\n" +
		"current-context: test\n" +
		"users:\n" +
		"- name: test\n" +
		"  user:\n" +
		"    token: dummy\n")
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatalf("failed to write kubeconfig: %v", err)
	}
	return path
}

func TestNewRESTConfig(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		opts := &Options{configFilePath: createTempKubeconfig(t)}
		cfg, err := NewRESTConfig(opts)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if cfg == nil || cfg.Host != "https://127.0.0.1" {
			t.Errorf("unexpected config: %#v", cfg)
		}
	})

	t.Run("invalid explicit path", func(t *testing.T) {
		opts := &Options{configFilePath: "/invalid/path"}
		cfg, err := NewRESTConfig(opts)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "--kubeconfig flag") {
			t.Fatalf("expected error about flag source, got %v", err)
		}
		if cfg != nil {
			t.Errorf("expected nil config, got %#v", cfg)
		}
	})

	t.Run("invalid env path", func(t *testing.T) {
		t.Setenv("KUBECONFIG", "/invalid/env/path")
		opts := &Options{}
		cfg, err := NewRESTConfig(opts)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "KUBECONFIG environment variable") {
			t.Fatalf("expected error about env source, got %v", err)
		}
		if cfg != nil {
			t.Errorf("expected nil config, got %#v", cfg)
		}
	})
}

func TestNewClientset(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		opts := &Options{configFilePath: createTempKubeconfig(t)}
		client, err := NewClientset(opts)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if client == nil {
			t.Fatalf("expected client, got nil")
		}
	})

	t.Run("rest config failure", func(t *testing.T) {
		opts := &Options{configFilePath: "/invalid/path"}
		client, err := NewClientset(opts)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if client != nil {
			t.Errorf("expected nil client, got %#v", client)
		}
	})
}

func TestNewClientsetWithRestConfig(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		opts := &Options{configFilePath: createTempKubeconfig(t)}
		client, cfg, err := NewClientsetWithRestConfig(opts)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if client == nil || cfg == nil {
			t.Fatalf("expected client and config, got %v %v", client, cfg)
		}
	})

	t.Run("rest config failure", func(t *testing.T) {
		opts := &Options{configFilePath: "/invalid/path"}
		client, cfg, err := NewClientsetWithRestConfig(opts)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if client != nil || cfg != nil {
			t.Errorf("expected nil client and config, got %v %v", client, cfg)
		}
	})
}
