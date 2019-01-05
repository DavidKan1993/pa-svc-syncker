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

package service

import (
	"fmt"
	"testing"
	"time"

	inwinv1 "github.com/inwinstack/blended/apis/inwinstack/v1"

	fake "github.com/inwinstack/blended/client/clientset/versioned/fake"
	opkit "github.com/inwinstack/operator-kit"
	"github.com/inwinstack/pa-svc-syncker/pkg/config"
	"github.com/inwinstack/pa-svc-syncker/pkg/constants"
	extensionsfake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	corefake "k8s.io/client-go/kubernetes/fake"

	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestController(t *testing.T) {
	client := fake.NewSimpleClientset()
	coreClient := corefake.NewSimpleClientset()
	extensionsClient := extensionsfake.NewSimpleClientset()

	ip := &inwinv1.IP{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "172.11.22.33",
			Namespace: "default",
		},
		Spec: inwinv1.IPSpec{
			PoolName:        "internet",
			UpdateNamespace: false,
		},
		Status: inwinv1.IPStatus{
			Phase:   inwinv1.IPActive,
			Address: "140.11.22.33",
		},
	}
	_, svcerr := client.InwinstackV1().IPs("default").Create(ip)
	assert.Nil(t, svcerr)

	svc := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-svc",
			Namespace: "default",
			Annotations: map[string]string{
				constants.AnnKeyExternalPool: "internet",
			},
		},
		Spec: v1.ServiceSpec{
			ExternalIPs: []string{"172.11.22.33"},
			Type:        v1.ServiceTypeLoadBalancer,
			Ports: []v1.ServicePort{
				v1.ServicePort{
					Port:     80,
					Protocol: v1.ProtocolTCP,
				},
			},
		},
	}
	_, iperr := coreClient.CoreV1().Services("default").Create(svc)
	assert.Nil(t, iperr)

	ctx := &opkit.Context{
		Clientset:             coreClient,
		APIExtensionClientset: extensionsClient,
		Interval:              500 * time.Millisecond,
		Timeout:               60 * time.Second,
	}

	conf := &config.OperatorConfig{
		IgnoreNamespaces: []string{"kube-system", "kube-public"},
		Retry:            5,
		Services:         []string{"k8s-tcp", "k8s-udp"},
		GroupName:        "",
		LogSettingName:   "",
	}
	controller := NewController(ctx, client, conf)

	// Test onAdd
	controller.onAdd(svc)

	usvc, err := coreClient.CoreV1().Services("default").Get("test-svc", metav1.GetOptions{})
	assert.Nil(t, err)

	publicIP, ok := usvc.Annotations[constants.AnnKeyPublicIP]
	assert.True(t, ok)
	assert.Equal(t, ip.Status.Address, publicIP)

	// Test onUpdate
	controller.onUpdate(svc, usvc)

	name := fmt.Sprintf("k8s-%s", ip.Status.Address)
	nat, err := client.InwinstackV1().NATs("default").Get(name, metav1.GetOptions{})
	assert.Nil(t, err)
	assert.Equal(t, ip.Status.Address, nat.Spec.DestinationAddresses[0])
	assert.Equal(t, usvc.Spec.ExternalIPs[0], nat.Spec.DatAddress)

	sec, err := client.InwinstackV1().Securities("default").Get(name, metav1.GetOptions{})
	assert.Equal(t, ip.Status.Address, sec.Spec.DestinationAddresses[0])
	assert.Equal(t, []string{"k8s-tcp", "k8s-udp"}, sec.Spec.Services)

	// Test onDelete
	assert.Nil(t, coreClient.CoreV1().Services("default").Delete("test-svc", nil))
	controller.onDelete(usvc)

	_, iperr1 := client.InwinstackV1().IPs("default").Get(ip.Name, metav1.GetOptions{})
	assert.NotNil(t, iperr1)

	_, naterr := client.InwinstackV1().NATs("default").Get(name, metav1.GetOptions{})
	assert.NotNil(t, naterr)

	_, secerr := client.InwinstackV1().Securities("default").Get(name, metav1.GetOptions{})
	assert.NotNil(t, secerr)
}
