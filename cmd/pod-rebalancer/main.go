package main

// Pod Rebalancer
// Deletes pod scheduled biased node.

import (
	"context"
	"fmt"
	"github.com/norseto/k8s-watchdogs/internal/rebalancer"
	"github.com/norseto/k8s-watchdogs/pkg/k8apps"
	"github.com/norseto/k8s-watchdogs/pkg/k8core"
	"github.com/norseto/k8s-watchdogs/pkg/k8sclient"

	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

func main() {
	var client kubernetes.Interface
	var namespace = metav1.NamespaceAll
	var ctx = context.Background()

	log.Info("Starting multiple pod rs Rebalancer...")

	client, err := k8sclient.NewClientset()
	if err != nil {
		log.Panic(fmt.Errorf("failed to create client: %w", err))
	}

	nodes, err := k8core.GetAllNodes(ctx, client)
	if err != nil {
		log.Panic(fmt.Errorf("failed to list nodes: %w", err))
	}

	replicas, err := getTargetReplicaSets(ctx, client, namespace)
	if err != nil {
		log.Panic(fmt.Errorf("failed to list replicaset: %w", err))
	}
	rs, err := getCandidatePods(ctx, client, namespace, nodes, replicas)
	if err != nil {
		log.Panic(fmt.Errorf("failed to list pods: %w", err))
	}

	if len(rs) < 1 {
		log.Info("No rs. Do nothing.")
		return
	}

	rsstat := k8apps.NewReplicaSetStatus(replicas)
	rebalanced := 0
	for _, r := range rs {
		name := r.Replicaset.Name
		if rsstat.IsRollingUpdating(ctx, r.Replicaset) {
			log.Info(fmt.Sprintf("May under rolling update. Leave untouched. rs: %s", name))
			continue
		}
		result, err := rebalancer.NewRebalancer(r).Rebalance(ctx, client)
		if err != nil {
			log.Error(fmt.Errorf("failed to rebalance rs: %s, error: %w", name, err))
		} else if result {
			log.Info(fmt.Sprintf("Rebalanced rs: %s", name))
			rebalanced++
		} else {
			log.Debug(fmt.Sprintf("No need to rebalance rs: %s", name))
		}
	}

	log.Info(fmt.Sprintf("Done multiple pod rs Rebalancer. Rebalanced %d ReplicaSet(s)", rebalanced))
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
		if !k8core.IsPodReadyRunning(po) {
			continue
		}
		for _, rs := range replicas {
			if !k8apps.IsPodOwnedBy(rs, &po) {
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
