package main

import (
	"context"
	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/norseto/k8s-watchdogs/pkg/k8sutils"
)

type replicaState struct {
	replicaset *appsv1.ReplicaSet
	nodes      []v1.Node
	podState   []podState
}

type podState struct {
	pod     v1.Pod
	node    v1.Node
	deleted bool
}

type rebalancer struct {
	current *replicaState
	maxRate float32
}

func (r *rebalancer) specReplicas() int32 {
	return *r.current.replicaset.Spec.Replicas
}

func (r *rebalancer) currentReplicas() int32 {
	return r.current.replicaset.Status.Replicas
}

func newRebalancer(current *replicaState) *rebalancer {
	return &rebalancer{current: current, maxRate: .25}
}

// Rebalance rebalances the pods across the nodes.
func (r *rebalancer) Rebalance(ctx context.Context, c kubernetes.Interface) (bool, error) {
	nodeCount := len(r.current.nodes)
	rs := r.current.replicaset
	sr := r.specReplicas()

	if nodeCount < 2 || sr < 2 || r.currentReplicas() < sr ||
		k8sutils.IsPodScheduleLimited(*rs) {
		return false, nil
	}

	deleted := 0
	maxDel := int(float32(sr) * r.maxRate)
	if maxDel < 1 {
		maxDel = 1
	}

	for i := 0; i < maxDel; i++ {
		node, num := r.maxPodNode()
		ave := float32(sr) / float32(nodeCount)
		if len(node) <= 0 || float32(num) < ave+1.0 {
			return deleted > 0, nil
		}
		if err := r.deleteNodePod(ctx, c, node); err != nil {
			return deleted > 0, err
		}
		deleted++
	}

	return deleted > 0, nil
}

// deleteNodePod deletes a pod.
func (r *rebalancer) deleteNodePod(ctx context.Context, c kubernetes.Interface, node string) error {
	l := len(r.current.podState)
	for i := 0; i < l; i++ {
		s := &r.current.podState[i]
		if s.node.Name == node && !s.deleted {
			log.Debug("Deleting pod " + s.pod.Name + " in " + node)
			s.deleted = true
			return k8sutils.DeletePod(ctx, c, s.pod)
		}
	}
	return nil
}

func (r *rebalancer) maxPodNode() (string, int) {
	m := map[string]int{}
	for _, n := range r.current.nodes {
		m[n.Name] = 0
	}
	for _, s := range r.current.podState {
		if !s.deleted {
			m[s.node.Name]++
		}
	}

	maxVal := 0
	maxNode := ""
	for k, v := range m {
		if v > maxVal {
			maxVal = v
			maxNode = k
		}
	}
	return maxNode, maxVal
}
