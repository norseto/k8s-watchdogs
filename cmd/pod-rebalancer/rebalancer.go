package main

import (
	"context"
	"fmt"
	"github.com/norseto/k8s-watchdogs/pkg/k8core"
	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	k8s "k8s.io/client-go/kubernetes"
)

// replicaState represents the state of a replica set.
// It contains a pointer to a replica set object, an array of nodes,
// and an array of pod states.
type replicaState struct {
	replicaset *appsv1.ReplicaSet
	nodes      []*v1.Node
	podStatus  []*podStatus
}

// podStatus represents the state of a pod.
// It contains a pointer to a pod object, a pointer to a node object,
// and a boolean flag indicating whether the pod has been deleted or not.
type podStatus struct {
	pod     *v1.Pod
	node    *v1.Node
	deleted bool
}

// rebalancer represents a rebalancer object.
type rebalancer struct {
	current          *replicaState
	maxRebalanceRate float32
}

// specReplicas returns the number of replicas specified in the current ReplicaSet.
func (r *rebalancer) specReplicas() int32 {
	if r.current == nil || r.current.replicaset == nil || r.current.replicaset.Spec.Replicas == nil {
		return 0
	}
	return *r.current.replicaset.Spec.Replicas
}

// currentReplicas returns the number of replicas currently running in the ReplicaSet.
func (r *rebalancer) currentReplicas() int32 {
	if r.current == nil || r.current.replicaset == nil || r.current.replicaset.Spec.Replicas == nil {
		return 0
	}
	return r.current.replicaset.Status.Replicas
}

// filterSchedulables filters the list of scheduleable nodes based on the pod specifications.
// It returns a new list of scheduleable nodes.
// If the current replicaState is nil or the length of current.podStatus is less than 1,
// it returns the original list of nodes.
// It assigns the first pod from current.podStatus to the variable "pod".
// If "pod" is nil, it returns the original list of nodes.
// It calls the k8sutils.FilterScheduleable function to filter the list of nodes based on the pod.Spec.
// It assigns the filtered list to the current.nodes.
func (r *rebalancer) filterSchedulables() {
	if r.current == nil || len(r.current.podStatus) < 1 {
		return
	}
	firstPod := r.current.podStatus[0].pod
	if firstPod == nil {
		return
	}

	nodes := k8core.FilterScheduleable(r.current.nodes, &firstPod.Spec)
	r.current.nodes = nodes
}

// newRebalancer returns a new instance of the rebalancer struct with the provided current
// replica state and a default maxRebalanceRate of 0.25.
// The rebalancer struct contains methods for rebalancing pods across nodes in a Kubernetes cluster.
func newRebalancer(current *replicaState) *rebalancer {
	ret := &rebalancer{current: current, maxRebalanceRate: .25}
	ret.filterSchedulables()
	return ret
}

// Rebalance rebalances the pods across the nodes in the cluster.
// It returns a boolean indicating if any pods were rebalanced and an error, if any.
// The rebalancing is done by deleting pods from the node that has the maximum number of pods
// until the pod count on that node is less than or equal to the average number of pods across all nodes plus one.
// The maximum number of pods to be deleted is calculated based on the specified rebalance rate.
// If the number of nodes is less than 2, the number of replicas is less than 2,
// or the current number of replicas is less than the specified replicas,
// no rebalancing is performed and the function returns false.
func (r *rebalancer) Rebalance(ctx context.Context, client k8s.Interface) (bool, error) {
	nodeCount := len(r.current.nodes)
	sr := r.specReplicas()

	if nodeCount < 2 || sr < 2 || r.currentReplicas() < sr {
		return false, nil
	}

	deleted := 0
	maxDel := int(float32(sr) * r.maxRebalanceRate)
	if maxDel < 1 {
		maxDel = 1
	}

	for i := 0; i < maxDel; i++ {
		node, num := r.getNodeWithMaxPods()
		if num < 1 {
			return deleted > 0, nil
		}

		ave := float32(sr) / float32(nodeCount)
		if len(node) <= 0 || float32(num) < ave+1.0 {
			return deleted > 0, nil
		}
		if err := r.deletePodOnNode(ctx, client, node); err != nil {
			return deleted > 0, fmt.Errorf("failed to delete pod: %v", err)
		}
		deleted++
	}

	return deleted > 0, nil
}

// deletePodOnNode deletes a pod on specified node.
func (r *rebalancer) deletePodOnNode(ctx context.Context, client k8s.Interface, node string) error {
	l := len(r.current.podStatus)
	for i := 0; i < l; i++ {
		s := r.current.podStatus[i]
		if s.node.Name == node && !s.deleted {
			log.Debug("Deleting pod " + s.pod.Name + " in " + node)
			s.deleted = true
			return k8core.DeletePod(ctx, client, *s.pod)
		}
	}
	return nil
}

// getNodeWithMaxPods returns the node with the maximum number of non-deleted pods and the corresponding pod count.
func (r *rebalancer) getNodeWithMaxPods() (string, int) {
	if r.current == nil {
		return "", 0
	}

	podCounts := r.countPodsPerNode()

	maxPods := 0
	nodeNameWithMaxPods := ""
	for nodeName, podCount := range podCounts {
		if podCount > maxPods {
			maxPods = podCount
			nodeNameWithMaxPods = nodeName
		}
	}
	return nodeNameWithMaxPods, maxPods
}

// countPodsPerNode returns a map containing the count of pods per node in the current replica state.
func (r *rebalancer) countPodsPerNode() map[string]int {
	podCounts := make(map[string]int)
	for _, s := range r.current.podStatus {
		if s.deleted {
			continue
		}
		podCounts[s.node.Name]++
	}
	return podCounts
}
