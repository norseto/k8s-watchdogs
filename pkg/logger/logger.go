package logger

import (
	"context"
	"flag"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	clog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// InitLogger initializes the logger configuration.
func InitLogger(rootCmd *cobra.Command) {
	rootCmd.PreRun = func(cmd *cobra.Command, args []string) {
		initLogger(cmd)
	}
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		key := "subcmd"
		if !cmd.HasParent() {
			key = "cmd"
			initLogger(rootCmd)
		}
		cmd.SetContext(WithContext(ctx, FromContext(ctx, key, cmd.Use)))
	}
}

func initLogger(rootCmd *cobra.Command) {
	opts := zap.Options{
		Development: false,
	}
	BindPFlags(&opts, rootCmd.PersistentFlags())
	cmdline := makeCommandLine(rootCmd.PersistentFlags())
	flagSet := flag.NewFlagSet("standard", flag.ContinueOnError)
	opts.BindFlags(flagSet)
	_ = flagSet.Parse(cmdline)
	logger := zap.New(zap.UseFlagOptions(&opts))
	rootCmd.SetContext(WithContext(rootCmd.Context(), logger))
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

// BindPFlags setups zap log options
func BindPFlags(o *zap.Options, fs *pflag.FlagSet) {
	// Set Development mode value
	fs.Bool("zap-devel", o.Development,
		"Development Mode defaults(encoder=consoleEncoder,logLevel=Debug,stackTraceLevel=Warn). "+
			"Production Mode defaults(encoder=jsonEncoder,logLevel=Info,stackTraceLevel=Error)")

	fs.String("zap-encoder", "console", "Zap log encoding (one of 'json' or 'console')")

	// Set the Log Level
	fs.String("zap-log-level", "info",
		"Zap Level to configure the verbosity of logging. Can be one of 'debug', 'info', 'error', "+
			"or any integer value > 0 which corresponds to custom debug levels of increasing verbosity")

	// Set the StrackTrace Level
	fs.String("zap-stacktrace-level", "error",
		"Zap Level at and above which stacktraces are captured (one of 'info', 'error', 'panic').")

	// Set the time encoding
	fs.String("zap-time-encoding", "epoch", "Zap time encoding (one of 'epoch', 'millis', 'nano', 'iso8601', 'rfc3339' or 'rfc3339nano'). Defaults to 'epoch'.")

	_ = fs.MarkHidden("zap-devel")
	_ = fs.MarkHidden("zap-encoder")
	_ = fs.MarkHidden("zap-log-level")
	_ = fs.MarkHidden("zap-stacktrace-level")
	_ = fs.MarkHidden("zap-time-encoding")
}

// makes command lines from FlagSet values
func makeCommandLine(fs *pflag.FlagSet) []string {
	result := []string{""}

	fs.VisitAll(func(f *pflag.Flag) {
		if f.Changed {
			result = append(result, fmt.Sprintf(" --%s=%v", f.Name, f.Value))
		}
	})
	return result
}
