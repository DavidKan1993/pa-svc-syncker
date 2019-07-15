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
	"testing"

	"github.com/inwinstack/pa-svc-syncker/pkg/constants"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestParseAddresses(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	tests := []struct {
		Addresses []string
		Namespace *corev1.Namespace
	}{
		{
			Addresses: []string{"172.22.132.99", "172.22.131.0/32"},
			Namespace: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test1",
					Annotations: map[string]string{
						constants.WhiteListAddressesKey: "172.22.132.99,172.22.131.0/32",
					},
				},
			},
		},
		{
			Addresses: []string{"any"},
			Namespace: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test2",
					Annotations: map[string]string{
						constants.WhiteListAddressesKey: "",
					},
				},
			},
		},
		{
			Addresses: nil,
			Namespace: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test3",
					Annotations: map[string]string{
						constants.WhiteListAddressesKey: "172.22.132.99,172.22.131.0/33",
					},
				},
			},
		},
	}

	for _, test := range tests {
		_, nserr := clientset.CoreV1().Namespaces().Create(test.Namespace)
		assert.Nil(t, nserr)

		sourceAddresses, _ := ParseAddresses(clientset, test.Namespace.Name)
		assert.Equal(t, test.Addresses, sourceAddresses)
	}
}
