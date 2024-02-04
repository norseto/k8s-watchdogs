/*
MIT License

Copyright (c) 2024 Norihiro Seto

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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestRestartDeployment(t *testing.T) {
	ctx := context.TODO()

	// Create a fake client
	client := fake.NewSimpleClientset()

	// Create a deployment for testing
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "test-namespace",
		},
	}

	// Add the deployment to the fake client
	_, err := client.AppsV1().Deployments(dep.Namespace).Create(ctx, dep, metav1.CreateOptions{})
	assert.NoError(t, err)

	// Call the RestartDeployment function
	err = RestartDeployment(ctx, client, dep)
	assert.NoError(t, err)

	// Get the updated deployment
	updatedDep, err := client.AppsV1().Deployments(dep.Namespace).Get(ctx, dep.Name, metav1.GetOptions{})
	assert.NoError(t, err)

	// Check if the annotation was added
	_, ok := updatedDep.Spec.Template.ObjectMeta.Annotations["kubectl.kubernetes.io/restartedAt"]
	assert.True(t, ok)

	// Check if the annotation value is a valid time format
	_, err = time.Parse(time.RFC3339, updatedDep.Spec.Template.ObjectMeta.Annotations["kubectl.kubernetes.io/restartedAt"])
	assert.NoError(t, err)

	// Clean up the test deployment
	err = client.AppsV1().Deployments(dep.Namespace).Delete(ctx, dep.Name, metav1.DeleteOptions{})
	assert.NoError(t, err)
}
