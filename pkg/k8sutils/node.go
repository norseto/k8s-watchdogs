package k8sutils

import (
	"context"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

/*
GetUntaintedNodes returns a list of untainted nodes in the Kubernetes cluster.

Parameters:
- ctx (context.Context): Context for the request.
- c (kubernetes.Interface): Kubernetes client interface.

Returns:
- []v1.Node: List of untainted nodes.
- error: Error, if any.

Summary:
This function retrieves all nodes from the Kubernetes cluster and filters out the nodes that do not have any taints. It returns the list of untainted nodes and an error, if any.
*/
func GetUntaintedNodes(_ context.Context, c kubernetes.Interface) ([]v1.Node, error) {
	var nodes []v1.Node
	all, err := c.CoreV1().Nodes().List(metav1.ListOptions{IncludeUninitialized: false})
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
