package logger

import (
	"context"
	"flag"
	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	clog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// InitLogger initializes a logger using default configuration options.
// The logger is configured based on command line flags.
//
// It returns a logr.Logger instance.
func InitLogger() logr.Logger {
	opts := zap.Options{
		Development: false,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	return zap.New(zap.UseFlagOptions(&opts))
}

// FromContext returns a logr.Logger instance based on the provided context and key-value pairs.
// The logger is retrieved from the context using clog.FromContext method.
// The key-value pairs are optional and can be used for additional logging information.
// It expects a context.Context as the first argument, followed by optional key-value pairs.
// It returns a logr.Logger instance.
func FromContext(ctx context.Context, keyAndValues ...interface{}) logr.Logger {
	return clog.FromContext(ctx, keyAndValues...)
}

// WithContext adds a logr.Logger to the provided context.
// The logr.Logger is added using clog.IntoContext().
// The context returned will have the added logr.Logger included.
//
// Example usage:
//
//	ctx := context.Background()
//	logger := logr.New()
//	ctxWithLogger := WithContext(ctx, logger)
//
// Parameters:
//   - ctx: The context to add the logger to.
//   - log: The logr.Logger to add to the context.
//
// Returns:
//
//	The modified context with the added logr.Logger.
func WithContext(ctx context.Context, log logr.Logger) context.Context {
	return clog.IntoContext(ctx, log)
}

// SetCmdContext sets the context for a given command and all its subcommands.
// It ignores zap options for the command and all subcommands.
// It sets the context for each subcommand by adding the command's name and usage to the context.
func SetCmdContext(ctx context.Context, cmd *cobra.Command) {
	ignoreZapOptions(cmd)
	for _, c := range cmd.Commands() {
		ignoreZapOptions(c)
		c.SetContext(WithContext(
			ctx, FromContext(ctx, "cmd", cmd.Use)))
	}
}

// ignoreZapOptions sets up hidden flags for zap logger options.
// It takes a *cobra.Command as input.
// The function retrieves the flags from the command and defines hidden bool and string flags
// for the zap logger options.
// The zap logger options include: "zap-encoder", "zap-log-level", "zap-stacktrace-level",
// and "zap-time-encoding". For each option, a corresponding flag is defined and marked as hidden.
// This function does not return anything.
func ignoreZapOptions(cmd *cobra.Command) {
	flg := cmd.Flags()
	options := []string{
		"zap-encoder",
		"zap-log-level",
		"zap-stacktrace-level",
		"zap-time-encoding",
	}
	flg.BoolP("zap-devel", "", false, "")
	_ = flg.MarkHidden("zap-devel")

	for _, o := range options {
		flg.StringP(o, "", "", "")
		_ = flg.MarkHidden(o)
	}
}
