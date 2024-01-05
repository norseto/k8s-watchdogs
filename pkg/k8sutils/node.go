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

// IsScheduleable checks whether a node is schedulable for a given pod specification.
func IsScheduleable(node *v1.Node, spec *v1.PodSpec) bool {
	return tolerationsTolerateTaints(spec.Tolerations, node.Spec.Taints)
}

// Function that checks if Tolerations can tolerate the given Taints
func tolerationsTolerateTaints(tolerations []v1.Toleration, taints []v1.Taint) bool {
	for _, taint := range taints {
		taintNotTolerated := true
		for _, toleration := range tolerations {
			if toleratesTaint(&toleration, &taint) {
				taintNotTolerated = false
				break
			}
		}
		if taintNotTolerated {
			return false
		}
	}
	return true
}

// Function that checks if a Taint is tolerated by a Toleration
func toleratesTaint(toleration *v1.Toleration, taint *v1.Taint) bool {
	if len(toleration.Effect) > 0 && toleration.Effect != taint.Effect {
		return false
	}
	if len(toleration.Key) > 0 && toleration.Key != taint.Key {
		return false
	}
	if len(toleration.Operator) > 0 && toleration.Operator != v1.TolerationOpExists {
		equalityBased := (toleration.Operator == v1.TolerationOpEqual && toleration.Value == taint.Value)
		existsBased := toleration.Operator == v1.TolerationOpExists
		return equalityBased || existsBased
	}
	return true
}
