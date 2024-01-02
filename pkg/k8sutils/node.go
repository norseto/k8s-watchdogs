package k8sutils

import (
	"context"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// GetUntaintedNodes returns a list of nodes that have no taints.
func GetUntaintedNodes(ctx context.Context, c kubernetes.Interface) ([]v1.Node, error) {
	var nodes []v1.Node
	all, err := c.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list nodes")
	}
	for _, n := range all.Items {
		if len(n.Spec.Taints) < 1 {
			nodes = append(nodes, n)
		}
	}
	return nodes, nil
}
