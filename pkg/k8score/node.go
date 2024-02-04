/*
MIT License

Copyright (c) 2019 Norihiro Seto

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

package k8score

import (
	"context"
	"fmt"
	"github.com/norseto/k8s-watchdogs/pkg/generics"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// GetAllNodes retrieves a list of all nodes in the Kubernetes cluster.
// It takes a context and a client as arguments.
// It returns a slice of pointers to Node objects and an error.
func GetAllNodes(ctx context.Context, client kubernetes.Interface) ([]*corev1.Node, error) {
	all, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}
	nodes := make([]*corev1.Node, len(all.Items))
	for i, n := range all.Items {
		nodes[i] = n.DeepCopy()
	}
	return nodes, nil
}

// CanSchedule checks if a given pod can be scheduled on a node based on various conditions.
func CanSchedule(node *corev1.Node, podSpec *corev1.PodSpec) bool {
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
func FilterScheduleable(nodes []*corev1.Node, podSpec *corev1.PodSpec) []*corev1.Node {
	var list []*corev1.Node
	request := GetPodRequestResources(*podSpec)

	for _, node := range nodes {
		capacity, err := GetNodeResourceCapacity(node)
		if err != nil || capacity.Cpu().Cmp(*request.Cpu()) < 0 ||
			capacity.Memory().Cmp(*request.Memory()) < 0 {
			continue
		}
		if CanSchedule(node, podSpec) {
			list = append(list, node)
		}
	}
	return list
}

// toleratesAllTaints checks whether a given node can tolerate all the taints specified in a pod's spec.
//
// Parameters:
// - node (*corev1.Node): The node to be checked for taint toleration.
// - podSpec (*corev1.PodSpec): The spec of the pod that Contains the taints to be tolerated.
//
// Returns:
// - bool: Whether the node tolerates all the taints in the pod spec.
//
// Summary:
// This function iterates over the taints specified in the node's spec and checks whether the pod spec
// has tolerations for each taint. It returns true if all the taints are tolerated and false otherwise.
func toleratesAllTaints(node *corev1.Node, podSpec *corev1.PodSpec) bool {
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
// - node (*corev1.Node): The node to check.
// - selector (*corev1.NodeSelector): The node selector to match against.
//
// Returns:
// - bool: Returns true if the node matches the selector, false otherwise.
//
// Summary:
// This function iterates over the list of node selector terms and checks if the given node matches any of them.
// If a matching term is found, it calls the `nodeSelectorTermMatches` function to perform the actual matching.
// Returns true if a matching term is found, false otherwise.
func nodeMatchesNodeSelector(node *corev1.Node, selector *corev1.NodeSelector) bool {
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
// - node (*corev1.Node): The node to check.
// - term (*corev1.NodeSelectorTerm): The NodeSelectorTerm to match against.
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
func nodeSelectorTermMatches(node *corev1.Node, term *corev1.NodeSelectorTerm) bool {
	for _, expr := range term.MatchExpressions {
		switch expr.Operator {
		case corev1.NodeSelectorOpIn:
			if !generics.Contains(node.Labels[expr.Key], expr.Values) {
				return false
			}
		case corev1.NodeSelectorOpNotIn:
			if generics.Contains(node.Labels[expr.Key], expr.Values) {
				return false
			}
		case corev1.NodeSelectorOpExists:
			if _, exists := node.Labels[expr.Key]; !exists {
				return false
			}
		case corev1.NodeSelectorOpDoesNotExist:
			if _, exists := node.Labels[expr.Key]; exists {
				return false
			}
		case corev1.NodeSelectorOpGt, corev1.NodeSelectorOpLt:
			// These operator not supported.
			return false
		}
	}
	return true
}

// GetNodeResourceCapacity retrieves the allocatable resource capacity of a node.
// It takes a pointer to a Node object as an argument.
// It returns a ResourceList and an error.
func GetNodeResourceCapacity(node *corev1.Node) (corev1.ResourceList, error) {
	cpu, found := node.Status.Allocatable[corev1.ResourceCPU]
	if !found {
		return nil, fmt.Errorf("node %s has no allocatable CPU", node.Name)
	}
	mem, found := node.Status.Allocatable[corev1.ResourceMemory]
	if !found {
		return nil, fmt.Errorf("node %s has no allocatable memory", node.Name)
	}
	return corev1.ResourceList{
		corev1.ResourceCPU:    cpu,
		corev1.ResourceMemory: mem,
	}, nil
}
