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

package deleteoldest

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/norseto/k8s-watchdogs/internal/pkg/validation"
	"github.com/norseto/k8s-watchdogs/pkg/kube/client"
	"github.com/norseto/k8s-watchdogs/pkg/logger"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func TestDeleteOldestPods(t *testing.T) {
	ctx := context.Background()
	now := metav1.Now()
	client := fake.NewSimpleClientset(&corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "test-pod-1", Namespace: "test-ns"},
		Status: corev1.PodStatus{
			StartTime: &metav1.Time{Time: now.Add(-10)},
			Conditions: []corev1.PodCondition{
				{Type: corev1.PodReady, Status: corev1.ConditionTrue},
			},
			Phase: corev1.PodRunning,
		},
	}, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "test-pod-2", Namespace: "test-ns"},
		Status: corev1.PodStatus{
			StartTime: &metav1.Time{Time: now.Add(-20)},
			Conditions: []corev1.PodCondition{
				{Type: corev1.PodReady, Status: corev1.ConditionTrue},
			},
			Phase: corev1.PodRunning,
		},
	})
	err := deleteOldestPods(ctx, client, "test-ns", "test", 3)
	if err == nil {
		t.Errorf("Expected error, but got nil")
	}
	err = deleteOldestPods(ctx, client, "test-ns", "test-pods", 2)
	if err == nil {
		t.Errorf("Expected error, but got nil")
	}
	err = deleteOldestPods(ctx, client, "test-ns", "test-pod", 1)
	if err != nil {
		t.Errorf("Expected nil, but got %v", err)
	}
}

func TestNewCommandUsageWhenMissingFlags(t *testing.T) {
	cmd := NewCommand()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if buf.Len() == 0 {
		t.Fatalf("expected usage output when required flags missing")
	}
}

func TestNewCommandInvalidPrefix(t *testing.T) {
	cmd := NewCommand()
	ctx := logger.WithContext(context.Background(), zap.New())
	ctx = client.WithContext(ctx, &client.Options{})
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"--prefix", "INVALID", "--minPods", "1"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !strings.Contains(err.Error(), "invalid prefix") {
		t.Fatalf("expected invalid prefix error, got %v", err)
	}
}

func TestNewCommandClientError(t *testing.T) {
	t.Setenv("KUBECONFIG", filepath.Join(t.TempDir(), "missing", "config"))
	t.Setenv("HOME", filepath.Join(t.TempDir(), "home"))

	cmd := NewCommand()
	ctx := logger.WithContext(context.Background(), zap.New())
	ctx = client.WithContext(ctx, &client.Options{})
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"--prefix", "ok", "--minPods", "1"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected client creation error")
	}
	if !strings.Contains(err.Error(), "failed to create client") {
		t.Fatalf("expected client failure message, got %v", err)
	}
}

func TestDeleteOldestPods_ValidationError(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleClientset()
	namespace := "Invalid_Namespace"

	err := deleteOldestPods(ctx, client, namespace, "test", 1)
	if err == nil {
		t.Fatalf("expected error, but got nil")
	}

	unwrapped := errors.Unwrap(err)
	if unwrapped == nil {
		t.Fatalf("expected wrapped validation error, but got nil")
	}

	expected := validation.ValidateNamespace(namespace)
	if unwrapped.Error() != expected.Error() {
		t.Fatalf("expected validation error %q, got %q", expected.Error(), unwrapped.Error())
	}
}

func TestDeleteOldestPods_ListPodsError(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleClientset()
	listErr := errors.New("list pods failed")
	client.PrependReactor("list", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, listErr
	})

	err := deleteOldestPods(ctx, client, "valid-ns", "test", 1)
	if err == nil {
		t.Fatalf("expected error, but got nil")
	}

	if !errors.Is(err, listErr) {
		t.Fatalf("expected error to wrap list error, but got %v", err)
	}
}

func TestDeleteOldestPods_InvalidPickedPod(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleClientset(&corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "other-pod", Namespace: "valid-ns"},
	})

	err := deleteOldestPods(ctx, client, "valid-ns", "match", 0)
	if err == nil {
		t.Fatalf("expected error, but got nil")
	}

	if err.Error() != "invalid pod selected for deletion" {
		t.Fatalf("expected invalid pod selection error, but got %v", err)
	}
}

func TestDeleteOldestPods_DeletePodError(t *testing.T) {
	ctx := context.Background()
	now := metav1.Now()
	deleteErr := errors.New("delete pod failed")
	client := fake.NewSimpleClientset(&corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "test-pod-1", Namespace: "valid-ns"},
		Status: corev1.PodStatus{
			StartTime:  &metav1.Time{Time: now.Add(-10)},
			Conditions: []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionTrue}},
			Phase:      corev1.PodRunning,
		},
	}, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "test-pod-2", Namespace: "valid-ns"},
		Status: corev1.PodStatus{
			StartTime:  &metav1.Time{Time: now.Add(-20)},
			Conditions: []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionTrue}},
			Phase:      corev1.PodRunning,
		},
	})
	client.PrependReactor("delete", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, deleteErr
	})

	err := deleteOldestPods(ctx, client, "valid-ns", "test-pod", 1)
	if err == nil {
		t.Fatalf("expected error, but got nil")
	}

	if !errors.Is(err, deleteErr) {
		t.Fatalf("expected error to wrap delete error, but got %v", err)
	}

	wrapped := errors.Unwrap(err)
	if wrapped == nil || !strings.Contains(wrapped.Error(), "failed to delete Pod") {
		t.Fatalf("expected wrapped error to include delete failure message, but got %v", wrapped)
	}
}

