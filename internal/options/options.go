package options

import (
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Options represents a set of configuration options.
type Options struct {
	namespace string
}

// BindCommonFlags binds the "namespace" flag to the "namespace" field in the Options struct.
// This allows the value provided for the flag to be assigned to the Options struct's namespace
func (o *Options) BindCommonFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&o.namespace, "namespace", metav1.NamespaceAll, "namespace")
}

// Namespace returns the value of the `namespace` field in the Options struct.
func (o *Options) Namespace() string {
	return o.namespace
}
