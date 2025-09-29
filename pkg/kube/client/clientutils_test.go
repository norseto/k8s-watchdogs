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

	"github.com/spf13/pflag"
)

func TestBindFlagsAndPFlags(t *testing.T) {
	t.Run("FlagSet", func(t *testing.T) {
		tmpFile := filepath.Join(t.TempDir(), "config")
		if err := os.WriteFile(tmpFile, []byte("apiVersion: v1\n"), 0o600); err != nil {
			t.Fatalf("failed to write temp kubeconfig: %v", err)
		}

		opts := &Options{}
		fs := flag.NewFlagSet("test", flag.ContinueOnError)
		opts.BindFlags(fs)

		if err := fs.Parse([]string{"--kubeconfig=" + tmpFile}); err != nil {
			t.Fatalf("failed to parse flags: %v", err)
		}

		if opts.configFilePath != tmpFile {
			t.Fatalf("expected configFilePath %q, got %q", tmpFile, opts.configFilePath)
		}
	})

	t.Run("PFlagSet", func(t *testing.T) {
		tmpFile := filepath.Join(t.TempDir(), "config")
		if err := os.WriteFile(tmpFile, []byte("apiVersion: v1\n"), 0o600); err != nil {
			t.Fatalf("failed to write temp kubeconfig: %v", err)
		}

		opts := &Options{}
		fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
		opts.BindPFlags(fs)

		if err := fs.Parse([]string{"--kubeconfig=" + tmpFile}); err != nil {
			t.Fatalf("failed to parse flags: %v", err)
		}

		if opts.configFilePath != tmpFile {
			t.Fatalf("expected configFilePath %q, got %q", tmpFile, opts.configFilePath)
		}

		flag := fs.Lookup("kubeconfig")
		if flag == nil {
			t.Fatalf("expected kubeconfig flag to be registered")
		}
		if !flag.Hidden {
			t.Fatalf("expected kubeconfig flag to be hidden")
		}
	})
}

var backup *flag.FlagSet

func init() {
	backup = flag.CommandLine
}

func TestGetKubeconfig(t *testing.T) {
	tests := []struct {
		name           string
		setup          func(t *testing.T) string
		expectedSource kubeconfigSource
	}{
		{
			name: "HOME path exists",
			setup: func(t *testing.T) string {
				homeDir := t.TempDir()
				kubeDir := filepath.Join(homeDir, ".kube")
				if err := os.MkdirAll(kubeDir, 0o755); err != nil {
					t.Fatalf("failed to create kube dir: %v", err)
				}
				kubeconfigPath := filepath.Join(kubeDir, "config")
				if err := os.WriteFile(kubeconfigPath, []byte("apiVersion: v1\n"), 0o600); err != nil {
					t.Fatalf("failed to create kubeconfig: %v", err)
				}
				t.Setenv("HOME", homeDir)
				t.Setenv("KUBECONFIG", "")
				return kubeconfigPath
			},
			expectedSource: kubeconfigSourceDefault,
		},
		{
			name: "HOME path without kubeconfig",
			setup: func(t *testing.T) string {
				homeDir := t.TempDir()
				t.Setenv("HOME", homeDir)
				t.Setenv("KUBECONFIG", "")
				return ""
			},
			expectedSource: kubeconfigSourceNone,
		},
		{
			name: "KUBECONFIG set",
			setup: func(t *testing.T) string {
				t.Setenv("HOME", "")
				kubeconfigPath := createTempKubeconfig(t)
				t.Setenv("KUBECONFIG", kubeconfigPath)
				return kubeconfigPath
			},
			expectedSource: kubeconfigSourceEnv,
		},
		{
			name: "Neither HOME, nor KUBECONFIG are set",
			setup: func(t *testing.T) string {
				t.Setenv("HOME", "")
				t.Setenv("KUBECONFIG", "")
				return ""
			},
			expectedSource: kubeconfigSourceNone,
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
			expectedPath := tt.setup(t)
			defer func() {
				flag.CommandLine = backup
			}()
			result, source, err := opts.GetConfigFilePath()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if expectedPath != "" && result != expectedPath {
				t.Errorf("expected %s, got %s", expectedPath, result)
			}
			if expectedPath == "" && result != "" {
				t.Errorf("expected empty result, got %s", result)
			}
			if source != tt.expectedSource {
				t.Errorf("expected source %v, got %v", tt.expectedSource, source)
			}
		})
	}

	t.Run("HomeWithoutKubeconfigReturnsNone", func(t *testing.T) {
		homeDir := t.TempDir()
		t.Setenv("HOME", homeDir)
		t.Setenv("KUBECONFIG", "")

		opts := &Options{}
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
		defer func() {
			flag.CommandLine = backup
		}()
		opts.BindFlags(flag.CommandLine)
		if err := flag.CommandLine.Parse([]string{}); err != nil {
			t.Fatalf("failed to parse flags: %v", err)
		}

		path, source, err := opts.GetConfigFilePath()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if path != "" {
			t.Fatalf("expected empty path, got %q", path)
		}
		if source != kubeconfigSourceNone {
			t.Fatalf("expected kubeconfigSourceNone, got %v", source)
		}
	})
}

