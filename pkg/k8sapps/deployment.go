package k8sapps

import (
	"context"
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"time"
)

const (
	restartPatchTemplate = `{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"%v"}}}}}`
)

// RestartDeployment restarts a deployment by updating its template metadata annotations with the current time.
func RestartDeployment(ctx context.Context, client kubernetes.Interface, dep *appsv1.Deployment) error {
	data := fmt.Sprintf(restartPatchTemplate, time.Now().Format(time.RFC3339))
	_, err := client.AppsV1().Deployments(dep.Namespace).Patch(ctx, dep.Name,
		types.StrategicMergePatchType, []byte(data),
		metav1.PatchOptions{FieldManager: "kubectl-rollout"})
	return err
}
