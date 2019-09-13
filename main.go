/*

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
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"

	tattletalev1beta1 "tattletale/api/v1beta1"

	"tattletale/controllers"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {

	_ = tattletalev1beta1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

type DependentsReverseCache struct {
	entries map[types.NamespacedName]sets.String
	sync.RWMutex
}

func (c *DependentsReverseCache) String() (s string) {
	for k, v := range c.entries {
		s += "[ " + k.String() + " ] = ( " + strings.Join(v.List(), ", ") + " ) | "
	}
	return
}

func (c *DependentsReverseCache) Insert(t types.NamespacedName, s string) sets.String {
	c.Lock()
	defer c.Unlock()
	if _, ok := c.entries[t]; !ok {
		c.entries[t] = sets.String{}
	}
	c.entries[t].Insert(s)
	return c.entries[t]
}

func (c *DependentsReverseCache) List(t types.NamespacedName) []string {
	c.RLock()
	defer c.RUnlock()
	if _, ok := c.entries[t]; !ok {
		return []string{}
	}
	return c.entries[t].List()
}

func (c *DependentsReverseCache) Delete(t types.NamespacedName, s string) sets.String {
	c.Lock()
	defer c.Unlock()
	if _, ok := c.entries[t]; !ok {
		return sets.String{}
	}
	c.entries[t].Delete(s)
	return c.entries[t]
}

func (c *DependentsReverseCache) GetSet(t types.NamespacedName) (sets.String, bool) {
	c.RLock()
	defer c.RUnlock()
	if _, ok := c.entries[t]; !ok {
		return sets.String{}, false
	}
	return c.entries[t], true
}

type SharedReverseCache struct {
	namespaceCache       DependentsReverseCache
	sourceConfigMapCache DependentsReverseCache
	targetConfigMapCache DependentsReverseCache
}

func (s *SharedReverseCache) Map(o handler.MapObject) []reconcile.Request {
	requests := []reconcile.Request{}

	switch t := o.Object.(type) {
	case *tattletalev1beta1.SharedConfigMap:
		m := o.Object.(*tattletalev1beta1.SharedConfigMap)
		setupLog.Info("Handling event", fmt.Sprintf("%T", t), o.Meta.GetName())
		namespacedname := strings.Join([]string{o.Meta.GetNamespace(), o.Meta.GetName()}, "/")
		// Creating/Updating Reverse Cache for Namespaces & Target ConfigMaps
		for _, v := range m.Spec.TargetNamespaces {
			// Namespaces
			s.namespaceCache.Insert(types.NamespacedName{Namespace: "", Name: v}, namespacedname)
			// ConfigMaps
			s.targetConfigMapCache.Insert(types.NamespacedName{Namespace: v, Name: m.Spec.SourceConfigMap}, namespacedname)
		}
		// Creating/Updating Reverse Cache for Source Configmaps
		s.sourceConfigMapCache.Insert(types.NamespacedName{Namespace: m.Spec.SourceNamespace, Name: m.Spec.SourceConfigMap}, namespacedname)
		setupLog.Info("namespace cache created", "cache:", s.namespaceCache.String())
		setupLog.Info("source cache created", "cache:", s.sourceConfigMapCache.String())
		setupLog.Info("target cache created", "cache:", s.targetConfigMapCache.String())

	case *corev1.Namespace:
		request := reconcile.Request{}
		ns, ok := s.namespaceCache.GetSet(types.NamespacedName{Namespace: o.Meta.GetNamespace(), Name: o.Meta.GetName()})
		if ok {
			setupLog.Info("Handling event", fmt.Sprintf("%T", t), o.Meta.GetName())
		}
		for _, req := range ns.List() {
			request.Namespace = strings.Split(req, "/")[0]
			request.Name = strings.Split(req, "/")[1]
			requests = append(requests, request)
		}

	case *corev1.ConfigMap:
		request := reconcile.Request{}
		source, ok := s.sourceConfigMapCache.GetSet(types.NamespacedName{Namespace: o.Meta.GetNamespace(), Name: o.Meta.GetName()})
		if ok {
			setupLog.Info("Handling event", fmt.Sprintf("%T", t), o.Meta.GetName())
		}
		for _, req := range source.List() {
			request.Namespace = strings.Split(req, "/")[0]
			request.Name = strings.Split(req, "/")[1]
			requests = append(requests, request)
		}

		target, ok := s.targetConfigMapCache.GetSet(types.NamespacedName{Namespace: o.Meta.GetNamespace(), Name: o.Meta.GetName()})
		if ok {
			setupLog.Info("Handling event", fmt.Sprintf("%T", t), o.Meta.GetName())
		}
		for _, req := range target.List() {
			request := reconcile.Request{}
			request.Namespace = strings.Split(req, "/")[0]
			request.Name = strings.Split(req, "/")[1]
			requests = append(requests, request)
		}
	}
	return requests
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
	flag.Parse()

	ctrl.SetLogger(zap.Logger(true))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		LeaderElection:     enableLeaderElection,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	controller, err := (&controllers.SharedConfigMapReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("SharedConfigMap"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr)
	if err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "SharedConfigMap")
		os.Exit(1)
	}

	// sharedSecretController, err = (&controllers.SharedSecretReconciler{
	// 	Client: mgr.GetClient(),
	// 	Log:    ctrl.Log.WithName("controllers").WithName("SharedSecret"),
	//  Scheme: mgr.GetScheme(),
	// }).SetupWithManager(mgr)
	// if err != nil {
	// 	setupLog.Error(err, "unable to create controller", "controller", "SharedSecret")
	// 	os.Exit(1)
	// }

	sharedcache := &SharedReverseCache{
		namespaceCache:       DependentsReverseCache{entries: map[types.NamespacedName]sets.String{}},
		sourceConfigMapCache: DependentsReverseCache{entries: map[types.NamespacedName]sets.String{}},
		targetConfigMapCache: DependentsReverseCache{entries: map[types.NamespacedName]sets.String{}},
	}

	sharedConfigMapSource := &source.Kind{
		Type: &tattletalev1beta1.SharedConfigMap{},
	}

	sharedConfigMapHandler := &handler.EnqueueRequestsFromMapFunc{
		ToRequests: sharedcache,
	}

	sharedConfigMapPredicate := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool { return true },
		UpdateFunc: func(e event.UpdateEvent) bool { return true },
		DeleteFunc: func(e event.DeleteEvent) bool { return true },
	}
	// SharedConfigMap Watch
	if err := controller.Watch(sharedConfigMapSource, sharedConfigMapHandler, sharedConfigMapPredicate); err != nil {
		// return err
		setupLog.Error(err, "problem setting up sharedconfigmap watcher")
		os.Exit(1)
	}

	namespaceSource := &source.Kind{
		Type: &corev1.Namespace{},
	}

	namespaceHandler := &handler.EnqueueRequestsFromMapFunc{
		ToRequests: sharedcache,
	}

	namespacePredicate := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool { return true },
		DeleteFunc: func(e event.DeleteEvent) bool { return false },
	}

	// Namespace Watch
	if err := controller.Watch(namespaceSource, namespaceHandler, namespacePredicate); err != nil {
		// return err
		setupLog.Error(err, "problem setting up namespace watcher")
		os.Exit(1)
	}

	configMapSource := &source.Kind{
		Type: &corev1.ConfigMap{},
	}

	configMapHandler := &handler.EnqueueRequestsFromMapFunc{
		ToRequests: sharedcache,
	}

	configMapPredicate := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool { return true },
		UpdateFunc: func(e event.UpdateEvent) bool { return true },
		DeleteFunc: func(e event.DeleteEvent) bool { return true },
	}

	// ConfigMap Watch
	if err := controller.Watch(configMapSource, configMapHandler, configMapPredicate); err != nil {
		// return err
		setupLog.Error(err, "problem setting up configmap watcher")
		os.Exit(1)
	}

	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
