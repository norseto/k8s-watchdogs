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

	nodes, err := k8sutils.GetUntaintedNodes(ctx, client)
	if err != nil {
		log.Panic(errors.Wrap(err, "failed to list nodes"))
	}

	replicasets, err := getTargetReplicaSets(client, namespace)
	if err != nil {
		log.Panic(errors.Wrap(err, "failed to list replicaset"))
	}
	rs, err := getCandidatePods(client, namespace, nodes, replicasets)
	if err != nil {
		log.Panic(errors.Wrap(err, "failed to list pods"))
	}

	if len(rs) < 1 {
		log.Info("No rs. Do nothing.")
		return
	}

	rsstat := k8sutils.NewReplicaSetStatus(replicasets)
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

// getTargetReplicaSets gets target replicaset.
// Parameter:
//
//	c *kubernetes.Clientset : clientset
//	ns string : namespace of replicaset
//
// Returns:
//
//	[]appsv1.ReplicaSet : All target replicasets that does not hace
//	                      affinity nor tolerations nor nodeselector
//	error : error if error happens
func getTargetReplicaSets(c kubernetes.Interface, ns string) ([]appsv1.ReplicaSet, error) {
	var replicasets []appsv1.ReplicaSet
	all, err := c.AppsV1().ReplicaSets(ns).List(metav1.ListOptions{IncludeUninitialized: false})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list replicaset")
	}
	replicasets = append(replicasets, all.Items...)
	return replicasets, nil
}

// getCandidatePods gets pod candidate.
func getCandidatePods(c kubernetes.Interface, ns string, nodes []v1.Node, rslist []appsv1.ReplicaSet) ([]*replicaState, error) {
	nodeMap := make(map[string]v1.Node)
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
			if !k8sutils.IsPodOwnedBy(rs, po) {
				continue
			}
			node := nodeMap[po.Spec.NodeName]
			postat := podState{pod: po, node: node}
			rstat, ok := rsmap[rs.ObjectMeta.UID]
			if !ok {
				rstat = &replicaState{replicaset: &rs, nodes: nodes}
				rsmap[rs.ObjectMeta.UID] = rstat
				stats = append(stats, rstat)
			}
			rstat.podState = append(rstat.podState, postat)
			break
		}
	}
	return stats, nil
}
