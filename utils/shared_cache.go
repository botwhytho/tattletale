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
	"fmt"
	"strings"
	"sync"

	tattletalev1beta1 "tattletale/api/v1beta1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	setupLog = ctrl.Log.WithName("setup")
)

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
	namespaceCache DependentsReverseCache
	sourcesCache   DependentsReverseCache
	targetsCache   DependentsReverseCache
}

func InitReverseCache() *SharedReverseCache {
	return &SharedReverseCache{
		namespaceCache: DependentsReverseCache{entries: map[types.NamespacedName]sets.String{}},
		sourcesCache:   DependentsReverseCache{entries: map[types.NamespacedName]sets.String{}},
		targetsCache:   DependentsReverseCache{entries: map[types.NamespacedName]sets.String{}},
	}
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
			s.targetsCache.Insert(types.NamespacedName{Namespace: v, Name: m.Spec.SourceConfigMap}, namespacedname)
		}
		// Creating/Updating Reverse Cache for Source Configmaps
		s.sourcesCache.Insert(types.NamespacedName{Namespace: m.Spec.SourceNamespace, Name: m.Spec.SourceConfigMap}, namespacedname)
		setupLog.Info("namespace cache created", "cache:", s.namespaceCache.String())
		setupLog.Info("sources cache created", "cache:", s.sourcesCache.String())
		setupLog.Info("targets cache created", "cache:", s.targetsCache.String())

	case *tattletalev1beta1.SharedSecret:
		m := o.Object.(*tattletalev1beta1.SharedSecret)
		setupLog.Info("Handling event", fmt.Sprintf("%T", t), o.Meta.GetName())
		namespacedname := strings.Join([]string{o.Meta.GetNamespace(), o.Meta.GetName()}, "/")
		// Creating/Updating Reverse Cache for Namespaces & Target ConfigMaps
		for _, v := range m.Spec.TargetNamespaces {
			// Namespaces
			s.namespaceCache.Insert(types.NamespacedName{Namespace: "", Name: v}, namespacedname)
			// ConfigMaps
			s.targetsCache.Insert(types.NamespacedName{Namespace: v, Name: m.Spec.SourceSecret}, namespacedname)
		}
		// Creating/Updating Reverse Cache for Source Configmaps
		s.sourcesCache.Insert(types.NamespacedName{Namespace: m.Spec.SourceNamespace, Name: m.Spec.SourceSecret}, namespacedname)
		setupLog.Info("namespace cache created", "cache:", s.namespaceCache.String())
		setupLog.Info("sources cache created", "cache:", s.sourcesCache.String())
		setupLog.Info("targets cache created", "cache:", s.targetsCache.String())

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
		source, ok := s.sourcesCache.GetSet(types.NamespacedName{Namespace: o.Meta.GetNamespace(), Name: o.Meta.GetName()})
		if ok {
			setupLog.Info("Handling event", fmt.Sprintf("%T", t), o.Meta.GetName())
		}
		for _, req := range source.List() {
			request.Namespace = strings.Split(req, "/")[0]
			request.Name = strings.Split(req, "/")[1]
			requests = append(requests, request)
		}

		target, ok := s.targetsCache.GetSet(types.NamespacedName{Namespace: o.Meta.GetNamespace(), Name: o.Meta.GetName()})
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
