package k8sutils

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGetAllNodes(t *testing.T) {
	client := fake.NewSimpleClientset()
	ctx := context.Background()

	t.Run("ReturnsNodesList", func(t *testing.T) {
		var nodes []*corev1.Node
		var err error

		nodes, err = GetAllNodes(ctx, client)
		assert.Equal(t, 0, len(nodes))
		assert.NoError(t, err)

		_, err = client.CoreV1().Nodes().Create(ctx, &corev1.Node{}, metav1.CreateOptions{})
		assert.NoError(t, err)

		nodes, err = GetAllNodes(ctx, client)
		assert.Equal(t, 1, len(nodes))
		assert.NoError(t, err)
	})
}

func TestCanSchedule(t *testing.T) {
	t.Run("ReturnsFalseForNonToleratedTaints", func(t *testing.T) {
		node := &corev1.Node{Spec: corev1.NodeSpec{Taints: []corev1.Taint{{Effect: "NoSchedule"}}}}
		podSpec := &corev1.PodSpec{}
		assert.False(t, CanSchedule(node, podSpec))
	})

	t.Run("ReturnsTrueForToleratedTaints", func(t *testing.T) {
		node := &corev1.Node{Spec: corev1.NodeSpec{Taints: []corev1.Taint{{Effect: "NoSchedule"}}}}
		podSpec := &corev1.PodSpec{Tolerations: []corev1.Toleration{{Effect: "NoSchedule"}}}
		assert.True(t, CanSchedule(node, podSpec))
	})
}

func TestToleratesAllTaints(t *testing.T) {
	t.Run("ReturnsTrueForNoTaints", func(t *testing.T) {
		node := &corev1.Node{Spec: corev1.NodeSpec{}}
		podSpec := &corev1.PodSpec{}
		assert.True(t, toleratesAllTaints(node, podSpec))
	})

	t.Run("ReturnsFalseForNonToleratedTaints", func(t *testing.T) {
		node := &corev1.Node{Spec: corev1.NodeSpec{Taints: []corev1.Taint{{Effect: "NoSchedule"}}}}
		podSpec := &corev1.PodSpec{}
		assert.False(t, toleratesAllTaints(node, podSpec))
	})

	t.Run("ReturnsTrueForToleratedTaints", func(t *testing.T) {
		node := &corev1.Node{Spec: corev1.NodeSpec{Taints: []corev1.Taint{{Effect: "NoSchedule"}}}}
		podSpec := &corev1.PodSpec{Tolerations: []corev1.Toleration{{Effect: "NoSchedule"}}}
		assert.True(t, toleratesAllTaints(node, podSpec))
	})
}

func TestNodeMatchesNodeSelector(t *testing.T) {
	testCases := []struct {
		name     string
		node     *corev1.Node
		selector *corev1.NodeSelector
		expected bool
	}{
		{
			name:     "NoSelectorReturnsTrue",
			node:     &corev1.Node{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"foo": "bar"}}},
			selector: &corev1.NodeSelector{NodeSelectorTerms: []corev1.NodeSelectorTerm{{MatchExpressions: []corev1.NodeSelectorRequirement{{}}}}},
			expected: true,
		},
		{
			name:     "MatchingSelectorReturnsTrue",
			node:     &corev1.Node{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"foo": "bar"}}},
			selector: &corev1.NodeSelector{NodeSelectorTerms: []corev1.NodeSelectorTerm{{MatchExpressions: []corev1.NodeSelectorRequirement{{Key: "foo", Values: []string{"bar"}, Operator: "In"}}}}},
			expected: true,
		},
		{
			name:     "NotMatchingSelectorReturnsTrue",
			node:     &corev1.Node{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"foo": "bar"}}},
			selector: &corev1.NodeSelector{NodeSelectorTerms: []corev1.NodeSelectorTerm{{MatchExpressions: []corev1.NodeSelectorRequirement{{Key: "foo2", Values: []string{"bar2"}, Operator: "In"}}}}},
			expected: false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, nodeMatchesNodeSelector(tt.node, tt.selector))
		})
	}
}
