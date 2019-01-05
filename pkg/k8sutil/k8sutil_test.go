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

	"github.com/inwinstack/pa-svc-syncker/pkg/constants"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func newService(name, address string) v1.Service {
	return v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Annotations: map[string]string{
				constants.AnnKeyPublicIP: address,
			},
		},
	}
}

func TestFilterServices(t *testing.T) {
	expected := &v1.ServiceList{
		Items: []v1.Service{
			newService("test1", "140.11.22.33"),
			newService("test4", "140.11.22.33"),
		},
	}

	svcs := &v1.ServiceList{
		Items: []v1.Service{
			newService("test1", "140.11.22.33"),
			newService("test2", "140.11.22.44"),
			newService("test2", "140.11.55.44"),
			newService("test4", "140.11.22.33"),
		},
	}

	FilterServices(svcs, "140.11.22.33")
	assert.Equal(t, expected, svcs)
}

func TestGetServiceList(t *testing.T) {
	client := fake.NewSimpleClientset()

	svcs := &v1.ServiceList{
		Items: []v1.Service{
			newService("test1", "140.11.22.33"),
			newService("test2", "140.11.22.33"),
		},
	}

	for _, svc := range svcs.Items {
		_, err := client.CoreV1().Services("default").Create(&svc)
		assert.Nil(t, err)
	}

	list, err := GetServiceList(client, "default")
	assert.Nil(t, err)
	assert.Equal(t, len(svcs.Items), len(list.Items))
}

func TestUpdateService(t *testing.T) {
	client := fake.NewSimpleClientset()
	svc := newService("test1", "140.11.22.33")

	csvc, err := client.CoreV1().Services("default").Create(&svc)
	assert.Nil(t, err)

	csvc.Spec.ExternalIPs = []string{"172.11.22.33"}
	usvc, err := UpdateService(client, "default", csvc)
	assert.Nil(t, err)
	assert.Equal(t, "172.11.22.33", usvc.Spec.ExternalIPs[0])
}
