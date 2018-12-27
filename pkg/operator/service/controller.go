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

package service

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/golang/glog"
	clientset "github.com/inwinstack/blended/client/clientset/versioned/typed/inwinstack/v1"
	opkit "github.com/inwinstack/operator-kit"
	"github.com/inwinstack/pa-svc-syncker/pkg/config"
	"github.com/inwinstack/pa-svc-syncker/pkg/constants"
	"github.com/inwinstack/pa-svc-syncker/pkg/k8sutil"
	"github.com/inwinstack/pa-svc-syncker/pkg/util"
	slice "github.com/thoas/go-funk"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/tools/cache"
)

var Resource = opkit.CustomResource{
	Name:    "service",
	Plural:  "services",
	Version: "v1",
	Kind:    reflect.TypeOf(v1.Service{}).Name(),
}

type ServiceController struct {
	ctx    *opkit.Context
	client clientset.InwinstackV1Interface
	conf   *config.OperatorConfig
}

func NewController(ctx *opkit.Context, client clientset.InwinstackV1Interface, conf *config.OperatorConfig) *ServiceController {
	return &ServiceController{ctx: ctx, client: client, conf: conf}
}

func (c *ServiceController) StartWatch(namespace string, stopCh chan struct{}) error {
	resourceHandlerFuncs := cache.ResourceEventHandlerFuncs{
		AddFunc:    c.onAdd,
		UpdateFunc: c.onUpdate,
		DeleteFunc: c.onDelete,
	}

	glog.Info("Start watching service resources.")
	watcher := opkit.NewWatcher(Resource, namespace, resourceHandlerFuncs, c.ctx.Clientset.CoreV1().RESTClient())
	go watcher.Watch(&v1.Service{}, stopCh)
	return nil
}

func (c *ServiceController) onAdd(obj interface{}) {
	svc := obj.(*v1.Service).DeepCopy()
	glog.V(2).Infof("Received add on Service %s in %s namespace.", svc.Name, svc.Namespace)

	c.makeAnnotations(svc)
	if err := c.syncSpec(nil, svc); err != nil {
		glog.Errorf("Failed to sync spec on Service %s in %s namespace: %+v.", svc.Name, svc.Namespace, err)
	}
}

func (c *ServiceController) onUpdate(oldObj, newObj interface{}) {
	old := oldObj.(*v1.Service).DeepCopy()
	svc := newObj.(*v1.Service).DeepCopy()
	glog.V(2).Infof("Received update on Service %s in %s namespace.", svc.Name, svc.Namespace)

	if svc.DeletionTimestamp == nil {
		if err := c.syncSpec(old, svc); err != nil {
			glog.Errorf("Failed to sync spec on Service %s in %s namespace: %+v.", svc.Name, svc.Namespace, err)
		}
	}
}

func (c *ServiceController) onDelete(obj interface{}) {
	svc := obj.(*v1.Service).DeepCopy()
	glog.V(2).Infof("Received delete on Service %s in %s namespace.", svc.Name, svc.Namespace)

	if slice.Contains(c.conf.IgnoreNamespaces, svc.Namespace) {
		return
	}

	if len(svc.Spec.Ports) == 0 || len(svc.Spec.ExternalIPs) == 0 {
		return
	}

	if err := c.cleanup(svc); err != nil {
		glog.Errorf("Failed to cleanup on Service %s in %s namespace: %+v.", svc.Name, svc.Namespace, err)
	}
}

func (c *ServiceController) makeAnnotations(svc *v1.Service) {
	if svc.Annotations == nil {
		svc.Annotations = map[string]string{}
	}

	if _, ok := svc.Annotations[constants.AnnKeyExternalPool]; !ok {
		svc.Annotations[constants.AnnKeyExternalPool] = constants.DefaultInternetPool
	}
}

func (c *ServiceController) makeRefresh(svc *v1.Service) {
	ip := svc.Annotations[constants.AnnKeyPublicIP]
	if util.ParseIP(ip) == nil {
		svc.Annotations[constants.AnnKeyServiceRefresh] = string(uuid.NewUUID())
	}
}

func (c *ServiceController) syncSpec(old *v1.Service, svc *v1.Service) error {
	if slice.Contains(c.conf.IgnoreNamespaces, svc.Namespace) {
		return nil
	}

	if len(svc.Spec.Ports) == 0 || len(svc.Spec.ExternalIPs) == 0 {
		return nil
	}

	if err := c.allocate(svc); err != nil {
		glog.Errorf("Failed to allocate Public IP: %s.", err)
	}

	addr := svc.Annotations[constants.AnnKeyPublicIP]
	if util.ParseIP(addr) != nil {
		c.syncService(svc, addr)
		c.syncNAT(svc, addr)
		c.syncSecurity(svc, addr)
	}

	c.makeRefresh(svc)
	if _, err := c.ctx.Clientset.CoreV1().Services(svc.Namespace).Update(svc); err != nil {
		return err
	}
	return nil
}

