/*
Copyright © 2018 inwinSTACK Inc

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
	"context"
	"fmt"
	"testing"
	"time"

	blendedv1 "github.com/inwinstack/blended/apis/inwinstack/v1"
	blendedfake "github.com/inwinstack/blended/client/clientset/versioned/fake"
	"github.com/inwinstack/pa-svc-syncker/pkg/config"
	"github.com/inwinstack/pa-svc-syncker/pkg/constants"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
)

func TestServiceController(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cfg := &config.Config{
		Threads:          2,
		PoolName:         "internet",
		IgnoreNamespaces: []string{"kube-system", "kube-public", "default"},
		SourceZones:      []string{"untrust"},
		DestinationZones: []string{"test"},
		SourceUsers:      []string{"any"},
		HipProfiles:      []string{"any"},
		Applications:     []string{"any"},
		Categories:       []string{"any"},
		Services:         []string{"k8s-tcp", "k8s-udp"},
		GroupName:        "",
		LogSettingName:   "",
	}

	clientset := fake.NewSimpleClientset()
	blendedset := blendedfake.NewSimpleClientset()
	informer := informers.NewSharedInformerFactory(clientset, 0)

	controller := NewController(cfg, clientset, blendedset, informer.Core().V1().Services())
	go informer.Start(ctx.Done())
	assert.Nil(t, controller.Run(ctx, cfg.Threads))

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test1",
			Annotations: map[string]string{
				constants.WhiteListAddressesKey: "172.22.132.99,172.22.131.0/32",
			},
		},
	}
	_, nserr := clientset.CoreV1().Namespaces().Create(ns)
	assert.Nil(t, nserr)

	ip := &blendedv1.IP{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "172.11.22.33",
			Namespace: ns.Name,
		},
		Spec: blendedv1.IPSpec{
			PoolName: cfg.PoolName,
		},
		Status: blendedv1.IPStatus{
			Phase:   blendedv1.IPActive,
			Address: "140.11.22.33",
		},
	}
	_, iperr := blendedset.InwinstackV1().IPs(ip.Namespace).Create(ip)
	assert.Nil(t, iperr)

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-svc",
			Namespace: ns.Name,
			Annotations: map[string]string{
				constants.ExternalPoolKey: cfg.PoolName,
			},
		},
		Spec: corev1.ServiceSpec{
			ExternalIPs: []string{"172.11.22.33"},
			Type:        corev1.ServiceTypeLoadBalancer,
			Ports: []corev1.ServicePort{
				corev1.ServicePort{
					Port:     80,
					Protocol: corev1.ProtocolTCP,
				},
			},
		},
	}
	_, svcerr := clientset.CoreV1().Services(svc.Namespace).Create(svc)
	assert.Nil(t, svcerr)

	// Check IP address
	failed := true
	for start := time.Now(); time.Since(start) < 2*time.Second; {
		svc, err := clientset.CoreV1().Services(ns.Name).Get(svc.Name, metav1.GetOptions{})
		assert.Nil(t, err)
		if address, ok := svc.Annotations[constants.PublicIPKey]; ok {
			assert.Equal(t, ip.Status.Address, address)
			failed = false
			break
		}
	}
	assert.Equal(t, false, failed, "cannot get public IP.")

	// Check NAT and Security
	failed = true
	name := fmt.Sprintf("%s-%s", constants.PolicyPrefix, ip.Status.Address)
	for start := time.Now(); time.Since(start) < 2*time.Second; {
		nat, _ := blendedset.InwinstackV1().NATs(ns.Name).Get(name, metav1.GetOptions{})
		if nat != nil {
			assert.Equal(t, ip.Status.Address, nat.Spec.DestinationAddresses[0])
			assert.Equal(t, svc.Spec.ExternalIPs[0], nat.Spec.DatAddress)
			failed = false
			break
		}
	}
	assert.Equal(t, false, failed, "cannot get NAT.")

	failed = true
	for start := time.Now(); time.Since(start) < 2*time.Second; {
		sec, _ := blendedset.InwinstackV1().Securities(ns.Name).Get(name, metav1.GetOptions{})
		if sec != nil {
			assert.Equal(t, ip.Status.Address, sec.Spec.DestinationAddresses[0])
			assert.Equal(t, cfg.Services, sec.Spec.Services)
			assert.Equal(t, cfg.DestinationZones, sec.Spec.DestinationZones)
			assert.Equal(t, []string{"172.22.132.99", "172.22.131.0/32"}, sec.Spec.SourceAddresses)
			failed = false
			break
		}
	}
	assert.Equal(t, false, failed, "cannot get Security.")

	// Test for deleting
	newSvc, _ := clientset.CoreV1().Services(ns.Name).Get(svc.Name, metav1.GetOptions{})
	assert.Nil(t, clientset.CoreV1().Services(ns.Name).Delete(svc.Name, nil))

	controller.cleanup(newSvc)

	ipList, err := blendedset.InwinstackV1().IPs(ns.Name).List(metav1.ListOptions{})
	assert.Nil(t, err)
	assert.Equal(t, 0, len(ipList.Items))

	natList, err := blendedset.InwinstackV1().NATs(ns.Name).List(metav1.ListOptions{})
	assert.Nil(t, err)
	assert.Equal(t, 0, len(natList.Items))

	secList, err := blendedset.InwinstackV1().Securities(ns.Name).List(metav1.ListOptions{})
	assert.Nil(t, err)
	assert.Equal(t, 0, len(secList.Items))

	cancel()
	controller.Stop()
}
