package main

import (
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
	pod  *v1.Pod
	node *v1.Node
}

type rebalancer struct {
	current *replicaState
}

func (r *rebalancer) specReplicas() int32 {
	return *r.current.replicaset.Spec.Replicas
}

func (r *rebalancer) currentReplicas() int32 {
	return r.current.replicaset.Status.Replicas
}

func newRebalancer(current *replicaState) *rebalancer {
	return &rebalancer{current: current}
}

func (r *rebalancer) Rebalance(c *kubernetes.Clientset) (bool, error) {
	nodeCount := len(r.current.nodes)
	rs := r.current.replicaset
	sr := r.specReplicas()

	if nodeCount < 2 || sr < 2 || r.currentReplicas() < sr ||
		k8sutils.IsPodScheduleLimeted(*rs) {
		return false, nil
	}

	node, num := r.maxPodNode()
	ave := float32(sr) / float32(nodeCount)
	if len(node) > 0 && float32(num) >= ave+1.0 {
		err := r.deleteNodePod(c, node)
		return true, err
	}

	return false, nil
}

// deleteNodePod deletes only one pod per replicaset.
func (r *rebalancer) deleteNodePod(c *kubernetes.Clientset, node string) error {
	for _, s := range r.current.podState {
		if s.node.Name == node {
			log.Debug("Deleting pod " + s.pod.Name + " in " + node)
			return k8sutils.DeletePod(c, *s.pod)
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
		m[s.node.Name]++
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
