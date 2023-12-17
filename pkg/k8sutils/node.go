package k8sutils

import (
	"context"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// GetUntaintedNodes gets normal(untainted) nodes.
// Parameter:
//
//	c *kubernetes.Clientset : clientset
//
// Returns:
//
//	[]v1.Node : All target nodes that does not contain TAINTED nodes
//	error : error if error happens
func GetUntaintedNodes(ctx context.Context, c *kubernetes.Clientset) ([]v1.Node, error) {
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
