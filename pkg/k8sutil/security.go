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

func newSecurity(name, addr, log, group string, services []string, svc *v1.Service) *inwinv1.Security {
	return &inwinv1.Security{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: svc.Namespace,
		},
		Spec: inwinv1.SecuritySpec{
			SourceZones:                     []string{"untrust"},
			SourceAddresses:                 []string{"any"},
			SourceUsers:                     []string{"any"},
			HipProfiles:                     []string{"any"},
			DestinationZones:                []string{"AI public service network"},
			DestinationAddresses:            []string{addr},
			Applications:                    []string{"any"},
			Services:                        services,
			Categories:                      []string{"any"},
			Action:                          "allow",
			IcmpUnreachable:                 false,
			DisableServerResponseInspection: false,
			LogEnd:                          true,
			LogSetting:                      log,
			Group:                           group,
			Description:                     "Auto sync Security for Kubernetes.",
		},
	}
}

func CreateSecurity(c clientset.Interface, name, addr, log, group string, services []string, svc *v1.Service) error {
	if _, err := c.InwinstackV1().Securities(svc.Namespace).Get(name, metav1.GetOptions{}); err == nil {
		return nil
	}

	newSec := newSecurity(name, addr, log, group, services, svc)
	if _, err := c.InwinstackV1().Securities(svc.Namespace).Create(newSec); err != nil {
		return err
	}
	return nil
}

func DeleteSecurity(c clientset.Interface, name, namespace string) error {
	return c.InwinstackV1().Securities(namespace).Delete(name, nil)
}
