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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SecurityParameter struct {
	Name             string
	Address          string
	Log              string
	Group            string
	Services         []string
	DestinationZones []string
	SourceAddresses  []string
}

func newSecurity(para *SecurityParameter, svc *v1.Service) *inwinv1.Security {
	return &inwinv1.Security{
		ObjectMeta: metav1.ObjectMeta{
			Name:      para.Name,
			Namespace: svc.Namespace,
		},
		Spec: inwinv1.SecuritySpec{
			SourceZones:                     []string{"untrust"},
			SourceAddresses:                 para.SourceAddresses,
			SourceUsers:                     []string{"any"},
			HipProfiles:                     []string{"any"},
			DestinationZones:                para.DestinationZones,
			DestinationAddresses:            []string{para.Address},
			Applications:                    []string{"any"},
			Services:                        para.Services,
			Categories:                      []string{"any"},
			Action:                          "allow",
			IcmpUnreachable:                 false,
			DisableServerResponseInspection: false,
			LogEnd:                          true,
			LogSetting:                      para.Log,
			Group:                           para.Group,
			Description:                     "Auto sync Security for Kubernetes.",
		},
	}
}

func CreateSecurity(c clientset.Interface, para *SecurityParameter, svc *v1.Service) error {
	if _, err := c.InwinstackV1().Securities(svc.Namespace).Get(para.Name, metav1.GetOptions{}); err == nil {
		return nil
	}

	newSec := newSecurity(para, svc)
	if _, err := c.InwinstackV1().Securities(svc.Namespace).Create(newSec); err != nil {
		return err
	}
	return nil
}

func DeleteSecurity(c clientset.Interface, name, namespace string) error {
	return c.InwinstackV1().Securities(namespace).Delete(name, nil)
}

func UpdateSecuritiesSourceIPs(c clientset.Interface, namespace string, addrs []string) error {
	secs, err := c.InwinstackV1().Securities(namespace).List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, sec := range secs.Items {
		sec.Spec.SourceAddresses = addrs
		if _, err := c.InwinstackV1().Securities(namespace).Update(&sec); err != nil {
			return err
		}
	}
	return nil
}
