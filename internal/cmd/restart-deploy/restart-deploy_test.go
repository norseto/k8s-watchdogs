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
	"errors"
	"fmt"
	"testing"

	"github.com/norseto/k8s-watchdogs/pkg/kube/client"
	"github.com/norseto/k8s-watchdogs/pkg/logger"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// Common context used in tests
var testCtx = context.TODO()

func swapNewClientset(t *testing.T, factory func(*client.Options) (kubernetes.Interface, error)) func() {
	t.Helper()

	original := newClientset
	newClientset = factory

	return func() {
		newClientset = original
	}
}

func useFakeClientset(t *testing.T, objects ...runtime.Object) (*fake.Clientset, func()) {
	t.Helper()

	fakeClient := fake.NewSimpleClientset(objects...)
	restore := swapNewClientset(t, func(*client.Options) (kubernetes.Interface, error) {
		return fakeClient, nil
	})

	return fakeClient, restore
}

func convertToRuntimeObjects(deployments []*v1.Deployment) []runtime.Object {
	objects := make([]runtime.Object, len(deployments))
	for i, dep := range deployments {
		objects[i] = dep.DeepCopy()
	}
	return objects
}

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

	t.Run("args validation requires names without all flag", func(t *testing.T) {
		cmd := NewCommand()

		err := cmd.Args(cmd, []string{})

		assert.Error(t, err)
	})

	t.Run("args validation allows empty when all flag set", func(t *testing.T) {
		cmd := NewCommand()

		err := cmd.Flags().Set("all", "true")
		assert.NoError(t, err)

		argErr := cmd.Args(cmd, []string{})

		assert.NoError(t, argErr)
	})
}

