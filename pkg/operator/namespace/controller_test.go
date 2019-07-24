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

package namespace

import (
	"context"
	"reflect"
	"testing"
	"time"

	blendedv1 "github.com/inwinstack/blended/apis/inwinstack/v1"
	blendedfake "github.com/inwinstack/blended/generated/clientset/versioned/fake"
	"github.com/inwinstack/pa-svc-syncker/pkg/config"
	"github.com/inwinstack/pa-svc-syncker/pkg/constants"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
)

func TestNamespaceController(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cfg := &config.Config{
		Threads: 2,
	}

	clientset := fake.NewSimpleClientset()
	blendedset := blendedfake.NewSimpleClientset()
	informer := informers.NewSharedInformerFactory(clientset, 0)

	controller := NewController(cfg, clientset, blendedset, informer.Core().V1().Namespaces())
	go informer.Start(ctx.Done())
	assert.Nil(t, controller.Run(ctx, cfg.Threads))

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test1",
		},
	}
	_, err := clientset.CoreV1().Namespaces().Create(ns)
	assert.Nil(t, err)

	sec := &blendedv1.Security{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sec",
			Namespace: ns.Name,
		},
		Spec: blendedv1.SecuritySpec{
			SourceZones:          []string{"untrust"},
			SourceAddresses:      []string{"any"},
			SourceUsers:          []string{"any"},
			HipProfiles:          []string{"any"},
			DestinationZones:     []string{"test-zone"},
			DestinationAddresses: []string{"140.23.110.10"},
			Applications:         []string{"any"},
			Categories:           []string{"any"},
			Services:             []string{"k8s-tcp80"},
			Action:               blendedv1.SecurityAllow,
		},
	}
	createSec, err := blendedset.InwinstackV1().Securities(ns.Name).Create(sec)
	assert.Nil(t, err)

	// Update Namespace annotation
	ns, err = clientset.CoreV1().Namespaces().Get(ns.Name, metav1.GetOptions{})
	ns.Annotations = map[string]string{
		constants.WhiteListAddressesKey: "172.22.132.99,172.22.131.0/32",
	}

	_, err = clientset.CoreV1().Namespaces().Update(ns)
	assert.Nil(t, err)

	failed := true
	for start := time.Now(); time.Since(start) < 2*time.Second; {
		sec, err = blendedset.InwinstackV1().Securities(ns.Name).Get(sec.Name, metav1.GetOptions{})
		assert.Nil(t, err)
		if !reflect.DeepEqual(createSec.Spec.SourceAddresses, sec.Spec.SourceAddresses) {
			assert.Equal(t, []string{"172.22.132.99", "172.22.131.0/32"}, sec.Spec.SourceAddresses)
			failed = false
			break
		}
	}
	assert.Equal(t, false, failed, "failed to update source addresses.")

	cancel()
	controller.Stop()
}
