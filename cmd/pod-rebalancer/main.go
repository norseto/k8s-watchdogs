package main

// Pod Rebalancer
// Deletes pod scheduled biased node.

import (
	"context"
	"fmt"
	"github.com/norseto/k8s-watchdogs/internal/rebalancer"
	"github.com/norseto/k8s-watchdogs/pkg/k8sapps"
	"github.com/norseto/k8s-watchdogs/pkg/k8sclient"
	"github.com/norseto/k8s-watchdogs/pkg/k8score"
	"github.com/norseto/k8s-watchdogs/pkg/logger"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

func main() {
	var client kubernetes.Interface
	var namespace = metav1.NamespaceAll

	ctx := logger.WithContext(context.Background(), logger.InitLogger())
	log := logger.FromContext(ctx, "cmd", "pod-rebalancer")

	log.Info("Starting multiple pod rs Rebalancer...")

	client, err := k8sclient.NewClientset()
	if err != nil {
		log.Error(err, "failed to create client")
		panic(err)
	}

	nodes, err := k8score.GetAllNodes(ctx, client)
	if err != nil {
		log.Error(err, "failed to list nodes")
		panic(err)
	}

	replicas, err := getTargetReplicaSets(ctx, client, namespace)
	if err != nil {
		log.Error(err, "failed to get replicaset")
		panic(err)
	}
	rs, err := getCandidatePods(ctx, client, namespace, nodes, replicas)
	if err != nil {
		log.Error(err, "failed to list pods")
		panic(err)
	}

	if len(rs) < 1 {
		log.Info("No rs. Do nothing.")
		return
	}

	rsstat := k8sapps.NewReplicaSetStatus(replicas)
	rebalanced := 0
	for _, r := range rs {
		name := r.Replicaset.Name
		if rsstat.IsRollingUpdating(ctx, r.Replicaset) {
			log.Info("May under rolling update. Leave untouched", "rs", name)
			continue
		}
		result, err := rebalancer.NewRebalancer(r).Rebalance(ctx, client)
		if err != nil {
			log.Error(err, "failed to rebalance", "rs", name)
		} else if result {
			log.Info("Rebalanced", "rs", name)
			rebalanced++
		} else {
			log.V(1).Info("No need to rebalance", "rs", name)
		}
	}

	log.Info("Done multiple pod rs Rebalancer", "rebalanced", rebalanced)
}

// getTargetReplicaSets gets target replica sets in a namespace.
func getTargetReplicaSets(ctx context.Context, client kubernetes.Interface, ns string) ([]*appsv1.ReplicaSet, error) {
	all, err := client.AppsV1().ReplicaSets(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list replicaset: %w", err)
	}
	replicas := make([]*appsv1.ReplicaSet, len(all.Items))
	for i, rs := range all.Items {
		replicas[i] = rs.DeepCopy()
	}
	return replicas, nil
}

// getCandidatePods gets pod candidate.
func getCandidatePods(ctx context.Context, client kubernetes.Interface, ns string, nodes []*v1.Node, replicas []*appsv1.ReplicaSet) ([]*rebalancer.ReplicaState, error) {
	nodeMap := make(map[string]*v1.Node)
	var stats []*rebalancer.ReplicaState
	rsmap := make(map[types.UID]*rebalancer.ReplicaState)

	for _, n := range nodes {
		nodeMap[n.Name] = n
	}

	pods, err := client.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pod for: %s, error: %w", ns, err)
	}
	for _, po := range pods.Items {
		if !k8score.IsPodReadyRunning(po) {
			continue
		}
		for _, rs := range replicas {
			if !k8sapps.IsPodOwnedBy(rs, &po) {
				continue
			}
			node := nodeMap[po.Spec.NodeName]
			postat := rebalancer.PodStatus{Pod: po.DeepCopy(), Node: node}
			rstat, ok := rsmap[rs.ObjectMeta.UID]
			if !ok {
				rstat = &rebalancer.ReplicaState{Replicaset: rs, Nodes: nodes}
				rsmap[rs.ObjectMeta.UID] = rstat
				stats = append(stats, rstat)
			}
			rstat.PodStatus = append(rstat.PodStatus, &postat)
			break
		}
	}
	return stats, nil
}
