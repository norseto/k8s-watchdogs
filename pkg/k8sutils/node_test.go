package k8sutils

import (
	"context"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"testing"
)

func TestGetUntaintedNodes(t *testing.T) {
	taint := v1.Taint{
		Key:    "key",
		Value:  "value",
		Effect: v1.TaintEffectNoSchedule,
	}

	untaintedNode := v1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "untaintedNode"},
		Spec:       v1.NodeSpec{},
	}

	taintedNode := v1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "taintedNode"},
		Spec:       v1.NodeSpec{Taints: []v1.Taint{taint}},
	}

	tests := []struct {
		name     string
		nodes    []v1.Node
		expected []v1.Node
	}{
		{
			name:     "NoNodes",
			nodes:    []v1.Node{},
			expected: []v1.Node{},
		},
		{
			name:     "UntaintedNode",
			nodes:    []v1.Node{untaintedNode},
			expected: []v1.Node{untaintedNode},
		},
		{
			name:     "TaintedNode",
			nodes:    []v1.Node{taintedNode},
			expected: []v1.Node{},
		},
		{
			name:     "UntaintedAndTaintedNodes",
			nodes:    []v1.Node{untaintedNode, taintedNode},
			expected: []v1.Node{untaintedNode},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := fake.NewSimpleClientset()
			for _, node := range tc.nodes {
				_, _ = client.CoreV1().Nodes().Create(context.Background(), &node, metav1.CreateOptions{})
			}

			got, err := GetUntaintedNodes(context.Background(), client)

			assert.NoError(t, err)
			assert.ElementsMatch(t, tc.expected, got)
		})
	}
}
