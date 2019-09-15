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
	"os"

	tattletalev1beta1 "tattletale/api/v1beta1"
	"tattletale/controllers"
	"tattletale/utils"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
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

	sharedConfigMapController, err := (&controllers.SharedConfigMapReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("SharedConfigMap"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr)
	if err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "SharedConfigMap")
		os.Exit(1)
	}

	sharedConfigMapCache := utils.InitReverseCache()

	// SharedConfigMap Watch
	if err := sharedConfigMapController.Watch(utils.InitSharedConfigMapWatch(sharedConfigMapCache)); err != nil {
		// return err
		setupLog.Error(err, "problem setting up sharedconfigmap watcher")
		os.Exit(1)
	}

	// Namespace Watch
	if err := sharedConfigMapController.Watch(utils.InitNamespaceWatch(sharedConfigMapCache)); err != nil {
		// return err
		setupLog.Error(err, "problem setting up namespace watcher")
		os.Exit(1)
	}

	// ConfigMap Watch
	if err := sharedConfigMapController.Watch(utils.InitConfigMapWatch(sharedConfigMapCache)); err != nil {
		// return err
		setupLog.Error(err, "problem setting up configmap watcher")
		os.Exit(1)
	}

	sharedSecretController, err := (&controllers.SharedSecretReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("SharedSecret"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr)
	if err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "SharedSecret")
		os.Exit(1)
	}

	sharedSecretCache := utils.InitReverseCache()

	// SharedSecret Watch
	if err := sharedSecretController.Watch(utils.InitSharedSecretWatch(sharedSecretCache)); err != nil {
		// return err
		setupLog.Error(err, "problem setting up sharedsecret watcher")
		os.Exit(1)
	}

	// Namespace Watch
	if err := sharedSecretController.Watch(utils.InitNamespaceWatch(sharedSecretCache)); err != nil {
		// return err
		setupLog.Error(err, "problem setting up namespace watcher")
		os.Exit(1)
	}

	// ConfigMap Watch
	if err := sharedSecretController.Watch(utils.InitSecretWatch(sharedSecretCache)); err != nil {
		// return err
		setupLog.Error(err, "problem setting up secret watcher")
		os.Exit(1)
	}

	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
