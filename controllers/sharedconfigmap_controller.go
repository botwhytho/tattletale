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

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	tattletalev1beta1 "tattletale/api/v1beta1"
)

// SharedConfigMapReconciler reconciles a SharedConfigMap object
type SharedConfigMapReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=tattletale.tattletale.dev,resources=sharedconfigmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tattletale.tattletale.dev,resources=sharedconfigmaps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=namespaces/status,verbs=get
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete

func (r *SharedConfigMapReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("sharedconfigmap", req.NamespacedName)
	log.V(1).Info("reconciling sharedconfigmap object")

	var sharedconfigmap tattletalev1beta1.SharedConfigMap
	var namespace corev1.Namespace
	var sourceconfigmap corev1.ConfigMap

	if err := r.Get(ctx, req.NamespacedName, &sharedconfigmap); err != nil {
		log.Error(err, "unable to get sharedconfigmap")
		return ctrl.Result{}, err
	}

	// Check if source configmap actually exists, if not skip
	if err := r.Get(ctx, client.ObjectKey{Namespace: sharedconfigmap.Spec.SourceNamespace, Name: sharedconfigmap.Spec.SourceConfigMap}, &sourceconfigmap); err != nil {
		if !apierrors.IsNotFound(err) {
			log.Error(err, "unable to get configmap")
			return ctrl.Result{}, err
		} else {
			log.V(1).Info("source configmap does not exist. skipping sync.")
			return ctrl.Result{Requeue: true}, nil
		}
	}

	// Loop through target namespaces and create/update configmaps
	for _, v := range sharedconfigmap.Spec.TargetNamespaces {
		// Try and get namespace
		if err := r.Get(ctx, client.ObjectKey{Namespace: "", Name: v}, &namespace); err != nil {
			// Error out
			// TODO: func ignoreNotFound from kubebuilder book, add to utils
			if !apierrors.IsNotFound(err) {
				log.Error(err, "unable to get namespace")
				return ctrl.Result{}, err
			} else {
				// Skip if namespace does not exist
				log.V(1).Info("namespace does not exist. skipping sync", "namespace", v)
				continue
			}
		}

		configmapFound := true
		var targetconfigmap corev1.ConfigMap
		// Test if configmap exists
		if err := r.Get(ctx, client.ObjectKey{Namespace: v, Name: sharedconfigmap.Spec.SourceConfigMap}, &targetconfigmap); err != nil {
			if !apierrors.IsNotFound(err) {
				log.Error(err, "unable to get configmap")
				return ctrl.Result{}, err
			}
			configmapFound = false
		}

		temp := corev1.ConfigMap{}
		temp.Name = sharedconfigmap.Spec.SourceConfigMap
		temp.Namespace = v
		temp.Data = sourceconfigmap.Data
		temp.BinaryData = sourceconfigmap.BinaryData
		newTargetConfigMap := temp.DeepCopyObject()

		// Creating configmap
		if !configmapFound {

			if err := r.Create(ctx, newTargetConfigMap); err != nil {
				log.Error(err, "unable to create configmap in target namespace")
				return ctrl.Result{}, err
			} else {
				log.V(1).Info("Succesfully created configmap", "namespace", v)
			}

		} else {
			// Updating configmap.
			// ### TODO update only if hashes have changed, downstream repercussions of redundantly updating
			// ### TODO: Think of updating status, here and in other places

			if err := r.Update(ctx, newTargetConfigMap); err != nil {
				log.Error(err, "unable to update configmap in target namespace")
				return ctrl.Result{}, err
			} else {
				log.V(1).Info("Succesfully updated configmap", "namespace", v)
			}

		}

	}

	// TODO: should we tolerate 'partial' errors
	// TODO: dealing with deletion of CRD, what to do with other objects, should be configurable
	return ctrl.Result{}, nil

}

func (r *SharedConfigMapReconciler) SetupWithManager(mgr ctrl.Manager) (controller.Controller, error) {
	return ctrl.NewControllerManagedBy(mgr).
		For(&tattletalev1beta1.SharedConfigMap{}).
		Build(r)
}
