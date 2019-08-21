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
		log.V(1).Info("source configmap does not exist. skipping sync.")
		return ctrl.Result{Requeue: true}, nil
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
				log.V(1).Info(v, "namespace does not exist. skipping sync here")
				continue
			}
		}

		configmapFound := true
		// Test if configmap exists
		if err := r.Get(ctx, client.ObjectKey{Namespace: v, Name: sharedconfigmap.Spec.SourceConfigMap}, &configmap); err != nil {
			if !apierrors.IsNotFound(err) {
				log.Error(err, "unable to get pod")
				return ctrl.Result{}, err
			}
			configmapFound = false
		}

		var targetconfigmap corev1.ConfigMap
		targetconfigmap.ObjectMeta = sourceconfigmap.ObjectMeta
		targetconfigmap.Name = sharedconfigmap.Spec.SourceConfigMap
		targetconfigmap.Namespace = v
		targetconfigmap.Data = sourceconfigmap.Data
		targetconfigmap.BinaryData = sourceconfigmap.BinaryData

		// Creating configmap
		if !configmapFound {

			// Setting owner reference to sharedconfigmap object ### TODO: change garbage collection by flags
			if err := ctrl.SetControllerReference(&sharedconfigmap, &targetconfigmap, r.Scheme); err != nil {
				log.Error(err, "unable to set configmap's owner reference")
				return ctrl.Result{}, err
			}

			if err := r.Create(ctx, &targetconfigmap); err != nil {
				log.Error(err, "unable to create configmap in target namespace")
				return ctrl.Result{}, err
			}

		} else {
			// Updating configmap. ###TODO update only if hashes have changed, donwstream repercussions of redundantly updating
			// ###TODO: Think of updating status, here and other places

			if err := r.Update(ctx, &targetconfigmap); err != nil {
				log.Error(err, "unable to update configmap in target namespace")
				return ctrl.Result{}, err
			}

		}

	}

	// TODO: should we tolerate 'partial' errors
	return ctrl.Result{}, nil

}

func (r *SharedConfigMapReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&tattletalev1beta1.SharedConfigMap{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}
