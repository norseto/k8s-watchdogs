package main

import (
	"bytes"
	"slices"
	"strings"
	"testing"

	watchdogs "github.com/norseto/k8s-watchdogs"
)

// contains reports whether s is in the list.
func contains(list []string, s string) bool {
	return slices.Contains(list, s)
}

func TestNewRootCmd(t *testing.T) {
	r := NewRootCmd()
	// Check basic metadata
	if r.Use != "watchdogs" {
		t.Errorf("Expected Use 'watchdogs', got '%s'", r.Use)
	}
	if r.Short != "Kubernetes watchdogs utilities" {
		t.Errorf("Unexpected Short description: %s", r.Short)
	}
	if !strings.HasPrefix(r.Long, "Kubernetes utilities") {
		t.Errorf("Unexpected Long description: %s", r.Long)
	}
	// Check subcommands
	expected := []string{"clean-evicted", "delete-oldest", "rebalance-pods", "restart-deploy", "restart-sts", "version"}
	cmds := r.Commands()
	if len(cmds) != len(expected) {
		t.Errorf("Expected %d subcommands, got %d", len(expected), len(cmds))
	}
	for _, name := range expected {
		if !contains(
			func() []string {
				var list []string
				for _, c := range cmds {
					list = append(list, c.Name())
				}
				return list
			}(), name) {
			t.Errorf("Missing subcommand: %s", name)
		}
	}
}

func TestRootCmd_NoArgs(t *testing.T) {
	root := NewRootCmd()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{})
	err := root.Execute()
	if err != nil {
		t.Fatalf("Expected no error on Execute with no args, got %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Usage:") {
		t.Errorf("Expected usage output, got: %s", out)
	}
}

func TestRootCmd_HelpFlag(t *testing.T) {
	root := NewRootCmd()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"--help"})
	err := root.Execute()
	if err != nil {
		t.Fatalf("Expected no error for help flag, got %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Usage:") {
		t.Errorf("Expected help output, got: %s", out)
	}
}

func TestVersionCmd(t *testing.T) {
	root := NewRootCmd()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"version"})
	err := root.Execute()
	if err != nil {
		t.Fatalf("Expected no error on version, got %v", err)
	}
	out := buf.String()
	// Version line
	if !strings.Contains(out, watchdogs.RELEASE_VERSION) {
		t.Errorf("Expected version %s in output, got: %s", watchdogs.RELEASE_VERSION, out)
	}
	// GitVersion line (may be empty but label should be present)
	if !strings.Contains(out, "GitVersion:") {
		t.Errorf("Expected GitVersion label in output, got: %s", out)
	}
}

func TestUnknownCmd(t *testing.T) {
	root := NewRootCmd()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"no-such-cmd"})
	err := root.Execute()
	if err == nil {
		t.Fatal("Expected error for unknown command, got nil")
	}
	errMsg := err.Error()
	if !strings.Contains(errMsg, "unknown command") {
		t.Errorf("Expected 'unknown command' error, got: %s", errMsg)
	}
}
