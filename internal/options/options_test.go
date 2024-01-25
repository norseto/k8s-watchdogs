package options

import (
	"testing"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestOptions_Namespace(t *testing.T) {
	testCases := []struct {
		name      string
		namespace string
		expected  string
	}{
		{
			name:      "DefaultNamespace",
			namespace: metav1.NamespaceAll,
			expected:  metav1.NamespaceAll,
		},
		{
			name:      "CustomNamespace",
			namespace: "my-namespace",
			expected:  "my-namespace",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			options := &Options{
				namespace: tc.namespace,
			}

			actual := options.Namespace()

			if actual != tc.expected {
				t.Errorf("Expected namespace to be %s, but got %s", tc.expected, actual)
			}
		})
	}
}

func TestOptions_BindCommonFlags(t *testing.T) {
	cmd := &cobra.Command{}
	options := &Options{}

	options.BindCommonFlags(cmd)

	namespaceFlag := cmd.Flag("namespace")
	if namespaceFlag == nil {
		t.Error("Expected namespace flag to be bound")
	} else {
		expected := metav1.NamespaceAll
		actual := namespaceFlag.Value.String()

		if actual != expected {
			t.Errorf("Expected namespace flag value to be %s, but got %s", expected, actual)
		}
	}
}