func TestPickOldest(t *testing.T) {
	now := metav1.Now()
	pods := []corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "test-pod-1"},
			Status: corev1.PodStatus{
				StartTime: &metav1.Time{Time: now.Add(-10)},
				Conditions: []corev1.PodCondition{
					{Type: corev1.PodReady, Status: corev1.ConditionTrue},
				},
				Phase: corev1.PodRunning,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "test-pod-2"},
			Status: corev1.PodStatus{
				StartTime: &metav1.Time{Time: now.Add(-20)},
				Conditions: []corev1.PodCondition{
					{Type: corev1.PodReady, Status: corev1.ConditionTrue},
				},
				Phase: corev1.PodRunning,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "test-pod-3"},
			Status: corev1.PodStatus{
				StartTime: &metav1.Time{Time: now.Add(-30)},
				Conditions: []corev1.PodCondition{
					{Type: corev1.PodReady, Status: corev1.ConditionFalse},
				},
				Phase: corev1.PodRunning,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "test-pod-4"},
			Status: corev1.PodStatus{
				StartTime: &metav1.Time{Time: now.Add(-40)},
				Conditions: []corev1.PodCondition{
					{Type: corev1.PodReady, Status: corev1.ConditionTrue},
				},
				Phase: corev1.PodPending,
			},
		},
	}
	pod, err := pickOldest("test-pod-", 2, pods)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if pod.Name != "test-pod-2" {
		t.Errorf("expected test-pod-2, but got %s", pod.Name)
	}

	_, err = pickOldest("test-pod-", 3, pods)
	if err == nil {
		t.Errorf("expected error, but got nil")
	}
}

func TestNewCommand(t *testing.T) {
	cmd := NewCommand()
	if cmd == nil {
		t.Errorf("Expected command, but got nil")
	}
}

func TestNewCommand_ClientCreationError(t *testing.T) {
	t.Setenv("KUBECONFIG", "/invalid/path")
	t.Setenv("KUBERNETES_SERVICE_HOST", "")
	t.Setenv("KUBERNETES_SERVICE_PORT", "")

	cmd := NewCommand()
	if cmd == nil {
		t.Fatalf("expected command, but got nil")
	}

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	ctx := client.WithContext(context.Background(), &client.Options{})
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"--namespace", "valid-ns", "--prefix", "test", "--minPods", "1"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected error, but got nil")
	}

	if !strings.Contains(err.Error(), "failed to create client") {
		t.Fatalf("expected client creation error, but got %v", err)
	}

	if !cmd.SilenceUsage {
		t.Fatalf("expected SilenceUsage to be true after attempting client creation")
	}
}

func TestNewCommand_EarlyExitWithoutPrefix(t *testing.T) {
	cmd := NewCommand()
	if cmd == nil {
		t.Fatalf("expected command, but got nil")
	}

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--namespace", "test-ns"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected nil error, but got %v", err)
	}

	if buf.Len() == 0 {
		t.Fatalf("expected usage output, but buffer was empty")
	}

	if cmd.SilenceUsage {
		t.Fatalf("expected SilenceUsage to remain false")
	}
}

func TestNewCommand_MinPodsTooHigh(t *testing.T) {
	cmd := NewCommand()
	if cmd == nil {
		t.Fatalf("expected command, but got nil")
	}

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--namespace", "test-ns", "--prefix", "test", "--minPods", "1001"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected error, but got nil")
	}

	if !strings.Contains(err.Error(), "minPods value too high for safety") {
		t.Fatalf("expected minPods safety error, but got %v", err)
	}

	if cmd.SilenceUsage {
		t.Fatalf("expected SilenceUsage to remain false when guard triggers")
	}
}

func TestValidatePodPrefix(t *testing.T) {
	tests := []struct {
		name    string
		prefix  string
		wantErr bool
	}{
		{name: "valid", prefix: "valid-prefix", wantErr: false},
		{name: "empty", prefix: "", wantErr: true},
		{name: "too-long", prefix: strings.Repeat("a", 51), wantErr: true},
		{name: "invalid-pattern", prefix: "Invalid_Prefix", wantErr: true},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidatePodPrefix(tt.prefix)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePodPrefix(%q) error = %v, wantErr %v", tt.prefix, err, tt.wantErr)
			}
		})
	}
}

func TestValidateNamespace(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		wantErr   bool
	}{
		{name: "valid", namespace: "valid-ns", wantErr: false},
		{name: "empty", namespace: "", wantErr: true},
		{name: "invalid-chars", namespace: "Invalid_Namespace", wantErr: true},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateNamespace(tt.namespace)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateNamespace(%q) error = %v, wantErr %v", tt.namespace, err, tt.wantErr)
			}
		})
	}
}