func (c *ServiceController) allocate(svc *v1.Service) error {
	pool := svc.Annotations[constants.AnnKeyExternalPool]
	public := util.ParseIP(svc.Annotations[constants.AnnKeyPublicIP])
	if public == nil && pool != "" {
		name := svc.Spec.ExternalIPs[0]
		ip, err := c.client.IPs(svc.Namespace).Get(name, metav1.GetOptions{})
		if err == nil {
			if ip.Status.Address != "" {
				delete(svc.Annotations, constants.AnnKeyServiceRefresh)
				svc.Annotations[constants.AnnKeyPublicIP] = ip.Status.Address
			}
			return nil
		}

		newIP := k8sutil.NewIP(svc.Spec.ExternalIPs[0], svc.Namespace, pool)
		if _, err := c.client.IPs(svc.Namespace).Create(newIP); err != nil {
			return err
		}
	}
	return nil
}

// Sync the PA service object
func (c *ServiceController) syncService(svc *v1.Service, addr string) {
	portMap := map[string][]string{}
	for _, p := range svc.Spec.Ports {
		port := strconv.Itoa(int(p.Port))
		protocol := strings.ToLower(string(p.Protocol))
		portMap[protocol] = append(portMap[protocol], port)
	}

	for protocol, ports := range portMap {
		name := fmt.Sprintf("k8s-%s-%s", addr, protocol)
		portsStr := strings.Join(ports, ",")
		if err := k8sutil.CreateOrUpdateService(c.client, name, portsStr, protocol); err != nil {
			glog.Warningf("Failed to create and update Service resource: %+v.", err)
		}
	}
}

// Sync the PA NAT policies
func (c *ServiceController) syncNAT(svc *v1.Service, addr string) {
	name := fmt.Sprintf("k8s-%s", addr)
	if err := k8sutil.CreateNAT(c.client, name, addr, svc); err != nil {
		glog.Warningf("Failed to create NAT resource: %+v.", err)
	}
}

// Sync the PA Security policies
func (c *ServiceController) syncSecurity(svc *v1.Service, addr string) {
	services := []string{}
	for _, p := range svc.Spec.Ports {
		protocol := strings.ToLower(string(p.Protocol))
		services = append(services, fmt.Sprintf("k8s-%s-%s", addr, protocol))
	}

	name := fmt.Sprintf("k8s-%s", addr)
	log := c.conf.LogSettingName
	group := c.conf.GroupName
	if err := k8sutil.CreateOrUpdateSecurity(c.client, name, addr, log, group, services, svc); err != nil {
		glog.Warningf("Failed to create and update Security resource: %+v.", err)
	}
}

func (c *ServiceController) cleanup(svc *v1.Service) error {
	pool := svc.Annotations[constants.AnnKeyExternalPool]
	public := util.ParseIP(svc.Annotations[constants.AnnKeyPublicIP])
	if public != nil && pool != "" {
		svcs, err := c.ctx.Clientset.CoreV1().Services(svc.Namespace).List(metav1.ListOptions{})
		if err != nil {
			return err
		}

		k8sutil.FilterServices(svcs, public.String())
		if len(svcs.Items) != 0 {
			return nil
		}

		if err := c.client.IPs(svc.Namespace).Delete(svc.Spec.ExternalIPs[0], nil); err != nil {
			return err
		}

		name := fmt.Sprintf("k8s-%s", public.String())
		if err := c.client.Securities(svc.Namespace).Delete(name, nil); err != nil {
			glog.Warningf("Failed to delete Security resource: %+v.", err)
		}

		if err := c.client.NATs(svc.Namespace).Delete(name, nil); err != nil {
			glog.Warningf("Failed to delete NAT resource: %+v.", err)
		}

		tcpName := fmt.Sprintf("%s-tcp", name)
		if err := c.client.Services().Delete(tcpName, nil); err != nil {
			glog.Warningf("Failed to delete TCP service resource: %+v.", err)
		}

		udpName := fmt.Sprintf("%s-udp", name)
		if err := c.client.Services().Delete(udpName, nil); err != nil {
			glog.Warningf("Failed to delete UDP service resource: %+v.", err)
		}
		return nil
	}
	return nil
}
