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

package namespace

import (
	"reflect"
	"strings"

	"github.com/golang/glog"
	clientset "github.com/inwinstack/blended/client/clientset/versioned"
	opkit "github.com/inwinstack/operator-kit"
	"github.com/inwinstack/pa-svc-syncker/pkg/config"
	"github.com/inwinstack/pa-svc-syncker/pkg/constants"
	"github.com/inwinstack/pa-svc-syncker/pkg/k8sutil"
	slice "github.com/thoas/go-funk"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

var Resource = opkit.CustomResource{
	Name:    "namespace",
	Plural:  "namespaces",
	Version: "v1",
	Kind:    reflect.TypeOf(v1.Namespace{}).Name(),
}

type NamespaceController struct {
	ctx    *opkit.Context
	client clientset.Interface
	cfg    *config.OperatorConfig
}

func NewController(ctx *opkit.Context, client clientset.Interface, cfg *config.OperatorConfig) *NamespaceController {
	return &NamespaceController{ctx: ctx, client: client, cfg: cfg}
}

func (c *NamespaceController) StartWatch(namespace string, stopCh chan struct{}) error {
	resourceHandlerFuncs := cache.ResourceEventHandlerFuncs{
		AddFunc:    c.onAdd,
		UpdateFunc: c.onUpdate,
	}

	glog.Info("Start watching service resources.")
	watcher := opkit.NewWatcher(Resource, namespace, resourceHandlerFuncs, c.ctx.Clientset.CoreV1().RESTClient())
	go watcher.Watch(&v1.Namespace{}, stopCh)
	return nil
}

func (c *NamespaceController) onAdd(obj interface{}) {
	ns := obj.(*v1.Namespace).DeepCopy()
	glog.V(2).Infof("Received add on %s namespace.", ns.Name)

	if ns.Status.Phase == v1.NamespaceActive {
		if err := c.updateSecurityPolicies(ns); err != nil {
			glog.Errorf("Failed to add sources addresss to all policies on %s namespace.: %+v.", ns.Name, err)
		}
	}
}

func (c *NamespaceController) onUpdate(oldObj, newObj interface{}) {
	ns := newObj.(*v1.Namespace).DeepCopy()
	glog.V(2).Infof("Received update on %s namespace.", ns.Name)

	if ns.Status.Phase == v1.NamespaceActive {
		if err := c.updateSecurityPolicies(ns); err != nil {
			glog.Errorf("Failed to add sources addresss to all policies on %s namespace.: %+v.", ns.Name, err)
		}
	}
}

func (c *NamespaceController) updateSecurityPolicies(ns *v1.Namespace) error {
	if slice.Contains(c.cfg.IgnoreNamespaces, ns.Name) {
		return nil
	}

	addrs := []string{"any"}
	if value, ok := ns.Annotations[constants.AnnKeyWhiteListAddresses]; ok {
		addrString := strings.TrimSpace(value)
		if len(addrString) > 0 {
			addrs = strings.Split(addrString, ",")
		}
	}
	if err := k8sutil.UpdateSecuritiesSourceIPs(c.client, ns.Name, addrs); err != nil {
		return err
	}
	return nil
}
