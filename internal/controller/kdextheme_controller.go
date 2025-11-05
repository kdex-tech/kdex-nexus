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
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	"kdex.dev/crds/render"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// KDexThemeReconciler reconciles a KDexTheme object
type KDexThemeReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=kdex.dev,resources=kdexthemes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexthemes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexthemes/finalizers,verbs=update

func (r *KDexThemeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	var theme kdexv1alpha1.KDexTheme
	if err := r.Get(ctx, req.NamespacedName, &theme); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	kdexv1alpha1.SetConditions(
		&theme.Status.Conditions,
		kdexv1alpha1.ConditionArgs{
			Degraded: &kdexv1alpha1.ConditionFields{
				Status:  metav1.ConditionFalse,
				Reason:  kdexv1alpha1.ConditionReasonReconciling,
				Message: "Reconciling",
			},
			Progressing: &kdexv1alpha1.ConditionFields{
				Status:  metav1.ConditionTrue,
				Reason:  kdexv1alpha1.ConditionReasonReconciling,
				Message: "Reconciling",
			},
			Ready: &kdexv1alpha1.ConditionFields{
				Status:  metav1.ConditionUnknown,
				Reason:  kdexv1alpha1.ConditionReasonReconciling,
				Message: "Reconciling",
			},
		},
	)
	if err := r.Status().Update(ctx, &theme); err != nil {
		return ctrl.Result{}, err
	}

	// Defer status update
	defer func() {
		theme.Status.ObservedGeneration = theme.Generation
		if err := r.Status().Update(ctx, &theme); err != nil {
			log.Error(err, "failed to update theme status")
		}
	}()

	if err := validateSpec(theme.Spec); err != nil {
		kdexv1alpha1.SetConditions(
			&theme.Status.Conditions,
			kdexv1alpha1.ConditionArgs{
				Degraded: &kdexv1alpha1.ConditionFields{
					Status:  metav1.ConditionTrue,
					Reason:  "SpecValidationFailed",
					Message: err.Error(),
				},
				Progressing: &kdexv1alpha1.ConditionFields{
					Status:  metav1.ConditionFalse,
					Reason:  "SpecValidationFailed",
					Message: "Spec invalid",
				},
				Ready: &kdexv1alpha1.ConditionFields{
					Status:  metav1.ConditionFalse,
					Reason:  "SpecValidationFailed",
					Message: "Spec invalid",
				},
			},
		)
		if err := r.Status().Update(ctx, &theme); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}

	kdexv1alpha1.SetConditions(
		&theme.Status.Conditions,
		kdexv1alpha1.ConditionArgs{
			Degraded: &kdexv1alpha1.ConditionFields{
				Status:  metav1.ConditionFalse,
				Reason:  kdexv1alpha1.ConditionReasonReconcileSuccess,
				Message: "Reconciliation successful",
			},
			Progressing: &kdexv1alpha1.ConditionFields{
				Status:  metav1.ConditionFalse,
				Reason:  kdexv1alpha1.ConditionReasonReconcileSuccess,
				Message: "Reconciliation successful",
			},
			Ready: &kdexv1alpha1.ConditionFields{
				Status:  metav1.ConditionTrue,
				Reason:  kdexv1alpha1.ConditionReasonReconcileSuccess,
				Message: "Reconciliation successful",
			},
		},
	)
	if err := r.Status().Update(ctx, &theme); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("reconciled KDexTheme")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KDexThemeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kdexv1alpha1.KDexTheme{}).
		Named("kdextheme").
		Complete(r)
}

func validateSpec(spec kdexv1alpha1.KDexThemeSpec) error {
	if spec.Image == "" || spec.RoutePath == "" {
		for _, asset := range spec.Assets {
			if asset.LinkHref == "" && asset.Style != "" {
				continue
			}

			if asset.LinkHref != "" && !strings.Contains(asset.LinkHref, "://") {
				return fmt.Errorf("linkHref %s contains relative url but no theme image was provided", asset.LinkHref)
			}
		}
	}

	if spec.Image != "" && spec.RoutePath == "" {
		return fmt.Errorf("routePath must be specified when an image is specified")
	}

	if spec.Image != "" && spec.RoutePath != "" {
		for _, asset := range spec.Assets {
			if asset.LinkHref == "" && asset.Style != "" {
				continue
			}

			if asset.LinkHref != "" &&
				!strings.Contains(asset.LinkHref, "://") &&
				!strings.HasPrefix(asset.LinkHref, spec.RoutePath) {

				return fmt.Errorf("linkHref %s is not prefixed by %s", asset.LinkHref, spec.RoutePath)
			}
		}
	}

	renderer := render.Renderer{}

	_, err := renderer.RenderOne(
		"theme-assets",
		spec.String(),
		render.DefaultTemplateData(),
	)

	return err
}
