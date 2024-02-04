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
	"testing"
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
				_ = os.Unsetenv("Home")
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
