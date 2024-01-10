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

	"github.com/pkg/errors"
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
		log.Panic(errors.Wrap(err, "failed to create client"))
	}

	nodes, err := k8core.GetAllNodes(ctx, client)
	if err != nil {
		log.Panic(errors.Wrap(err, "failed to list nodes"))
	}

	replicas, err := getTargetReplicaSets(ctx, client, namespace)
	if err != nil {
		log.Panic(errors.Wrap(err, "failed to list replicaset"))
	}
	rs, err := getCandidatePods(ctx, client, namespace, nodes, replicas)
	if err != nil {
		log.Panic(errors.Wrap(err, "failed to list pods"))
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
			log.Info(fmt.Sprint("May under rolling update. Leave untouched. rs: ", name))
			continue
		}
		result, err := rebalancer.NewRebalancer(r).Rebalance(ctx, client)
		if err != nil {
			log.Error(errors.Wrap(err, fmt.Sprint("failed to rebalance rs: ", name)))
		} else if result {
			log.Info(fmt.Sprint("Rebalanced rs: ", name))
			rebalanced++
		} else {
			log.Debug(fmt.Sprint("No need to rebalance rs: ", name))
		}
	}

	log.Info("Done multiple pod rs Rebalancer. Rebalanced ", rebalanced, " ReplicaSet(s)")
}

// getTargetReplicaSets gets target replica sets in a namespace.
func getTargetReplicaSets(ctx context.Context, client kubernetes.Interface, ns string) ([]*appsv1.ReplicaSet, error) {
	all, err := client.AppsV1().ReplicaSets(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list replicaset")
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
		return nil, errors.Wrap(err, fmt.Sprint("failed to list pod for ", ns))
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
