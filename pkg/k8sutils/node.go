package k8sutils

import (
	"context"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

/*
GetAllNodes returns a list of untainted nodes in the Kubernetes cluster.

Parameters:
- ctx (context.Context): Context for the request.
- c (kubernetes.Interface): Kubernetes client interface.

Returns:
- []v1.Node: List of untainted nodes.
- error: Error, if any.

Summary:
This function retrieves all nodes from the Kubernetes cluster and filters out the nodes that do not have any taints. It returns the list of untainted nodes and an error, if any.
*/
func GetAllNodes(_ context.Context, c kubernetes.Interface) ([]*v1.Node, error) {
	all, err := c.CoreV1().Nodes().List(metav1.ListOptions{IncludeUninitialized: false})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list nodes")
	}
	nodes := make([]*v1.Node, len(all.Items))
	for i, n := range all.Items {
		nodes[i] = n.DeepCopy()
	}
	return nodes, nil
}

// CanSchedule checks if a pod can be scheduled on a given node based on various criteria.
func CanSchedule(node *v1.Node, pod *v1.Pod) bool {
	// Check Taints and Tolerations
	if !toleratesAllTaints(node, pod) {
		return false
	}

	// Check nodeSelector
	if pod.Spec.NodeSelector != nil {
		for key, value := range pod.Spec.NodeSelector {
			if nodeValue, exists := node.Labels[key]; !exists || nodeValue != value {
				return false
			}
		}
	}

	// Check NodeAffinity
	if pod.Spec.Affinity != nil && pod.Spec.Affinity.NodeAffinity != nil {
		nodeAffinity := pod.Spec.Affinity.NodeAffinity
		if nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution != nil {
			if !nodeMatchesNodeSelector(node, nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution) {
				return false
			}
		}
	}

	return true
}

// toleratesAllTaints checks that pod is tolerated with all taints in the node.
func toleratesAllTaints(node *v1.Node, pod *v1.Pod) bool {
	for _, taint := range node.Spec.Taints {
		if !toleratesTaint(pod, taint) {
			return false
		}
	}
	return true
}

// nodeMatchesNodeSelector checks that the pod matches all node selectors
func nodeMatchesNodeSelector(node *v1.Node, selector *v1.NodeSelector) bool {
	for _, term := range selector.NodeSelectorTerms {
		if nodeSelectorTermMatches(node, &term) {
			return true
		}
	}
	return false
}

// nodeSelectorTermMatches checks that the pod matches the specific node selector
func nodeSelectorTermMatches(node *v1.Node, term *v1.NodeSelectorTerm) bool {
	for _, expr := range term.MatchExpressions {
		switch expr.Operator {
		case v1.NodeSelectorOpIn:
			if !contains(node.Labels[expr.Key], expr.Values) {
				return false
			}
		case v1.NodeSelectorOpNotIn:
			if contains(node.Labels[expr.Key], expr.Values) {
				return false
			}
		case v1.NodeSelectorOpExists:
			if _, exists := node.Labels[expr.Key]; !exists {
				return false
			}
		case v1.NodeSelectorOpDoesNotExist:
			if _, exists := node.Labels[expr.Key]; exists {
				return false
			}
		case v1.NodeSelectorOpGt, v1.NodeSelectorOpLt:
			// These operator not supported.
			return true
		}
	}
	return true
}

// contains checks that the string is contains in the specified list
func contains(s string, list []string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
