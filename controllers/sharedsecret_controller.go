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

// SharedSecretReconciler reconciles a SharedSecret object
type SharedSecretReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=tattletale.tattletale.dev,resources=sharedsecrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tattletale.tattletale.dev,resources=sharedsecrets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=namespaces/status,verbs=get
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete

func (r *SharedSecretReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("sharedcsecret", req.NamespacedName)
	log.V(1).Info("reconciling sharedsecret object")

	var sharedsecret tattletalev1beta1.SharedSecret
	var namespace corev1.Namespace
	var sourcesecret corev1.Secret

	if err := r.Get(ctx, req.NamespacedName, &sharedsecret); err != nil {
		log.Error(err, "unable to get sharedsecret")
		return ctrl.Result{}, err
	}

	// Check if source secret actually exists, if not skip
	if err := r.Get(ctx, client.ObjectKey{Namespace: sharedsecret.Spec.SourceNamespace, Name: sharedsecret.Spec.SourceSecret}, &sourcesecret); err != nil {
		if !apierrors.IsNotFound(err) {
			log.Error(err, "unable to get secret")
			return ctrl.Result{}, err
		} else {
			log.V(1).Info("source secret does not exist. skipping sync.")
			return ctrl.Result{}, nil
		}
	}

	// Loop through target namespaces and create/update secrets
	for _, v := range sharedsecret.Spec.Targets {
		// Try and get namespace
		if err := r.Get(ctx, client.ObjectKey{Namespace: "", Name: v.Namespace}, &namespace); err != nil {
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

		secretFound := true
		var targetsecret corev1.Secret

		secretName := ""
		if v.NewName != "" {
			secretName = v.NewName
		} else {
			secretName = sharedsecret.Spec.SourceSecret
		}
		// Test if secret exists
		if err := r.Get(ctx, client.ObjectKey{Namespace: v.Namespace, Name: secretName}, &targetsecret); err != nil {
			if !apierrors.IsNotFound(err) {
				log.Error(err, "unable to get secret")
				return ctrl.Result{}, err
			}
			secretFound = false
		}

		temp := corev1.Secret{}
		temp.Name = secretName
		temp.Namespace = v.Namespace
		temp.Data = sourcesecret.Data
		newTargetSecret := temp.DeepCopyObject()

		// Creating secret
		if !secretFound {

			if err := r.Create(ctx, newTargetSecret); err != nil {
				log.Error(err, "unable to create secret in target namespace")
				return ctrl.Result{}, err
			} else {
				log.V(1).Info("Succesfully created secret", "namespace", v)
			}

		} else {
			// Updating secret.
			// ### TODO update only if hashes have changed, downstream repercussions of redundantly updating
			// ### TODO: Think of updating status, here and in other places

			if err := r.Update(ctx, newTargetSecret); err != nil {
				log.Error(err, "unable to update secret in target namespace")
				return ctrl.Result{}, err
			} else {
				log.V(1).Info("Succesfully updated secret", "namespace", v)
			}

		}

	}

	// TODO: should we tolerate 'partial' errors
	// TODO: dealing with deletion of CRD, what to do with other objects, should be configurable
	return ctrl.Result{}, nil
}

func (r *SharedSecretReconciler) SetupWithManager(mgr ctrl.Manager) (controller.Controller, error) {
	return ctrl.NewControllerManagedBy(mgr).
		For(&tattletalev1beta1.SharedSecret{}).
		Build(r)
}
