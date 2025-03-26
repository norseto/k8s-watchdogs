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

package kube

import (
	"context"
	"testing"

	"github.com/norseto/k8s-watchdogs/pkg/logger"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
)

func TestRestartStatefulSet(t *testing.T) {
	ctx := logger.WithContext(context.Background(), logger.InitLogger())
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sts",
			Namespace: "default",
		},
	}
	client := fake.NewSimpleClientset(sts)

	// Check if patch was called
	var patchCalled bool
	client.PrependReactor("patch", "statefulsets", func(action ktesting.Action) (bool, runtime.Object, error) {
		patchCalled = true
		patchAction := action.(ktesting.PatchAction)
		// Verify patch contains restart annotation
		assert.Contains(t, string(patchAction.GetPatch()), "kubectl.kubernetes.io/restartedAt")
		return false, nil, nil
	})

	err := RestartStatefulSet(ctx, client, sts)
	assert.NoError(t, err)
	assert.True(t, patchCalled, "Expected patch to be called")
}
