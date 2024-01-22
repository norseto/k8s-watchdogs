/*
MIT License

Copyright (c) 2019-2024 Norihiro Seto

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

package rebalance_pods

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
	"github.com/spf13/cobra"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

// New returns a new Cobra command for re-balancing pods.
func New() *cobra.Command {
	return &cobra.Command{
		Use:   "rebalance-pods",
		Short: "Delete bias scheduled pods",
		Run: func(cmd *cobra.Command, args []string) {
			_ = rebalancePods(cmd.Context())
		},
	}
}

func rebalancePods(ctx context.Context) error {
	var client kubernetes.Interface
	var namespace = metav1.NamespaceAll

	log := logger.FromContext(ctx)

	client, err := k8sclient.NewClientset(k8sclient.FromContext(ctx))
	if err != nil {
		log.Error(err, "failed to create client")
		return err
	}

	nodes, err := k8score.GetAllNodes(ctx, client)
	if err != nil {
		log.Error(err, "failed to list nodes")
		return err
	}

	replicas, err := getTargetReplicaSets(ctx, client, namespace)
	if err != nil {
		log.Error(err, "failed to get replicaset")
		return err
	}
	rs, err := getCandidatePods(ctx, client, namespace, nodes, replicas)
	if err != nil {
		log.Error(err, "failed to list pods")
		return err
	}

	if len(rs) < 1 {
		log.Info("No rs. Do nothing.")
		return nil
	}

	rsstat := k8sapps.NewReplicaSetStatus(replicas)
	rebalanced := 0
	for _, r := range rs {
		name := r.Replicaset.Name
		if rsstat.IsRollingUpdating(ctx, r.Replicaset) {
			log.Info("May under rolling update. Leave untouched", "rs", name)
			continue
		}
		result, err := rebalancer.NewRebalancer(ctx, r).Rebalance(ctx, client)
		if err != nil {
			log.Error(err, "failed to rebalance", "rs", name)
		} else if result {
			log.Info("Rebalanced", "rs", name)
			rebalanced++
		} else {
			log.V(1).Info("No need to rebalance", "rs", name)
		}
	}

	return nil
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