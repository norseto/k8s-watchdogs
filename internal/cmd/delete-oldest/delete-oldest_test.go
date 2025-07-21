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
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
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
