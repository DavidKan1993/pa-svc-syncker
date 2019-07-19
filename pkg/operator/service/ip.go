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
	"github.com/thoas/go-funk"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *Controller) newIP(name, namespace, pool string) (*blendedv1.IP, error) {
	ip := &blendedv1.IP{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: blendedv1.IPSpec{
			PoolName: pool,
		},
	}
	return c.blendedset.InwinstackV1().IPs(namespace).Create(ip)
}

func (c *Controller) allocate(svc *v1.Service) error {
	pool := svc.Annotations[constants.ExternalPoolKey]
	address := net.ParseIP(svc.Annotations[constants.PublicIPKey])
	if address == nil && len(pool) > 0 {
		name := svc.Spec.ExternalIPs[0]
		ip, err := c.blendedset.InwinstackV1().IPs(svc.Namespace).Get(name, metav1.GetOptions{})
		if err == nil {
			if net.ParseIP(ip.Status.Address) != nil {
				svc.Annotations[constants.PublicIPKey] = ip.Status.Address
				if !funk.ContainsString(svc.ObjectMeta.Finalizers, constants.Finalizer) {
					svc.ObjectMeta.Finalizers = append(svc.ObjectMeta.Finalizers, constants.Finalizer)
				}
			}
			return nil
		}

		if _, err := c.newIP(name, svc.Namespace, pool); err != nil {
			return err
		}
		return fmt.Errorf("not allocate an IP")
	}
	return nil
}

func (c *Controller) deallocate(svc *v1.Service) error {
	name := svc.Spec.ExternalIPs[0]
	if _, err := c.blendedset.InwinstackV1().IPs(svc.Namespace).Get(name, metav1.GetOptions{}); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil
		}
		return err
	}
	return c.blendedset.InwinstackV1().IPs(svc.Namespace).Delete(name, nil)
}
