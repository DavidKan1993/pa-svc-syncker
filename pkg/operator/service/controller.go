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
	"context"
	"fmt"
	"net"
	"time"

	"github.com/golang/glog"
	blended "github.com/inwinstack/blended/generated/clientset/versioned"
	"github.com/inwinstack/pa-svc-syncker/pkg/config"
	"github.com/inwinstack/pa-svc-syncker/pkg/constants"
	"github.com/thoas/go-funk"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	informerv1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	listerv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// Controller represents the controller of service
type Controller struct {
	cfg *config.Config

	clientset  kubernetes.Interface
	blendedset blended.Interface
	lister     listerv1.ServiceLister
	synced     cache.InformerSynced
	queue      workqueue.RateLimitingInterface
}

// NewController creates an instance of the service controller
func NewController(
	cfg *config.Config,
	clientset kubernetes.Interface,
	blendedset blended.Interface,
	informer informerv1.ServiceInformer) *Controller {

	controller := &Controller{
		cfg:        cfg,
		clientset:  clientset,
		blendedset: blendedset,
		lister:     informer.Lister(),
		synced:     informer.Informer().HasSynced,
		queue:      workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Services"),
	}
	glog.Info("Setting up the Service event handlers.")

	informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueue,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueue(new)
		},
	})
	return controller
}

// Run serves the service controller
func (c *Controller) Run(ctx context.Context, threadiness int) error {
	glog.Info("Starting Service controller")
	glog.Info("Waiting for Service informer caches to sync")
	if ok := cache.WaitForCacheSync(ctx.Done(), c.synced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	glog.Info("Starting Service workers")
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, ctx.Done())
	}

	glog.Info("Started Service workers")
	return nil
}

// Stop stops the service controller
func (c *Controller) Stop() {
	glog.Info("Stopping the Service controller")
	c.queue.ShutDown()
}

func (c *Controller) runWorker() {
	defer utilruntime.HandleCrash()
	for c.processNextWorkItem() {
	}
}

func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.queue.Get()
	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		defer c.queue.Done(obj)
		key, ok := obj.(string)
		if !ok {
			c.queue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("Service controller expected string in workqueue but got %#v", obj))
			return nil
		}

		if err := c.reconcile(key); err != nil {
			c.queue.AddRateLimited(key)
			return fmt.Errorf("Service controller error syncing '%s': %s, requeuing", key, err.Error())
		}

		c.queue.Forget(obj)
		glog.V(2).Infof("Service controller successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}
	return true
}

func (c *Controller) enqueue(obj interface{}) {
	svc := obj.(*v1.Service).DeepCopy()
	if funk.Contains(c.cfg.IgnoreNamespaces, svc.Namespace) {
		glog.V(3).Infof("Service controller ignored '%s/%s'", svc.Namespace, svc.Name)
		return
	}

	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.queue.Add(key)
}

func (c *Controller) reconcile(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return err
	}

	svc, err := c.lister.Services(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("service '%s' in work queue no longer exists", key))
			return err
		}
		return err
	}

	// If service was deleted, it will clean up NAT, and Security
	if !svc.ObjectMeta.DeletionTimestamp.IsZero() {
		if err := c.cleanup(svc); err != nil {
			return err
		}
		return nil
	}

	address := net.ParseIP(svc.Annotations[constants.PublicIPKey])
	if address == nil {
		return fmt.Errorf("failed to get the public IP")
	}

	if err := c.createNAT(address.String(), svc); err != nil {
		return err
	}

	if err := c.createSecurity(address.String(), svc); err != nil {
		return err
	}
	return nil
}

func (c *Controller) cleanup(svc *v1.Service) error {
	svcCopy := svc.DeepCopy()
	address := net.ParseIP(svcCopy.Annotations[constants.PublicIPKey])
	if address == nil {
		return nil
	}

	svcs, err := c.clientset.CoreV1().Services(svcCopy.Namespace).List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	items := funk.Filter(svcs.Items, func(s v1.Service) bool {
		v := s.Annotations[constants.PublicIPKey]
		return v == address.String()
	})

	// If this namespace has other services are used the same public IP,
	// it will not release the Security and NAT.
	if len(items.([]v1.Service)) > 1 {
		return nil
	}

	if err := c.deleteNAT(address.String(), svcCopy); err != nil {
		return err
	}

	if err := c.deleteSecurity(address.String(), svcCopy); err != nil {
		return err
	}
	return nil
}
