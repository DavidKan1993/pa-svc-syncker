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
	inwinclientset "github.com/inwinstack/blended/client/clientset/versioned/typed/inwinstack/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newService(name, port, protocol string) *inwinv1.Service {
	return &inwinv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: inwinv1.ServiceSpec{
			SourcePort:      "",
			Protocol:        protocol,
			DestinationPort: port,
			Description:     "Auto sync Service for Kubernetes.",
		},
	}
}

func CreateOrUpdateService(c inwinclientset.InwinstackV1Interface, name, port, protocol string) error {
	svc, err := c.Services().Get(name, metav1.GetOptions{})
	if err == nil {
		svc.Spec.Protocol = protocol
		svc.Spec.DestinationPort = port
		if _, err := c.Services().Update(svc); err != nil {
			return err
		}
		return nil
	}

	newSvc := newService(name, port, protocol)
	if _, err := c.Services().Create(newSvc); err != nil {
		return err
	}
	return nil
}