func TestGetKubeconfigValidationError(t *testing.T) {
	dir := t.TempDir()

	opts := &Options{configFilePath: dir}

	resolved, source, err := opts.GetConfigFilePath()
	if err == nil {
		t.Fatalf("expected validation error, got nil")
	}
	if resolved != "" {
		t.Fatalf("expected empty resolved path, got %v", resolved)
	}
	if source != kubeconfigSourceFlag {
		t.Fatalf("expected source kubeconfigSourceFlag, got %v", source)
	}
	if !strings.Contains(err.Error(), "must be a regular file") {
		t.Fatalf("expected error to mention regular file requirement, got %v", err)
	}
	if !strings.Contains(err.Error(), "--kubeconfig flag") {
		t.Fatalf("expected error to mention flag source, got %v", err)
	}
}

func TestGetKubeconfigPathTraversal(t *testing.T) {
	tempDir := t.TempDir()
	sneakyDir := filepath.Join(tempDir, "nested", "..sneaky")
	if err := os.MkdirAll(sneakyDir, 0o755); err != nil {
		t.Fatalf("failed to create sneaky directory: %v", err)
	}

	kubeconfigPath := filepath.Join(sneakyDir, "config")
	if err := os.WriteFile(kubeconfigPath, []byte("apiVersion: v1\n"), 0o600); err != nil {
		t.Fatalf("failed to create kubeconfig: %v", err)
	}

	opts := &Options{configFilePath: kubeconfigPath}
	if _, _, err := opts.GetConfigFilePath(); err == nil {
		t.Fatalf("expected path traversal validation error, got nil")
	} else {
		if !strings.Contains(err.Error(), "path traversal detected") {
			t.Fatalf("expected path traversal error, got %v", err)
		}
		if !strings.Contains(err.Error(), "--kubeconfig flag") {
			t.Fatalf("expected error to mention flag source, got %v", err)
		}
	}
}

func TestGetKubeconfigSymlinkResolution(t *testing.T) {
	dir := t.TempDir()
	targetDir := filepath.Join(dir, "target")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatalf("failed to create target dir: %v", err)
	}
	targetFile := filepath.Join(targetDir, "config")
	if err := os.WriteFile(targetFile, []byte("apiVersion: v1\n"), 0o600); err != nil {
		t.Fatalf("failed to create kubeconfig: %v", err)
	}

	symlinkPath := filepath.Join(dir, "link")
	if err := os.Symlink(targetFile, symlinkPath); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	opts := &Options{configFilePath: symlinkPath}
	resolved, source, err := opts.GetConfigFilePath()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resolved != targetFile {
		t.Fatalf("expected resolved path %s, got %s", targetFile, resolved)
	}
	if source != kubeconfigSourceFlag {
		t.Fatalf("expected kubeconfigSourceFlag, got %v", source)
	}
}

