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
	"k8s.io/client-go/kubernetes/fake"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCleanEvictedPods(t *testing.T) {
	tests := []struct {
		name        string
		pods        []v1.Pod
		wantErr     bool
		wantDeleted int
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
					ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "pod1", Annotations: map[string]string{"Evicted": "True"}},
					Status: v1.PodStatus{
						Phase:   v1.PodFailed,
						Reason:  "Evicted",
						Message: "pod was Evicted",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "pod2", Annotations: map[string]string{"Evicted": "True"}},
					Status: v1.PodStatus{
						Phase:   v1.PodFailed,
						Reason:  "Evicted",
						Message: "pod was Evicted",
					},
				},
			},
			wantErr:     false,
			wantDeleted: 2,
		},
		{
			name: "MixedPods",
			pods: []v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "pod1", Annotations: map[string]string{"Evicted": "True"}},
					Status: v1.PodStatus{
						Phase:   v1.PodFailed,
						Reason:  "Evicted",
						Message: "pod was Evicted",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "pod2"},
				},
			},
			wantErr:     false,
			wantDeleted: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewSimpleClientset()
			for _, pod := range tt.pods {
				fmt.Println(client.CoreV1().Pods("test").Create(context.Background(), &pod, metav1.CreateOptions{}))
			}
			err := cleanEvictedPods(context.Background(), client, "test")
			if (err != nil) != tt.wantErr {
				t.Errorf("cleanEvictedPods() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			pods, _ := client.CoreV1().Pods("test").List(context.Background(), metav1.ListOptions{})
			if deleted := len(tt.pods) - len(pods.Items); deleted != tt.wantDeleted {
				t.Errorf("cleanEvictedPods() deleted = %v, wantDeleted %v", deleted, tt.wantDeleted)
			}
		})
	}
}

func TestNewCommand(t *testing.T) {
	assert.NotNil(t, NewCommand())
}
