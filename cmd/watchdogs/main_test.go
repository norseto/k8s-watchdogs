package main

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/spf13/cobra"
)

func TestRunSuccess(t *testing.T) {
	origFactory := rootCmdFactory
	rootCmdFactory = func() *cobra.Command {
		cmd := &cobra.Command{Use: "test", Run: func(cmd *cobra.Command, args []string) {}}
		cmd.SetContext(context.Background())
		return cmd
	}
	t.Cleanup(func() { rootCmdFactory = origFactory })

	if code := run(); code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestRunFailure(t *testing.T) {
	origFactory := rootCmdFactory
	rootCmdFactory = func() *cobra.Command {
		cmd := &cobra.Command{Use: "test", RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("boom")
		}}
		cmd.SetContext(context.Background())
		return cmd
	}
	t.Cleanup(func() { rootCmdFactory = origFactory })

	if code := run(); code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
}

func TestMainUsesExitFunc(t *testing.T) {
	origFactory := rootCmdFactory
	rootCmdFactory = func() *cobra.Command {
		cmd := &cobra.Command{Use: "test", Run: func(cmd *cobra.Command, args []string) {}}
		cmd.SetContext(context.Background())
		return cmd
	}
	t.Cleanup(func() { rootCmdFactory = origFactory })

	origExit := exitFunc
	defer func() { exitFunc = origExit }()

	var captured int
	exitFunc = func(code int) {
		captured = code
	}

	main()

	if captured != 0 {
		t.Fatalf("expected exit code 0, got %d", captured)
	}
}

func TestNewRootCmd(t *testing.T) {
	cmd := NewRootCmd()
	if cmd.Use != "watchdogs" {
		t.Fatalf("expected use watchdogs, got %s", cmd.Use)
	}
	if len(cmd.Commands()) == 0 {
		t.Fatalf("expected subcommands to be registered")
	}
	if cmd.Context() == nil {
		t.Fatalf("expected context to be configured")
	}
}

func TestRootCmdRunUsage(t *testing.T) {
	cmd := NewRootCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	cmd.Run(cmd, nil)

	if buf.Len() == 0 {
		t.Fatalf("expected usage output")
	}
}

func TestRootCmdPersistentHooks(t *testing.T) {
	cmd := NewRootCmd()
	if cmd.PersistentPreRun == nil || cmd.PersistentPostRun == nil {
		t.Fatalf("expected persistent hooks to be configured")
	}

	cmd.PersistentPreRun(cmd, nil)
	cmd.PersistentPostRun(cmd, nil)
}
