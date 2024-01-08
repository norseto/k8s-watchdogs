package main

// Pod Rebalancer
// Deletes pod scheduled biased node.

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"github.com/norseto/k8s-watchdogs/pkg/k8sutils"
)

func main() {
	var client kubernetes.Interface
	var namespace = metav1.NamespaceAll

	log.Info("Starting multiple pod rs rebalancer...")

	ctx := context.Background()
	client, err := k8sutils.NewClientset()
	if err != nil {
		log.Panic(errors.Wrap(err, "failed to create client"))
	}

	nodes, err := k8sutils.GetAllNodes(ctx, client)
	if err != nil {
		log.Panic(errors.Wrap(err, "failed to list nodes"))
	}

	replicas, err := getTargetReplicaSets(client, namespace)
	if err != nil {
		log.Panic(errors.Wrap(err, "failed to list replicaset"))
	}
	rs, err := getCandidatePods(client, namespace, nodes, replicas)
	if err != nil {
		log.Panic(errors.Wrap(err, "failed to list pods"))
	}

	if len(rs) < 1 {
		log.Info("No rs. Do nothing.")
		return
	}

	rsstat := k8sutils.NewReplicaSetStatus(replicas)
	rebalanced := 0
	for _, r := range rs {
		name := r.replicaset.Name
		if rsstat.IsRollingUpdating(ctx, r.replicaset) {
			log.Info(fmt.Sprint("May under rolling update. Leave untouched. rs: ", name))
			continue
		}
		result, err := newRebalancer(r).Rebalance(client)
		if err != nil {
			log.Error(errors.Wrap(err, fmt.Sprint("failed to rebalance rs: ", name)))
		} else if result {
			log.Info(fmt.Sprint("Rebalanced rs: ", name))
			rebalanced++
		} else {
			log.Debug(fmt.Sprint("No need to rebalance rs: ", name))
		}
	}

	log.Info("Done multiple pod rs rebalancer. Rebalanced ", rebalanced, " ReplicaSet(s)")
}

// getTargetReplicaSets gets the target replica sets in a given namespace.
//
// Parameters:
// - c: The Kubernetes client interface.
// - ns: The namespace to search for replica sets.
//
// Returns an array of appsv1.ReplicaSet pointers and an error, if any.
func getTargetReplicaSets(c kubernetes.Interface, ns string) ([]*appsv1.ReplicaSet, error) {
	all, err := c.AppsV1().ReplicaSets(ns).List(metav1.ListOptions{IncludeUninitialized: false})
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
func getCandidatePods(c kubernetes.Interface, ns string, nodes []*v1.Node, rslist []*appsv1.ReplicaSet) ([]*replicaState, error) {
	nodeMap := make(map[string]*v1.Node)
	var stats []*replicaState
	rsmap := make(map[types.UID]*replicaState)

	for _, n := range nodes {
		nodeMap[n.Name] = n
	}

	pods, err := c.CoreV1().Pods(ns).List(metav1.ListOptions{IncludeUninitialized: false})
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprint("failed to list pod for ", ns))
	}
	for _, po := range pods.Items {
		if !k8sutils.IsPodReadyRunning(nil, po) {
			continue
		}
		for _, rs := range rslist {
			if !k8sutils.IsPodOwnedBy(rs, &po) {
				continue
			}
			node := nodeMap[po.Spec.NodeName]
			postat := podState{pod: po.DeepCopy(), node: node}
			rstat, ok := rsmap[rs.ObjectMeta.UID]
			if !ok {
				rstat = &replicaState{replicaset: rs, nodes: nodes}
				rsmap[rs.ObjectMeta.UID] = rstat
				stats = append(stats, rstat)
			}
			rstat.podState = append(rstat.podState, &postat)
			break
		}
	}
	return stats, nil
}
