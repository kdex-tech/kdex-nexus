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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// MicroFrontEndAppReconciler reconciles a MicroFrontEndApp object
type MicroFrontEndAppReconciler struct {
	client.Client
	RequeueDelay time.Duration
	Scheme       *runtime.Scheme
}

// +kubebuilder:rbac:groups=kdex.dev,resources=microfrontendapps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kdex.dev,resources=microfrontendapps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kdex.dev,resources=microfrontendapps/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *MicroFrontEndAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	var app kdexv1alpha1.MicroFrontEndApp
	if err := r.Get(ctx, req.NamespacedName, &app); err != nil {
		log.Error(err, "unable to fetch MicroFrontEndApp")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Validate the package reference
	validated := r.validatePackageReference(ctx, &app)

	if !validated {
		if apimeta.IsStatusConditionFalse(app.Status.Conditions, kdexv1alpha1.ConditionTypeReady.String()) {
			condition := apimeta.FindStatusCondition(app.Status.Conditions, kdexv1alpha1.ConditionTypeReady.String())
			if condition.Reason == "PackageValidationFailed" {
				log.Info("reconcile failed due to failed validation", "app", app)
				return ctrl.Result{}, nil
			}
		}

		log.Info("package validation not complete")
		apimeta.SetStatusCondition(&app.Status.Conditions, *kdexv1alpha1.NewCondition(
			kdexv1alpha1.ConditionTypeReady,
			metav1.ConditionFalse,
			"PackageValidationInProgress",
			"Package validation is in progress",
		))
		if err := r.Status().Update(ctx, &app); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: r.RequeueDelay}, nil
	}

	log.Info("reconciled MicroFrontEndApp", "app", app)

	return ctrl.Result{}, nil
}

// validatePackageReference fetches, extracts, and validates the NPM package reference that contains the App.
// This is a placeholder for the actual implementation.
func (r *MicroFrontEndAppReconciler) validatePackageReference(
	ctx context.Context, app *kdexv1alpha1.MicroFrontEndApp,
) bool {
	log := logf.FromContext(ctx)

	if apimeta.IsStatusConditionTrue(app.Status.Conditions, kdexv1alpha1.ConditionTypeReady.String()) {
		return true
	}

	go func() {
		log.Info("validating package reference for MicroFrontEndApp", "app", app.Name)

		nrc := &NPMRegistryConfiguration{
			AuthData: AuthData{
				Password: "test",
				Username: "rotty3000",
			},
			Host:   "npm.docker.localhost",
			Secure: false,
		}

		packageURL := fmt.Sprintf("%s/%s", nrc.GetAddress(), app.Spec.PackageReference.Name)

		req, err := http.NewRequest("GET", packageURL, nil)
		if err != nil {
			fmt.Println("Error creating request:", err)
			return
		}

		authorization := nrc.EncodeAuthorization()
		if authorization != "" {
			req.Header.Set("Authorization", authorization)
		}

		req.Header.Set("Accept", "application/vnd.npm.formats+json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("Error sending request:", err)
			return
		}
		defer resp.Body.Close()

		fmt.Println("Response Status:", resp.Status)

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("Error reading response body:", err)
			return
		}

		packageInfo := &PackageInfo{}
		err = json.Unmarshal(body, &packageInfo)

		if err == nil {
			latestVersion := packageInfo.DistTags.Latest

			if latestVersion != "" {
				latestVersionContent, ok := packageInfo.Versions[latestVersion]

				if ok {
					err = isPackageAnESModule(&latestVersionContent)
				}
			}
		}

		if err != nil {
			apimeta.SetStatusCondition(
				&app.Status.Conditions,
				*kdexv1alpha1.NewCondition(
					kdexv1alpha1.ConditionTypeReady,
					metav1.ConditionFalse,
					"PackageValidationFailed",
					err.Error(),
				),
			)
		} else {
			apimeta.SetStatusCondition(
				&app.Status.Conditions,
				*kdexv1alpha1.NewCondition(
					kdexv1alpha1.ConditionTypeReady,
					metav1.ConditionTrue,
					kdexv1alpha1.ConditionReasonReconcileSuccess,
					"all references resolved successfully",
				),
			)
		}

		if err := r.Status().Update(ctx, app); err != nil {
			log.Error(err, "failed to update app status")
		}
	}()

	return false
}

// SetupWithManager sets up the controller with the Manager.
func (r *MicroFrontEndAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kdexv1alpha1.MicroFrontEndApp{}).
		Named("microfrontendapp").
		Complete(r)
}

func isPackageAnESModule(packageJSON *PackageJSON) error {
	if packageJSON.Browser != "" {
		return nil
	}

	if packageJSON.Type == "module" {
		return nil
	}

	if packageJSON.Exports != nil {
		browser, ok := packageJSON.Exports["browser"]

		if ok && browser != "" {
			return nil
		}

		imp, ok := packageJSON.Exports["import"]

		if ok && imp != "" {
			return nil
		}
	}

	if strings.HasSuffix(packageJSON.Main, ".mjs") {
		return nil
	}

	return fmt.Errorf("package does not contain an ES module")
}
