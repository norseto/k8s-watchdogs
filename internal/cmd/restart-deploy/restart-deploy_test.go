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

package restartdeploy

import (
	"context"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"testing"
)

// TestNewCommand validates the NewCommand function
func TestNewCommand(t *testing.T) {

	t.Run("test new command", func(t *testing.T) {
		cmd := NewCommand()

		// Check command usage
		assert.Equal(t, "restart-deploy", cmd.Use)

		// Check subcommand
		assert.Equal(t, "Restart deployment", cmd.Short)
	})
}

func TestRestartDeployment(t *testing.T) {
	mockClient := fake.NewSimpleClientset()

	deployment := &v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "default",
		},
	}

	_, err := mockClient.AppsV1().Deployments("default").Create(context.TODO(), deployment, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("restart valid deployment", func(t *testing.T) {
		err := restartDeployment(context.TODO(), mockClient, "default", []string{"test-deployment"})
		if err != nil {
			t.Fatal(err)
		}
	})

	// Enter the name of deployment that does not exist
	t.Run("restart invalid deployment", func(t *testing.T) {
		err := restartDeployment(context.TODO(), mockClient, "default", []string{"invalid-deployment"})
		assert.NotNil(t, err)
	})
}
