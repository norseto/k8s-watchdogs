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
