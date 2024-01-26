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

package k8sclient

import (
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
