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
	inwinv1 "github.com/inwinstack/blended/apis/inwinstack/v1"
	clientset "github.com/inwinstack/blended/client/clientset/versioned"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newNAT(name, addr string, svc *v1.Service) *inwinv1.NAT {
	return &inwinv1.NAT{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: svc.Namespace,
		},
		Spec: inwinv1.NATSpec{
			Type:                 inwinv1.NATIPv4,
			SourceZones:          []string{"untrust"},
			SourceAddresses:      []string{"any"},
			DestinationAddresses: []string{addr},
			DestinationZone:      "untrust",
			ToInterface:          "any",
			Service:              "any",
			SatType:              inwinv1.NATSatNone,
			DatType:              inwinv1.NATDatStatic,
			DatAddress:           svc.Spec.ExternalIPs[0],
			Description:          "Auto sync DNAT for Kubernetes.",
		},
	}
}

func CreateNAT(c clientset.Interface, name, addr string, svc *v1.Service) error {
	if _, err := c.InwinstackV1().NATs(svc.Namespace).Get(name, metav1.GetOptions{}); err == nil {
		return nil
	}

	newNAT := newNAT(name, addr, svc)
	if _, err := c.InwinstackV1().NATs(svc.Namespace).Create(newNAT); err != nil {
		return err
	}
	return nil
}

func DeleteNAT(c clientset.Interface, name, namespace string) error {
	return c.InwinstackV1().NATs(namespace).Delete(name, nil)
}
