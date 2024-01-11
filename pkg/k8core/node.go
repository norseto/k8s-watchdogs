package k8core

import (
	"context"
	"fmt"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// GetAllNodes retrieves a list of all nodes in the Kubernetes cluster.
// It takes a context and a client as arguments.
// It returns a slice of pointers to Node objects and an error.
func GetAllNodes(ctx context.Context, client kubernetes.Interface) ([]*v1.Node, error) {
	all, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}
	nodes := make([]*v1.Node, len(all.Items))
	for i, n := range all.Items {
		nodes[i] = n.DeepCopy()
	}
	return nodes, nil
}

// CanSchedule checks if a given pod can be scheduled on a node based on various conditions.
func CanSchedule(node *v1.Node, podSpec *v1.PodSpec) bool {
	// Check schedulability
	if node.Spec.Unschedulable {
		return false
	}

	// Check Taints and Tolerations
	if !toleratesAllTaints(node, podSpec) {
		return false
	}

	// Check nodeSelector
	if podSpec.NodeSelector != nil {
		for key, value := range podSpec.NodeSelector {
			if nodeValue, exists := node.Labels[key]; !exists || nodeValue != value {
				return false
			}
		}
	}

	// Check NodeAffinity
	if podSpec.Affinity != nil && podSpec.Affinity.NodeAffinity != nil {
		nodeAffinity := podSpec.Affinity.NodeAffinity
		if nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution != nil {
			if !nodeMatchesNodeSelector(node, nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution) {
				return false
			}
		}
	}

	return true
}

// FilterScheduleable filters the list of nodes based on whether each node can schedule the given pod spec.
// It takes a slice of node pointers and a pod spec as arguments.
// It returns a new slice of node pointers that can schedule the pod spec.
func FilterScheduleable(nodes []*v1.Node, podSpec *v1.PodSpec) []*v1.Node {
	var list []*v1.Node
	for _, node := range nodes {
		if CanSchedule(node, podSpec) {
			list = append(list, node)
		}
	}
	return list
}

// toleratesAllTaints checks whether a given node can tolerate all the taints specified in a pod's spec.
//
// Parameters:
// - node (*v1.Node): The node to be checked for taint toleration.
// - podSpec (*v1.PodSpec): The spec of the pod that contains the taints to be tolerated.
//
// Returns:
// - bool: Whether the node tolerates all the taints in the pod spec.
//
// Summary:
// This function iterates over the taints specified in the node's spec and checks whether the pod spec
// has tolerations for each taint. It returns true if all the taints are tolerated and false otherwise.
func toleratesAllTaints(node *v1.Node, podSpec *v1.PodSpec) bool {
	for _, taint := range node.Spec.Taints {
		if !toleratesTaint(podSpec, taint) {
			return false
		}
	}
	return true
}

// nodeMatchesNodeSelector checks if a node matches the given node selector.
//
// Parameters:
// - node (*v1.Node): The node to check.
// - selector (*v1.NodeSelector): The node selector to match against.
//
// Returns:
// - bool: Returns true if the node matches the selector, false otherwise.
//
// Summary:
// This function iterates over the list of node selector terms and checks if the given node matches any of them.
// If a matching term is found, it calls the `nodeSelectorTermMatches` function to perform the actual matching.
// Returns true if a matching term is found, false otherwise.
func nodeMatchesNodeSelector(node *v1.Node, selector *v1.NodeSelector) bool {
	for _, term := range selector.NodeSelectorTerms {
		if nodeSelectorTermMatches(node, &term) {
			return true
		}
	}
	return false
}

// nodeSelectorTermMatches checks whether a node matches a given NodeSelectorTerm.
//
// Parameters:
// - node (*v1.Node): The node to check.
// - term (*v1.NodeSelectorTerm): The NodeSelectorTerm to match against.
//
// Returns:
// - bool: Whether the node matches the NodeSelectorTerm.
//
// Summary:
// This function iterates through the MatchExpressions of a NodeSelectorTerm and
// checks if the given node satisfies each expression.
// It returns true if all expressions are satisfied.
// Limitations:
// This function does not support v1.NodeSelectorOpGt nor v1.NodeSelectorOpLt.
// If these selector is specified, will return false.
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
			return false
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
