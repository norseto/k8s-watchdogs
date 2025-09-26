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

package cleanevicted

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	k8stesting "k8s.io/client-go/testing"

	"github.com/norseto/k8s-watchdogs/internal/pkg/validation"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestCleanEvictedPods(t *testing.T) {
	tests := []struct {
		name             string
		pods             []v1.Pod
		wantErr          bool
		wantDeleted      int
		deleteShouldFail bool
	}{
		{
			name:        "NoPods",
			pods:        []v1.Pod{},
			wantErr:     false,
			wantDeleted: 0,
		},
		{
			name: "EvictedPods",
			pods: []v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "pod1"},
					Status:     v1.PodStatus{Phase: v1.PodFailed, Reason: "Evicted"},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "pod2"},
					Status:     v1.PodStatus{Phase: v1.PodFailed, Reason: "Evicted"},
				},
			},
			wantErr:     false,
			wantDeleted: 2,
		},
		{
			name: "MixedPods",
			pods: []v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "pod1"},
					Status:     v1.PodStatus{Phase: v1.PodFailed, Reason: "Evicted"},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "pod2"},
				},
			},
			wantErr:     false,
			wantDeleted: 1,
		},
		{
			name: "ErrorDeletingPods",
			pods: []v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "pod1"},
					Status:     v1.PodStatus{Phase: v1.PodFailed, Reason: "Evicted"},
				},
			},
			wantErr:          true,
			wantDeleted:      0,
			deleteShouldFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewSimpleClientset()
			for i := range tt.pods {
				_, err := client.CoreV1().Pods("test").Create(context.Background(), &tt.pods[i], metav1.CreateOptions{})
				assert.NoError(t, err)
			}

			if tt.deleteShouldFail {
				client.PrependReactor("delete", "pods", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, fmt.Errorf("failed to delete pod")
				})
			}

			err := cleanEvictedPods(context.Background(), client, "test")
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			pods, err := client.CoreV1().Pods("test").List(context.Background(), metav1.ListOptions{})
			assert.NoError(t, err)

			assert.Equal(t, len(tt.pods)-tt.wantDeleted, len(pods.Items))
		})
	}
}

// TestCleanEvictedPodsLimit verifies deletion is capped at 100 pods.
func TestCleanEvictedPodsLimit(t *testing.T) {
	client := fake.NewSimpleClientset()
	for i := 0; i < 150; i++ {
		pod := v1.Pod{
			ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: fmt.Sprintf("pod-%d", i)},
			Status:     v1.PodStatus{Phase: v1.PodFailed, Reason: "Evicted"},
		}
		_, err := client.CoreV1().Pods("test").Create(context.Background(), &pod, metav1.CreateOptions{})
		assert.NoError(t, err)
	}

	err := cleanEvictedPods(context.Background(), client, "test")
	assert.NoError(t, err)

	deleteActions := 0
	for _, action := range client.Actions() {
		if action.Matches("delete", "pods") {
			deleteActions++
		}
	}
	assert.Equal(t, 100, deleteActions)

	pods, err := client.CoreV1().Pods("test").List(context.Background(), metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Equal(t, 50, len(pods.Items))
}

func TestCleanEvictedPodsInvalidNamespace(t *testing.T) {
	client := fake.NewSimpleClientset()

	err := cleanEvictedPods(context.Background(), client, "Invalid_Namespace")

	if assert.Error(t, err) {
		assert.True(t, strings.HasPrefix(err.Error(), "invalid namespace"))
	}

	assert.Empty(t, client.Actions())
}

func TestCleanEvictedPodsListError(t *testing.T) {
	client := fake.NewSimpleClientset()

	client.PrependReactor("list", "pods", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, nil, fmt.Errorf("synthetic list error")
	})

	err := cleanEvictedPods(context.Background(), client, "test")

	assert.Error(t, err)

	for _, action := range client.Actions() {
		if action.Matches("delete", "pods") {
			t.Fatalf("unexpected delete action after list error: %#v", action)
		}
	}
}

// TestValidateNamespace tests namespace validation.
func TestValidateNamespace(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		wantErr   bool
	}{
		{name: "Empty", namespace: "", wantErr: true},
		{name: "Invalid", namespace: "Invalid_Namespace", wantErr: true},
		{name: "Valid", namespace: "valid-namespace", wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateNamespace(tt.namespace)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewCommand(t *testing.T) {
	assert.NotNil(t, NewCommand())
}
