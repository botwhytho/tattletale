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

package utils

import (
	"os"

	tattletalev1beta1 "tattletale/api/v1beta1"

	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var (
	setupLog = ctrl.Log.WithName("setup")
)

func InitSharedConfigMapWatch(cache *SharedReverseCache) (*source.Kind, *handler.EnqueueRequestsFromMapFunc, *predicate.Funcs) {

	sharedConfigMapPredicate := &predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool { return true },
		UpdateFunc: func(e event.UpdateEvent) bool { return true },
		DeleteFunc: func(e event.DeleteEvent) bool { return true },
	}

	return &source.Kind{Type: &tattletalev1beta1.SharedConfigMap{}}, &handler.EnqueueRequestsFromMapFunc{ToRequests: cache}, sharedConfigMapPredicate
}

func InitSharedSecretWatch(cache *SharedReverseCache) (*source.Kind, *handler.EnqueueRequestsFromMapFunc, *predicate.Funcs) {

	sharedSecretPredicate := &predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool { return true },
		UpdateFunc: func(e event.UpdateEvent) bool { return true },
		DeleteFunc: func(e event.DeleteEvent) bool { return true },
	}

	return &source.Kind{Type: &tattletalev1beta1.SharedSecret{}}, &handler.EnqueueRequestsFromMapFunc{ToRequests: cache}, sharedSecretPredicate
}

func InitNamespaceWatch(cache *SharedReverseCache) (*source.Kind, *handler.EnqueueRequestsFromMapFunc, *predicate.Funcs) {

	namespacePredicate := &predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool { return true },
		DeleteFunc: func(e event.DeleteEvent) bool { return false },
	}

	return &source.Kind{Type: &corev1.Namespace{}}, &handler.EnqueueRequestsFromMapFunc{ToRequests: cache}, namespacePredicate
}

func InitConfigMapWatch(cache *SharedReverseCache) (*source.Kind, *handler.EnqueueRequestsFromMapFunc, *predicate.Funcs) {

	configmapPredicate := &predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool { return true },
		UpdateFunc: func(e event.UpdateEvent) bool { return true },
		DeleteFunc: func(e event.DeleteEvent) bool { return true },
	}

	return &source.Kind{Type: &corev1.ConfigMap{}}, &handler.EnqueueRequestsFromMapFunc{ToRequests: cache}, configmapPredicate
}

func InitSecretWatch(cache *SharedReverseCache) (*source.Kind, *handler.EnqueueRequestsFromMapFunc, *predicate.Funcs) {

	secretPredicate := &predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool { return true },
		UpdateFunc: func(e event.UpdateEvent) bool { return true },
		DeleteFunc: func(e event.DeleteEvent) bool { return true },
	}

	return &source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestsFromMapFunc{ToRequests: cache}, secretPredicate
}

func InitSharedConfigMapWatchers(controller controller.Controller) {

	sharedConfigMapCache := InitReverseCache()

	// SharedConfigMap Watch
	if err := controller.Watch(InitSharedConfigMapWatch(sharedConfigMapCache)); err != nil {
		setupLog.Error(err, "problem setting up sharedconfigmap watcher")
		os.Exit(1)
	}

	// Namespace Watch
	if err := controller.Watch(InitNamespaceWatch(sharedConfigMapCache)); err != nil {
		setupLog.Error(err, "problem setting up namespace watcher")
		os.Exit(1)
	}

	// ConfigMap Watch
	if err := controller.Watch(InitConfigMapWatch(sharedConfigMapCache)); err != nil {
		setupLog.Error(err, "problem setting up configmap watcher")
		os.Exit(1)
	}
}

func InitSharedSecretWatchers(controller controller.Controller) {

	sharedSecretCache := InitReverseCache()

	// SharedSecret Watch
	if err := controller.Watch(InitSharedSecretWatch(sharedSecretCache)); err != nil {
		setupLog.Error(err, "problem setting up sharedsecret watcher")
		os.Exit(1)
	}

	// Namespace Watch
	if err := controller.Watch(InitNamespaceWatch(sharedSecretCache)); err != nil {
		setupLog.Error(err, "problem setting up namespace watcher")
		os.Exit(1)
	}

	// Secret Watch
	if err := controller.Watch(InitSecretWatch(sharedSecretCache)); err != nil {
		setupLog.Error(err, "problem setting up secret watcher")
		os.Exit(1)
	}
}