func TestGetKubeconfigDeniedPrefix(t *testing.T) {
	dir := t.TempDir()
	sensitiveDir := filepath.Join(dir, "sensitive")
	if err := os.MkdirAll(sensitiveDir, 0o755); err != nil {
		t.Fatalf("failed to create sensitive dir: %v", err)
	}
	sensitiveFile := filepath.Join(sensitiveDir, "config")
	if err := os.WriteFile(sensitiveFile, []byte("apiVersion: v1\n"), 0o600); err != nil {
		t.Fatalf("failed to create kubeconfig: %v", err)
	}

	opts := &Options{configFilePath: sensitiveFile}
	opts.SetPathPrefixDenyList([]string{sensitiveDir})

	if _, _, err := opts.GetConfigFilePath(); err == nil {
		t.Fatalf("expected error for denied prefix, got nil")
	}
}

func TestGetKubeconfigDeniedPrefixViaSymlink(t *testing.T) {
	dir := t.TempDir()
	sensitiveDir := filepath.Join(dir, "sensitive")
	if err := os.MkdirAll(sensitiveDir, 0o755); err != nil {
		t.Fatalf("failed to create sensitive dir: %v", err)
	}
	sensitiveFile := filepath.Join(sensitiveDir, "config")
	if err := os.WriteFile(sensitiveFile, []byte("apiVersion: v1\n"), 0o600); err != nil {
		t.Fatalf("failed to create kubeconfig: %v", err)
	}

	allowedDir := filepath.Join(dir, "allowed")
	if err := os.MkdirAll(allowedDir, 0o755); err != nil {
		t.Fatalf("failed to create allowed dir: %v", err)
	}
	symlinkPath := filepath.Join(allowedDir, "config")
	if err := os.Symlink(sensitiveFile, symlinkPath); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	opts := &Options{configFilePath: symlinkPath}
	opts.SetPathPrefixDenyList([]string{sensitiveDir})

	if _, _, err := opts.GetConfigFilePath(); err == nil {
		t.Fatalf("expected error for denied prefix via symlink, got nil")
	}
}

func TestGetKubeconfigAllowedPrefix(t *testing.T) {
	dir := t.TempDir()
	kubeconfigPath := filepath.Join(dir, "config")
	if err := os.WriteFile(kubeconfigPath, []byte("apiVersion: v1\n"), 0o600); err != nil {
		t.Fatalf("failed to create kubeconfig: %v", err)
	}

	opts := &Options{configFilePath: kubeconfigPath}
	opts.SetPathPrefixAllowList([]string{dir})

	resolved, _, err := opts.GetConfigFilePath()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resolved != kubeconfigPath {
		t.Fatalf("expected resolved path %s, got %s", kubeconfigPath, resolved)
	}
}

func TestSetPathPrefixAllowListEmptyEntryIgnored(t *testing.T) {
	opts := &Options{}
	prefixes := []string{""}

	opts.SetPathPrefixAllowList(prefixes)

	if len(opts.allowedPathPrefixes) != 1 {
		t.Fatalf("expected exactly one allowed prefix, got %d", len(opts.allowedPathPrefixes))
	}

	prefixes[0] = "/mutated"

	if opts.allowedPathPrefixes[0] != "" {
		t.Fatalf("expected internal prefixes to remain unchanged, got %q", opts.allowedPathPrefixes[0])
	}

	dir := t.TempDir()
	kubeconfigPath := filepath.Join(dir, "config")
	if err := os.WriteFile(kubeconfigPath, []byte("apiVersion: v1\n"), 0o600); err != nil {
		t.Fatalf("failed to create kubeconfig: %v", err)
	}

	opts.configFilePath = kubeconfigPath

	resolved, source, err := opts.GetConfigFilePath()
	if err == nil {
		t.Fatalf("expected error because empty prefixes are ignored")
	}
	if resolved != "" {
		t.Fatalf("expected empty resolved path, got %s", resolved)
	}
	if source != kubeconfigSourceFlag {
		t.Fatalf("expected kubeconfigSourceFlag, got %v", source)
	}
	if !strings.Contains(err.Error(), "not under an allowed prefix") {
		t.Fatalf("expected error to mention allowed prefix validation, got %v", err)
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
