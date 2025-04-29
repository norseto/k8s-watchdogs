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

package restartdeploy

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
)

// Common context used in tests
var testCtx = context.TODO()

// TestNewCommand validates the NewCommand function
func TestNewCommand(t *testing.T) {
	t.Run("test new command", func(t *testing.T) {
		cmd := NewCommand()

		// Check command usage
		assert.Equal(t, "restart-deploy [deployment-name|--all]", cmd.Use)

		// Check subcommand
		assert.Equal(t, "Restart deployments by name or all with --all", cmd.Short)

		// Verify --all flag exists
		allFlag := cmd.Flag("all")
		assert.NotNil(t, allFlag, "Expected --all flag to exist")
		assert.Equal(t, "a", allFlag.Shorthand, "Expected shorthand for --all to be -a")
	})
}

func TestRestartDeployment(t *testing.T) {
	mockClient := fake.NewSimpleClientset()

	deployment := &v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "default",
		},
	}

	_, err := mockClient.AppsV1().Deployments("default").Create(context.TODO(), deployment, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("restart valid deployment", func(t *testing.T) {
		err := restartDeployment(testCtx, mockClient, "default", []string{"test-deployment"})
		if err != nil {
			t.Fatal(err)
		}
	})

	// Enter the name of deployment that does not exist
	t.Run("restart invalid deployment", func(t *testing.T) {
		err := restartDeployment(testCtx, mockClient, "default", []string{"invalid-deployment"})
		assert.NotNil(t, err)
	})
}

func TestRestartAllDeployments(t *testing.T) {
	// Use common context
	ctx := testCtx

	// Create mock client with multiple deployments
	deployment1 := &v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deployment-1",
			Namespace: "default",
		},
	}
	deployment2 := &v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deployment-2",
			Namespace: "default",
		},
	}

	mockClient := fake.NewSimpleClientset(deployment1, deployment2)

	// Track if patch was called for each deployment
	patchCalls := make(map[string]bool)
	mockClient.PrependReactor("patch", "deployments", func(action ktesting.Action) (bool, runtime.Object, error) {
		patchAction := action.(ktesting.PatchAction)
		name := patchAction.GetName()
		patchCalls[name] = true
		return false, nil, nil
	})

	// Test restarting all deployments
	t.Run("restart all deployments", func(t *testing.T) {
		err := restartAllDeployments(ctx, mockClient, "default")
		assert.NoError(t, err, "Expected no error when restarting all deployments")

		// Verify both deployments were patched
		assert.True(t, patchCalls["deployment-1"], "Expected deployment-1 to be restarted")
		assert.True(t, patchCalls["deployment-2"], "Expected deployment-2 to be restarted")
	})

	// Test empty namespace (no deployments)
	t.Run("no deployments in namespace", func(t *testing.T) {
		emptyClient := fake.NewSimpleClientset()
		err := restartAllDeployments(ctx, emptyClient, "empty")
		assert.NoError(t, err, "Expected no error when no deployments exist")
	})
}
