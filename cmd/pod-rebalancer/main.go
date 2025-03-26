package main

// Pod Rebalancer
// Deletes pod scheduled biased node.

import (
	"context"
	"flag"
	"fmt"

	"github.com/norseto/k8s-watchdogs/internal/rebalancer"
	"github.com/norseto/k8s-watchdogs/pkg/generics"
	"github.com/norseto/k8s-watchdogs/pkg/kube"
	"github.com/norseto/k8s-watchdogs/pkg/kube/client"
	"github.com/norseto/k8s-watchdogs/pkg/logger"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

func main() {
	var kubeClient kubernetes.Interface
	var namespace = metav1.NamespaceAll

	ctx := logger.WithContext(context.Background(), logger.InitLogger())
	log := logger.FromContext(ctx, "cmd", "pod-rebalancer")

	log.Info("Starting multiple pod rs Rebalancer...")

	opt := &client.Options{}
	opt.BindFlags(flag.CommandLine)
	flag.Parse()

	kubeClient, err := client.NewClientset(opt)
	if err != nil {
		log.Error(err, "failed to create kubeClient")
		panic(err)
	}

	nodes, err := kube.GetAllNodes(ctx, kubeClient)
	if err != nil {
		log.Error(err, "failed to list nodes")
		panic(err)
	}

	replicas, err := getTargetReplicaSets(ctx, kubeClient, namespace)
	if err != nil {
		log.Error(err, "failed to get replicaset")
		panic(err)
	}
	rs, err := getCandidatePods(ctx, kubeClient, namespace, nodes, replicas)
	if err != nil {
		log.Error(err, "failed to list pods")
		panic(err)
	}

	if len(rs) < 1 {
		log.Info("No rs. Do nothing.")
		return
	}

	rsstat := kube.NewReplicaSetStatus(replicas)
	rebalanced := 0
	for _, r := range rs {
		name := r.Replicaset.Name
		if rsstat.IsRollingUpdating(ctx, r.Replicaset) {
			log.Info("May under rolling update. Leave untouched", "rs", name)
			continue
		}
		result, err := rebalancer.NewRebalancer(ctx, r).Rebalance(ctx, kubeClient)
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
	replicas := generics.Convert(all.Items,
		func(rs appsv1.ReplicaSet) *appsv1.ReplicaSet { return rs.DeepCopy() }, nil)
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
		if !kube.IsPodReadyRunning(po) {
			continue
		}
		// Skip pods that cannot be safely rebalanced
		if !kube.CanBeRebalanced(&po) {
			continue
		}
		for _, rs := range replicas {
			if !kube.IsPodOwnedBy(rs, &po) {
				continue
			}
			postat := rebalancer.PodStatus{Pod: po.DeepCopy()}
			rstat, ok := rsmap[rs.UID]
			if !ok {
				rstat = &rebalancer.ReplicaState{Replicaset: rs, Nodes: nodes}
				rsmap[rs.UID] = rstat
				stats = append(stats, rstat)
			}
			rstat.PodStatus = append(rstat.PodStatus, &postat)
			break
		}
	}
	return stats, nil
}
