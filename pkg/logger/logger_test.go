package logger

import (
	"context"
	"flag"
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func TestInitLogger(t *testing.T) {
	// Keep the original flag set
	originalCommandLine := flag.CommandLine
	// Create a new flag set for testing
	fs := flag.NewFlagSet(t.Name(), flag.ContinueOnError)
	// Bind zap options to the test flag set
	opts := zap.Options{
		Development: false,
	}
	opts.BindFlags(fs)

	// Restore the original flag set after test
	defer func() {
		flag.CommandLine = originalCommandLine
	}()

	logger := InitLogger()
	assert.NotNil(t, logger)
}

func TestInitLoggerWithParsedFlags(t *testing.T) {
	originalCommandLine := flag.CommandLine
	defer func() { flag.CommandLine = originalCommandLine }()

	fs := flag.NewFlagSet(t.Name(), flag.ContinueOnError)
	if err := fs.Parse([]string{}); err != nil {
		t.Fatalf("failed to parse flagset: %v", err)
	}
	flag.CommandLine = fs

	logger := InitLogger()
	assert.NotNil(t, logger)
}

func TestFromContext(t *testing.T) {
	ctx := context.Background()
	logger := zap.New()

	// Test without key-values
	ctx = WithContext(ctx, logger)
	loggerFromCtx := FromContext(ctx)
	assert.NotNil(t, loggerFromCtx)

	// Test with key-values
	loggerWithKV := FromContext(ctx, "key", "value")
	assert.NotNil(t, loggerWithKV)
}

func TestWithContext(t *testing.T) {
	ctx := context.Background()
	logger := zap.New()

	newCtx := WithContext(ctx, logger)
	assert.NotNil(t, newCtx)

	// Verify logger can be retrieved
	loggerFromCtx := FromContext(newCtx)
	assert.NotNil(t, loggerFromCtx)
}

func TestInitCmdLogger(t *testing.T) {
	rootCmd := &cobra.Command{
		Use: "test",
		Run: func(cmd *cobra.Command, args []string) {},
	}

	InitCmdLogger(rootCmd)

	// Verify flags are bound
	assert.NotNil(t, rootCmd.PersistentFlags().Lookup("zap-devel"))
	assert.NotNil(t, rootCmd.PersistentFlags().Lookup("zap-encoder"))
	assert.NotNil(t, rootCmd.PersistentFlags().Lookup("zap-log-level"))

	// Test PreRun hook
	ctx := context.Background()
	rootCmd.SetContext(ctx)
	if rootCmd.PersistentPreRun != nil {
		rootCmd.PersistentPreRun(rootCmd, []string{})
		logger := FromContext(rootCmd.Context())
		assert.NotNil(t, logger)
	}

	// Test PostRun hook
	if rootCmd.PersistentPostRun != nil {
		rootCmd.PersistentPostRun(rootCmd, []string{})
	}
}

func TestMakeCmdValue(t *testing.T) {
	// Create a command hierarchy
	rootCmd := &cobra.Command{Use: "root"}
	subCmd := &cobra.Command{Use: "sub"}
	subSubCmd := &cobra.Command{Use: "subsub"}

	rootCmd.AddCommand(subCmd)
	subCmd.AddCommand(subSubCmd)

	tests := []struct {
		name     string
		cmd      *cobra.Command
		expected string
	}{
		{
			name:     "root command",
			cmd:      rootCmd,
			expected: "root",
		},
		{
			name:     "sub command",
			cmd:      subCmd,
			expected: "root.sub",
		},
		{
			name:     "subsub command",
			cmd:      subSubCmd,
			expected: "root.sub.subsub",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := makeCmdValue(tt.cmd)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMakeCommandLine(t *testing.T) {
	// Save original args and restore after test
	originalArgs := os.Args
	defer func() {
		os.Args = originalArgs
	}()

	// Set test args
	os.Args = []string{"test-binary"}

	fs := pflag.NewFlagSet("test", pflag.ExitOnError)
	fs.String("test-flag", "default", "test flag")
	_ = fs.Set("test-flag", "value")

	result := makeCommandLine(fs)

	assert.Contains(t, result, "test-binary")
	assert.Contains(t, result, "--test-flag=value")
}

func TestBindPFlags(t *testing.T) {
	opts := &zap.Options{
		Development: false,
	}
	fs := pflag.NewFlagSet("test", pflag.ExitOnError)

	bindPFlags(opts, fs)

	// Verify all flags are created and hidden
	flags := []string{
		"zap-devel",
		"zap-encoder",
		"zap-log-level",
		"zap-stacktrace-level",
		"zap-time-encoding",
	}

	for _, flag := range flags {
		f := fs.Lookup(flag)
		assert.NotNil(t, f)
		// hiddenフラグの確認方法を修正
		hidden := f.Hidden
		assert.True(t, hidden, "flag %s should be hidden", flag)
	}
}

func TestSetupLogger(t *testing.T) {
	// Create a command hierarchy
	rootCmd := &cobra.Command{Use: "root"}
	subCmd := &cobra.Command{Use: "sub"}
	rootCmd.AddCommand(subCmd)

	// Save original args and restore after test
	originalArgs := os.Args
	defer func() {
		os.Args = originalArgs
	}()

	// Set test args with zap options
	os.Args = []string{"test-binary", "--zap-log-level=debug"}

	opts := &zap.Options{
		Development: false,
	}

	// Test with root command
	ctx := context.Background()
	rootCmd.SetContext(ctx)
	setupLogger(opts, rootCmd)
	assert.NotNil(t, rootCmd.Context())

	// Test with sub command
	subCmd.SetContext(ctx)
	setupLogger(opts, subCmd)
	assert.NotNil(t, subCmd.Context())
}
