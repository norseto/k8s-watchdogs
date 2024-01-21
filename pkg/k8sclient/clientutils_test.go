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
