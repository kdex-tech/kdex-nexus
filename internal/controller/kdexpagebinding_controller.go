/*
Copyright 2025.

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

package controller

import (
	"os"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	"kdex.dev/crds/configuration"
	nexuswebhook "kdex.dev/nexus/internal/webhook"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// KDexPageBindingReconciler reconciles a KDexPageBinding object
type KDexPageBindingReconciler struct {
	client.Client
	Configuration configuration.NexusConfiguration
	RequeueDelay  time.Duration
	Scheme        *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager.
func (r *KDexPageBindingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if os.Getenv("ENABLE_WEBHOOKS") != FALSE {
		return ctrl.NewWebhookManagedBy(mgr).
			For(&kdexv1alpha1.KDexPageBinding{}).
			WithDefaulter(&nexuswebhook.KDexPageBindingDefaulter{}).
			WithValidator(&nexuswebhook.KDexPageBindingValidator{}).
			Complete()
	}

	return nil
}
