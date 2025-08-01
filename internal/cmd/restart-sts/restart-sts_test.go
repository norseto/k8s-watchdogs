/*
MIT License

Copyright (c) 2019-2025 Norihiro Seto

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

package restartsts

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
)

// Common context used in tests
var testCtx = context.TODO()

func TestValidateResourceName(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid", "valid-name", false},
		{"with dot", "sts.app", false},
		{"numeric", "abc123", false},
		{"empty", "", true},
		{"invalid char", "Invalid*", true},
		{"too long", strings.Repeat("a", 254), true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateResourceName(tc.input)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateNamespace(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"default", "default", false},
		{"dash", "my-ns", false},
		{"empty", "", true},
		{"invalid char", "Invalid*", true},
		{"too long", strings.Repeat("a", 64), true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateNamespace(tc.input)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRestartStatefulSet(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		sts := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-sts",
				Namespace: "default",
			},
		}
		client := fake.NewSimpleClientset(sts)

		var patchCalled bool
		client.PrependReactor("patch", "statefulsets", func(action ktesting.Action) (bool, runtime.Object, error) {
			patchCalled = true
			return false, nil, nil
		})

		err := restartStatefulSet(testCtx, client, "default", []string{"test-sts"})
		assert.NoError(t, err)
		assert.True(t, patchCalled, "Expected patch to be called")
	})

	t.Run("too many targets", func(t *testing.T) {
		var names []string
		for i := 0; i < 51; i++ {
			names = append(names, fmt.Sprintf("sts-%d", i))
		}
		client := fake.NewSimpleClientset()
		err := restartStatefulSet(testCtx, client, "default", names)
		assert.Error(t, err)
	})

	t.Run("get failure", func(t *testing.T) {
		sts := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-sts",
				Namespace: "default",
			},
		}
		client := fake.NewSimpleClientset(sts)
		client.PrependReactor("get", "statefulsets", func(action ktesting.Action) (bool, runtime.Object, error) {
			return true, nil, assert.AnError
		})
		err := restartStatefulSet(testCtx, client, "default", []string{"test-sts"})
		assert.Error(t, err)
	})

	t.Run("restart failure", func(t *testing.T) {
		sts := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-sts",
				Namespace: "default",
			},
		}
		client := fake.NewSimpleClientset(sts)
		client.PrependReactor("patch", "statefulsets", func(action ktesting.Action) (bool, runtime.Object, error) {
			return true, nil, assert.AnError
		})
		err := restartStatefulSet(testCtx, client, "default", []string{"test-sts"})
		assert.Error(t, err)
	})
}

func TestNewCommand(t *testing.T) {
	cmd := NewCommand()
	assert.Equal(t, "restart-sts [statefulset-name|--all]", cmd.Use)

	// Verify --all flag exists
	allFlag := cmd.Flag("all")
	assert.NotNil(t, allFlag, "Expected --all flag to exist")
	assert.Equal(t, "a", allFlag.Shorthand, "Expected shorthand for --all to be -a")

	t.Run("invalid namespace", func(t *testing.T) {
		cmd := NewCommand()
		cmd.Flags().Set("namespace", "Invalid*")
		cmd.SetArgs([]string{"test"})
		err := cmd.Execute()
		assert.Error(t, err)
	})

	t.Run("invalid name", func(t *testing.T) {
		cmd := NewCommand()
		cmd.SetArgs([]string{"Invalid*"})
		err := cmd.Execute()
		assert.Error(t, err)
	})
}

func TestRestartAllStatefulSets(t *testing.T) {
	// Use common context
	ctx := testCtx

	// Create mock client with multiple statefulsets
	sts1 := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "statefulset-1",
			Namespace: "default",
		},
	}
	sts2 := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "statefulset-2",
			Namespace: "default",
		},
	}

	mockClient := fake.NewSimpleClientset(sts1, sts2)

	// Track if patch was called for each statefulset
	patchCalls := make(map[string]bool)
	mockClient.PrependReactor("patch", "statefulsets", func(action ktesting.Action) (bool, runtime.Object, error) {
		patchAction := action.(ktesting.PatchAction)
		name := patchAction.GetName()
		patchCalls[name] = true
		return false, nil, nil
	})

	// Test restarting all statefulsets
	t.Run("restart all statefulsets", func(t *testing.T) {
		err := restartAllStatefulSets(ctx, mockClient, "default")
		assert.NoError(t, err, "Expected no error when restarting all statefulsets")

		// Verify both statefulsets were patched
		assert.True(t, patchCalls["statefulset-1"], "Expected statefulset-1 to be restarted")
		assert.True(t, patchCalls["statefulset-2"], "Expected statefulset-2 to be restarted")
	})

	// Test case where some statefulsets fail to restart
	t.Run("some statefulsets fail to restart", func(t *testing.T) {
		// Reset patch calls
		patchCalls = make(map[string]bool)

		// Create a new mock client for this test case
		failingClient := fake.NewSimpleClientset(sts1, sts2)
		failingClient.PrependReactor("patch", "statefulsets", func(action ktesting.Action) (bool, runtime.Object, error) {
			patchAction := action.(ktesting.PatchAction)
			name := patchAction.GetName()
			patchCalls[name] = true
			if name == "statefulset-2" {
				return true, nil, assert.AnError
			}
			return false, nil, nil
		})

		err := restartAllStatefulSets(ctx, failingClient, "default")
		assert.Error(t, err, "Expected an error when some statefulsets fail to restart")
		assert.Contains(t, err.Error(), "statefulset-2", "Expected error message to contain the name of the failed statefulset")

		// Verify that both statefulsets were attempted to be patched
		assert.True(t, patchCalls["statefulset-1"], "Expected statefulset-1 to be restarted")
		assert.True(t, patchCalls["statefulset-2"], "Expected statefulset-2 to be attempted to be restarted")
	})

	t.Run("list fails", func(t *testing.T) {
		failingClient := fake.NewSimpleClientset()
		failingClient.PrependReactor("list", "statefulsets", func(action ktesting.Action) (bool, runtime.Object, error) {
			return true, nil, assert.AnError
		})
		err := restartAllStatefulSets(ctx, failingClient, "default")
		assert.Error(t, err)
	})

	t.Run("no statefulsets", func(t *testing.T) {
		emptyClient := fake.NewSimpleClientset()
		err := restartAllStatefulSets(ctx, emptyClient, "default")
		assert.NoError(t, err)
	})
}
