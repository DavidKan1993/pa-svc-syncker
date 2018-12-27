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
	"strings"

	inwinv1 "github.com/inwinstack/blended/apis/inwinstack/v1"
	clientset "github.com/inwinstack/blended/client/clientset/versioned/typed/inwinstack/v1"
	"github.com/inwinstack/pa-svc-syncker/pkg/constants"
	slice "github.com/thoas/go-funk"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
)

func newService(name, ports, protocol string) *inwinv1.Service {
	return &inwinv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: inwinv1.ServiceSpec{
			SourcePort:      "",
			Protocol:        protocol,
			DestinationPort: ports,
			Description:     "Auto sync Service for Kubernetes.",
		},
	}
}

func appendPort(old string, new string) string {
	ports := strings.Split(old, ",")
	addPorts := strings.Split(new, ",")
	ports = append([]string{}, append(ports, addPorts...)...)
	return strings.Join(slice.UniqString(ports), ",")
}

func makeToCommit(svc *inwinv1.Service) {
	if svc.Annotations == nil {
		svc.Annotations = map[string]string{}
	}

	if _, ok := svc.Annotations[constants.AnnKeyServiceRefresh]; !ok {
		svc.Annotations[constants.AnnKeyServiceRefresh] = string(uuid.NewUUID())
	}
}

func CreateOrUpdateService(c clientset.InwinstackV1Interface, name, ports, protocol, namespace string) error {
	svc, err := c.Services(namespace).Get(name, metav1.GetOptions{})
	if err == nil {
		svc.Spec.Protocol = protocol
		svc.Spec.DestinationPort = appendPort(svc.Spec.DestinationPort, ports)
		makeToCommit(svc)
		if _, err := c.Services(namespace).Update(svc); err != nil {
			return err
		}
		return nil
	}

	newSvc := newService(name, ports, protocol)
	if _, err := c.Services(namespace).Create(newSvc); err != nil {
		return err
	}
	return nil
}
