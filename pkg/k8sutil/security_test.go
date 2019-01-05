/*
Copyright © 2018 inwinSTACK.inc
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
   http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package k8sutil

import (
	"testing"

	fake "github.com/inwinstack/blended/client/clientset/versioned/fake"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSecurity(t *testing.T) {
	svc := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: v1.ServiceSpec{
			ExternalIPs: []string{"172.11.22.33"},
		},
	}

	client := fake.NewSimpleClientset()
	assert.Nil(t, CreateSecurity(client, "test-sec", "140.11.22.33", "", "", []string{"k8s-tcp"}, svc))

	sec, err := client.InwinstackV1().Securities(svc.Namespace).Get("test-sec", metav1.GetOptions{})
	assert.Nil(t, err)
	assert.Equal(t, "140.11.22.33", sec.Spec.DestinationAddresses[0])
	assert.Equal(t, []string{"k8s-tcp"}, sec.Spec.Services)
	assert.Nil(t, DeleteSecurity(client, "test-sec", svc.Namespace))
}