func TestNewCommandRunEValidation(t *testing.T) {
	ctx := logger.WithContext(context.Background(), zap.New())

	t.Run("invalid namespace flag", func(t *testing.T) {
		cmd := NewCommand()
		cmd.SetContext(ctx)

		err := cmd.Flags().Set("namespace", "Invalid_Namespace")
		assert.NoError(t, err)

		runErr := cmd.RunE(cmd, []string{"valid-name"})
		assert.Error(t, runErr)
		assert.Contains(t, runErr.Error(), "invalid namespace")
	})

	t.Run("invalid deployment name", func(t *testing.T) {
		cmd := NewCommand()
		cmd.SetContext(ctx)

		err := cmd.Flags().Set("namespace", "default")
		assert.NoError(t, err)

		runErr := cmd.RunE(cmd, []string{"Invalid_Name"})
		assert.Error(t, runErr)
		assert.Contains(t, runErr.Error(), "invalid deployment name")
	})

	t.Run("restart all patches every deployment", func(t *testing.T) {
		cmd := NewCommand()
		cmd.SetContext(ctx)

		err := cmd.Flags().Set("namespace", "default")
		assert.NoError(t, err)

		err = cmd.Flags().Set("all", "true")
		assert.NoError(t, err)

		deployments := []*v1.Deployment{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "deployment-a",
					Namespace: "default",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "deployment-b",
					Namespace: "default",
				},
			},
		}

		fakeClient, restore := useFakeClientset(t, convertToRuntimeObjects(deployments)...)
		defer restore()

		patched := make(map[string]int)
		fakeClient.PrependReactor("patch", "deployments", func(action ktesting.Action) (bool, runtime.Object, error) {
			patchAction := action.(ktesting.PatchAction)
			name := patchAction.GetName()
			patched[name]++
			return true, &v1.Deployment{}, nil
		})

		runErr := cmd.RunE(cmd, nil)
		assert.NoError(t, runErr)

		for _, dep := range deployments {
			assert.Equal(t, 1, patched[dep.Name])
		}
	})

	t.Run("only targeted deployments are patched", func(t *testing.T) {
		cmd := NewCommand()
		cmd.SetContext(ctx)

		err := cmd.Flags().Set("namespace", "default")
		assert.NoError(t, err)

		deployments := []*v1.Deployment{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "deployment-a",
					Namespace: "default",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "deployment-b",
					Namespace: "default",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "deployment-c",
					Namespace: "default",
				},
			},
		}

		fakeClient, restore := useFakeClientset(t, convertToRuntimeObjects(deployments)...)
		defer restore()

		patched := make(map[string]int)
		fakeClient.PrependReactor("patch", "deployments", func(action ktesting.Action) (bool, runtime.Object, error) {
			patchAction := action.(ktesting.PatchAction)
			name := patchAction.GetName()
			patched[name]++
			return true, &v1.Deployment{}, nil
		})

		runErr := cmd.RunE(cmd, []string{"deployment-a", "deployment-c"})
		assert.NoError(t, runErr)

		assert.Equal(t, 1, patched["deployment-a"])
		assert.Equal(t, 1, patched["deployment-c"])
		assert.Zero(t, patched["deployment-b"])
	})

	t.Run("client creation error is returned", func(t *testing.T) {
		cmd := NewCommand()
		cmd.SetContext(ctx)

		err := cmd.Flags().Set("namespace", "default")
		assert.NoError(t, err)

		restore := swapNewClientset(t, func(*client.Options) (kubernetes.Interface, error) {
			return nil, errors.New("boom")
		})
		defer restore()

		runErr := cmd.RunE(cmd, []string{"valid-deploy"})
		assert.Error(t, runErr)
		assert.Contains(t, runErr.Error(), "failed to create client: boom")
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

	t.Run("restart missing deployment without get error", func(t *testing.T) {
		missingClient := fake.NewSimpleClientset()
		missingClient.PrependReactor("get", "deployments", func(action ktesting.Action) (bool, runtime.Object, error) {
			var obj *v1.Deployment
			return true, obj, nil
		})

		err := restartDeployment(testCtx, missingClient, "default", []string{"missing-deployment"})

		assert.EqualError(t, err, "deployment default/missing-deployment not found")
	})

	t.Run("too many targets", func(t *testing.T) {
		var names []string
		for i := 0; i < 51; i++ {
			names = append(names, fmt.Sprintf("deployment-%d", i))
		}

		err := restartDeployment(testCtx, mockClient, "default", names)

		assert.Error(t, err)
	})

	t.Run("restart deployment patch error", func(t *testing.T) {
		failingClient := fake.NewSimpleClientset(deployment)
		failingClient.PrependReactor("patch", "deployments", func(action ktesting.Action) (bool, runtime.Object, error) {
			return true, nil, assert.AnError
		})

		err := restartDeployment(testCtx, failingClient, "default", []string{"test-deployment"})

		assert.Error(t, err)
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

	// Test case where some deployments fail to restart
	t.Run("some deployments fail to restart", func(t *testing.T) {
		// Reset patch calls
		patchCalls = make(map[string]bool)

		// Create a new mock client for this test case
		failingClient := fake.NewSimpleClientset(deployment1, deployment2)
		failingClient.PrependReactor("patch", "deployments", func(action ktesting.Action) (bool, runtime.Object, error) {
			patchAction := action.(ktesting.PatchAction)
			name := patchAction.GetName()
			patchCalls[name] = true
			if name == "deployment-2" {
				return true, nil, assert.AnError
			}
			return false, nil, nil
		})

		err := restartAllDeployments(ctx, failingClient, "default")
		assert.Error(t, err, "Expected an error when some deployments fail to restart")
		assert.Contains(t, err.Error(), "deployment-2", "Expected error message to contain the name of the failed deployment")

		// Verify that both deployments were attempted to be patched
		assert.True(t, patchCalls["deployment-1"], "Expected deployment-1 to be restarted")
		assert.True(t, patchCalls["deployment-2"], "Expected deployment-2 to be attempted to be restarted")
	})

	t.Run("no deployments available", func(t *testing.T) {
		emptyClient := fake.NewSimpleClientset()

		err := restartAllDeployments(ctx, emptyClient, "default")

		assert.NoError(t, err)
	})

	t.Run("list deployments failure", func(t *testing.T) {
		errClient := fake.NewSimpleClientset()
		errClient.PrependReactor("list", "deployments", func(action ktesting.Action) (bool, runtime.Object, error) {
			return true, nil, assert.AnError
		})

		err := restartAllDeployments(ctx, errClient, "default")

		assert.Error(t, err)
	})
}
