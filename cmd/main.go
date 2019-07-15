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

package main

import (
	"context"
	goflag "flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/golang/glog"
	blended "github.com/inwinstack/blended/client/clientset/versioned"
	"github.com/inwinstack/pa-svc-syncker/pkg/config"
	"github.com/inwinstack/pa-svc-syncker/pkg/operator"
	"github.com/inwinstack/pa-svc-syncker/pkg/version"
	flag "github.com/spf13/pflag"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	cfg        = &config.Config{}
	kubeconfig string
	ver        bool
)

func parserFlags() {
	flag.StringVarP(&kubeconfig, "kubeconfig", "", "", "Absolute path to the kubeconfig file.")
	flag.IntVarP(&cfg.Threads, "threads", "", 2, "Number of worker threads used by the controller.")
	flag.StringSliceVarP(&cfg.IgnoreNamespaces, "ignore-namespaces", "", nil, "Ignore namespaces for syncing objects.")
	flag.StringSliceVarP(&cfg.Services, "services", "", []string{"k8s-tcp", "k8s-udp"}, "The service objects of security policy.")
	flag.StringSliceVarP(&cfg.SourceZones, "source-zones", "", []string{"untrust"}, "The source zones of security policy.")
	flag.StringSliceVarP(&cfg.DestinationZones, "destination-zones", "", []string{"AI public service network"}, "The destination zones of security policy.")
	flag.StringSliceVarP(&cfg.SourceUsers, "source-users", "", []string{"any"}, "The source users of security policy.")
	flag.StringSliceVarP(&cfg.HipProfiles, "hip-profiles", "", []string{"any"}, "The hip profiles of security policy.")
	flag.StringSliceVarP(&cfg.Applications, "applications", "", []string{"any"}, "The applications of security policy.")
	flag.StringSliceVarP(&cfg.Categories, "categories", "", []string{"any"}, "The categories of security policy.")
	flag.StringVarP(&cfg.LogSettingName, "log-setting", "", "", "The log-setting name of security policy.")
	flag.StringVarP(&cfg.GroupName, "group", "", "", "The group name of security policy.")
	flag.StringVarP(&cfg.PoolName, "pool", "", "internet", "The pool name of public IP.")
	flag.BoolVarP(&ver, "version", "", false, "Display the version.")
	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	flag.Parse()
}

func restConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		cfg, err := clientcmd.BuildConfigFromFlags("master", kubeconfig)
		if err != nil {
			return nil, err
		}
		return cfg, nil
	}

	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func main() {
	defer glog.Flush()
	parserFlags()

	if ver {
		fmt.Fprintf(os.Stdout, "%s\n", version.GetVersion())
		os.Exit(0)
	}

	k8scfg, err := restConfig(kubeconfig)
	if err != nil {
		glog.Fatalf("Failed to build kubeconfig: %s", err.Error())
	}

	client, err := kubernetes.NewForConfig(k8scfg)
	if err != nil {
		glog.Fatalf("Failed to build Kubernetes client: %s", err.Error())
	}

	blendedclient, err := blended.NewForConfig(k8scfg)
	if err != nil {
		glog.Fatalf("Failed to build Blended client: %s", err.Error())
	}

	ctx, cancel := context.WithCancel(context.Background())
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	op := operator.New(cfg, client, blendedclient)
	if err := op.Run(ctx); err != nil {
		glog.Fatalf("Error serving operator instance: %s.", err)
	}

	<-signalChan
	cancel()
	op.Stop()
	glog.Infof("Shutdown signal received, exiting...")
}
