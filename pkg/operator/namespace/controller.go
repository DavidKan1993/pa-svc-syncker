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

package namespace

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/glog"
	blended "github.com/inwinstack/blended/generated/clientset/versioned"
	"github.com/inwinstack/pa-svc-syncker/pkg/config"
	"github.com/inwinstack/pa-svc-syncker/pkg/operator/service"
	"github.com/thoas/go-funk"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	informerv1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	listerv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// Controller represents the controller of namespace
type Controller struct {
	cfg *config.Config

	clientset  kubernetes.Interface
	blendedset blended.Interface
	lister     listerv1.NamespaceLister
	synced     cache.InformerSynced
	queue      workqueue.RateLimitingInterface
}

// NewController creates an instance of the namespace controller
func NewController(
	cfg *config.Config,
	clientset kubernetes.Interface,
	blendedset blended.Interface,
	informer informerv1.NamespaceInformer) *Controller {
	controller := &Controller{
		cfg:        cfg,
		clientset:  clientset,
		blendedset: blendedset,
		lister:     informer.Lister(),
		synced:     informer.Informer().HasSynced,
		queue:      workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Namespaces"),
	}
	informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueue,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueue(new)
		},
	})
	return controller
}

// Run serves the namespace controller
func (c *Controller) Run(ctx context.Context, threadiness int) error {
	glog.Info("Starting Namespace controller")
	glog.Info("Waiting for Namespace informer caches to sync")
	if ok := cache.WaitForCacheSync(ctx.Done(), c.synced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, ctx.Done())
	}
	return nil
}

// Stop stops the namespace controller
func (c *Controller) Stop() {
	glog.Info("Stopping the Namespace controller")
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
			utilruntime.HandleError(fmt.Errorf("Namespace controller expected string in workqueue but got %#v", obj))
			return nil
		}

		if err := c.reconcile(key); err != nil {
			c.queue.AddRateLimited(key)
			return fmt.Errorf("Namespace controller error syncing '%s': %s, requeuing", key, err.Error())
		}

		c.queue.Forget(obj)
		glog.V(2).Infof("Namespace controller successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}
	return true
}

func (c *Controller) enqueue(obj interface{}) {
	ns := obj.(*v1.Namespace).DeepCopy()
	if funk.Contains(c.cfg.IgnoreNamespaces, ns.Name) {
		glog.V(3).Infof("Namespace controller ignored '%s'", ns.Name)
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
	_, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return err
	}

	if _, err := c.lister.Get(name); err != nil {
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("namespace '%s' in work queue no longer exists", key))
			return err
		}
		return err
	}

	sourceAddresses, err := service.ParseAddresses(c.clientset, name)
	if err != nil {
		return err
	}

	if err := c.updateSecurity(name, sourceAddresses); err != nil {
		return err
	}
	return nil
}
