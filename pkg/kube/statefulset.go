/*
MIT License

Copyright (c) 2019-2025 Norihiro Seto

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

package kube

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

// RestartStatefulSet restarts a statefulset by updating its template metadata annotations with the current time.
func RestartStatefulSet(ctx context.Context, client kubernetes.Interface, sts *appsv1.StatefulSet) error {
	patch := map[string]any{
		"spec": map[string]any{
			"template": map[string]any{
				"metadata": map[string]any{
					"annotations": map[string]string{
						"kubectl.kubernetes.io/restartedAt": time.Now().Format(time.RFC3339),
					},
				},
			},
		},
	}
	data, err := json.Marshal(patch)
	if err != nil {
		return fmt.Errorf("failed to marshal patch data: %w", err)
	}
	_, err = client.AppsV1().StatefulSets(sts.Namespace).Patch(ctx, sts.Name,
		types.StrategicMergePatchType, data,
		metav1.PatchOptions{FieldManager: "kubectl-rollout"})
	if err != nil {
		return fmt.Errorf("failed to restart StatefulSet %q: %w", sts.Name, err)
	}
	return nil
}
