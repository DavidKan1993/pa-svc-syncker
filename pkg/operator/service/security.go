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
	"fmt"
	"net"
	"strings"

	blendedv1 "github.com/inwinstack/blended/apis/inwinstack/v1"
	"github.com/inwinstack/pa-svc-syncker/pkg/constants"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func (c *Controller) newSecurity(name, addr string, sourceAddresses []string, svc *v1.Service) *blendedv1.Security {
	return &blendedv1.Security{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: svc.Namespace,
		},
		Spec: blendedv1.SecuritySpec{
			SourceZones:                     c.cfg.SourceZones,
			SourceAddresses:                 sourceAddresses,
			SourceUsers:                     c.cfg.SourceUsers,
			HipProfiles:                     c.cfg.HipProfiles,
			DestinationZones:                c.cfg.DestinationZones,
			DestinationAddresses:            []string{addr},
			Applications:                    c.cfg.Applications,
			Services:                        c.cfg.Services,
			Categories:                      c.cfg.Applications,
			Action:                          "allow",
			IcmpUnreachable:                 false,
			DisableServerResponseInspection: false,
			LogEnd:                          true,
			LogSetting:                      c.cfg.LogSettingName,
			Group:                           c.cfg.GroupName,
			Description:                     "Automatically sync Security for Kubernetes service.",
		},
	}
}

func (c *Controller) createSecurity(addr string, svc *v1.Service) error {
	name := fmt.Sprintf("%s-%s", constants.PolicyPrefix, addr)
	if _, err := c.blendedset.InwinstackV1().Securities(svc.Namespace).Get(name, metav1.GetOptions{}); err == nil {
		return nil
	}

	sources, err := ParseAddresses(c.clientset, svc.Namespace)
	if err != nil {
		return err
	}

	sec := c.newSecurity(name, addr, sources, svc)
	if _, err := c.blendedset.InwinstackV1().Securities(svc.Namespace).Create(sec); err != nil {
		return err
	}
	return nil
}

func (c *Controller) deleteSecurity(addr string, svc *v1.Service) error {
	name := fmt.Sprintf("%s-%s", constants.PolicyPrefix, addr)
	if _, err := c.blendedset.InwinstackV1().Securities(svc.Namespace).Get(name, metav1.GetOptions{}); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil
		}
		return err
	}
	return c.blendedset.InwinstackV1().Securities(svc.Namespace).Delete(name, nil)
}

// ParseAddresses parses the whitelist IP address from Namespace's annotation
func ParseAddresses(clientset kubernetes.Interface, namespace string) ([]string, error) {
	sourceAddresses := []string{"any"}
	ns, err := clientset.CoreV1().Namespaces().Get(namespace, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	if value, ok := ns.Annotations[constants.WhiteListAddressesKey]; ok {
		if s := strings.TrimSpace(value); len(s) > 0 {
			addresses := strings.Split(s, ",")
			for _, addr := range addresses {
				ip := net.ParseIP(addr)
				if ip != nil {
					continue
				}

				_, _, err := net.ParseCIDR(addr)
				if err != nil {
					return nil, err
				}
			}
			sourceAddresses = addresses
		}
	}
	return sourceAddresses, nil
}
