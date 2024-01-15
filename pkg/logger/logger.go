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

func SetCmdContext(ctx context.Context, cmd *cobra.Command) {
	for _, c := range cmd.Commands() {
		c.SetContext(WithContext(
			ctx, FromContext(ctx, "cmd", cmd.Use)))
	}
}

//func ignoreZapOptions(cmd *cobra.Command) {
//	//   - zap-devel:
//	//     Development Mode defaults(encoder=consoleEncoder,logLevel=Debug,stackTraceLevel=Warn)
//	//     Production Mode defaults(encoder=jsonEncoder,logLevel=Info,stackTraceLevel=Error)
//	//   - zap-encoder: Zap log encoding (one of 'json' or 'console')
//	//   - zap-log-level: Zap Level to configure the verbosity of logging. Can be one of 'debug', 'info', 'error',
//	//     or any integer value > 0 which corresponds to custom debug levels of increasing verbosity").
//	//   - zap-stacktrace-level: Zap Level at and above which stacktraces are captured (one of 'info', 'error' or 'panic')
//	//   - zap-time-encoding: Zap time encoding (one of 'epoch', 'millis', 'nano', 'iso8601', 'rfc3339' or 'rfc3339nano'),
//	//     Defaults to 'epoch'.
//}
