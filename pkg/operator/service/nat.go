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
	"strings"

	blendedv1 "github.com/inwinstack/blended/apis/inwinstack/v1"
	"github.com/inwinstack/pa-svc-syncker/pkg/constants"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *Controller) newNAT(name, addr string, svc *v1.Service) *blendedv1.NAT {
	return &blendedv1.NAT{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: svc.Namespace,
		},
		Spec: blendedv1.NATSpec{
			Type:                 blendedv1.NATIPv4,
			SourceZones:          c.cfg.SourceZones,
			SourceAddresses:      []string{"any"},
			DestinationAddresses: []string{addr},
			DestinationZone:      "untrust",
			ToInterface:          "any",
			Service:              "any",
			SatType:              blendedv1.NATSatNone,
			DatType:              blendedv1.NATDatStatic,
			DatAddress:           svc.Spec.ExternalIPs[0],
			Description:          "Automatically sync NAT for Kubernetes service.",
		},
	}
}

func (c *Controller) createNAT(addr string, svc *v1.Service) error {
	name := fmt.Sprintf("%s-%s", constants.PolicyPrefix, addr)
	if _, err := c.blendedset.InwinstackV1().NATs(svc.Namespace).Get(name, metav1.GetOptions{}); err == nil {
		return nil
	}

	nat := c.newNAT(name, addr, svc)
	if _, err := c.blendedset.InwinstackV1().NATs(svc.Namespace).Create(nat); err != nil {
		return err
	}
	return nil
}

func (c *Controller) deleteNAT(addr string, svc *v1.Service) error {
	name := fmt.Sprintf("%s-%s", constants.PolicyPrefix, addr)
	if _, err := c.blendedset.InwinstackV1().NATs(svc.Namespace).Get(name, metav1.GetOptions{}); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil
		}
		return err
	}
	return c.blendedset.InwinstackV1().NATs(svc.Namespace).Delete(name, nil)
}
