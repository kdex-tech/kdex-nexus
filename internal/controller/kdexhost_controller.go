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
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	"kdex.dev/crds/configuration"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const hostFinalizerName = "kdex.dev/kdex-nexus-host-finalizer"

// KDexHostReconciler reconciles a KDexHost object
type KDexHostReconciler struct {
	client.Client
	Configuration configuration.NexusConfiguration
	RequeueDelay  time.Duration
	Scheme        *runtime.Scheme
}

// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=httproutes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete

// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles,verbs=get;list;watch;create;update;patch;delete

// +kubebuilder:rbac:groups=kdex.dev,resources=kdexhosts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexhosts/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexhosts/finalizers,verbs=update
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexhostcontrollers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexscriptlibraries,verbs=get;list;watch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexthemes,verbs=get;list;watch

func (r *KDexHostReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	var host kdexv1alpha1.KDexHost
	if err := r.Get(ctx, req.NamespacedName, &host); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if host.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(&host, hostFinalizerName) {
			controllerutil.AddFinalizer(&host, hostFinalizerName)
			if err := r.Update(ctx, &host); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{Requeue: true}, nil
		}
	} else {
		if controllerutil.ContainsFinalizer(&host, hostFinalizerName) {
			hostController := &kdexv1alpha1.KDexHostController{
				ObjectMeta: metav1.ObjectMeta{
					Name:      host.Name,
					Namespace: host.Namespace,
				},
			}
			if err := r.Delete(ctx, hostController); client.IgnoreNotFound(err) != nil {
				return ctrl.Result{}, err
			}

			controllerutil.RemoveFinalizer(&host, hostFinalizerName)
			if err := r.Update(ctx, &host); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	kdexv1alpha1.SetConditions(
		&host.Status.Conditions,
		kdexv1alpha1.ConditionStatuses{
			Degraded:    metav1.ConditionFalse,
			Progressing: metav1.ConditionTrue,
			Ready:       metav1.ConditionUnknown,
		},
		kdexv1alpha1.ConditionReasonReconciling,
		"Reconciling",
	)
	if err := r.Status().Update(ctx, &host); err != nil {
		return ctrl.Result{}, err
	}

	// Defer status update
	defer func() {
		host.Status.ObservedGeneration = host.Generation
		if err := r.Status().Update(ctx, &host); err != nil {
			log.Info("failed to update status", "err", err)
		}
	}()

	_, shouldReturn, r1, err := resolveTheme(ctx, r.Client, &host, &host.Status.Conditions, host.Spec.DefaultThemeRef, r.RequeueDelay)
	if shouldReturn {
		return r1, err
	}

	_, shouldReturn, r1, err = resolveScriptLibrary(ctx, r.Client, &host, &host.Status.Conditions, host.Spec.ScriptLibraryRef, r.RequeueDelay)
	if shouldReturn {
		return r1, err
	}

	return ctrl.Result{}, r.innerReconcile(ctx, &host)
}

// SetupWithManager sets up the controller with the Manager.
func (r *KDexHostReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kdexv1alpha1.KDexHost{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&kdexv1alpha1.KDexHostController{}).
		Owns(&rbacv1.Role{}).
		Owns(&rbacv1.RoleBinding{}).
		Watches(
			&kdexv1alpha1.KDexScriptLibrary{},
			handler.EnqueueRequestsFromMapFunc(r.findHostsForScriptLibrary),
		).
		Watches(
			&kdexv1alpha1.KDexTheme{},
			handler.EnqueueRequestsFromMapFunc(r.findHostsForTheme)).
		Named("kdexhost").
		Complete(r)
}

func (r *KDexHostReconciler) innerReconcile(ctx context.Context, host *kdexv1alpha1.KDexHost) error {
	log := logf.FromContext(ctx)

	if err := r.createOrUpdateFocusedController(ctx, host); err != nil {
		return err
	}

	if err := r.createOrUpdateHostControllerResource(ctx, host); err != nil {
		return err
	}

	kdexv1alpha1.SetConditions(
		&host.Status.Conditions,
		kdexv1alpha1.ConditionStatuses{
			Degraded:    metav1.ConditionFalse,
			Progressing: metav1.ConditionFalse,
			Ready:       metav1.ConditionTrue,
		},
		kdexv1alpha1.ConditionReasonReconcileSuccess,
		"Reconciliation successful",
	)
	if err := r.Status().Update(ctx, host); err != nil {
		return err
	}

	log.Info("reconciled KDexHost")

	return nil
}

func (r *KDexHostReconciler) createOrUpdateFocusedController(
	ctx context.Context,
	host *kdexv1alpha1.KDexHost,
) error {
	if err := r.createOrUpdateConfigMap(ctx, host); err != nil {
		return err
	}

	if err := r.createOrUpdateRole(ctx, host); err != nil {
		return err
	}

	if err := r.createOrUpdateServiceAccount(ctx, host); err != nil {
		return err
	}

	if err := r.createOrUpdateRoleBinding(ctx, host); err != nil {
		return err
	}

	if err := r.createOrUpdateDeployment(ctx, host); err != nil {
		return err
	}

	return r.createOrUpdateService(ctx, host)
}

func (r *KDexHostReconciler) createOrUpdateConfigMap(
	ctx context.Context,
	host *kdexv1alpha1.KDexHost,
) error {
	codecs := serializer.NewCodecFactory(r.Scheme)

	info, ok := runtime.SerializerInfoForMediaType(codecs.SupportedMediaTypes(), "application/yaml")
	if !ok {
		return fmt.Errorf("no YAML serializer found")
	}

	encoder := codecs.EncoderForVersion(info.Serializer, configuration.GroupVersion)

	var buf bytes.Buffer
	if err := encoder.Encode(&r.Configuration, &buf); err != nil {
		return fmt.Errorf("failed to encode object to YAML: %w", err)
	}

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      host.Name,
			Namespace: host.Namespace,
		},
	}

	if _, err := ctrl.CreateOrUpdate(ctx, r.Client, configMap, func() error {
		configMap.Annotations = host.Annotations
		configMap.Labels = host.Labels
		if configMap.Labels == nil {
			configMap.Labels = make(map[string]string)
		}
		configMap.Labels["app.kubernetes.io/name"] = "kdex-web"
		configMap.Labels["kdex.dev/focus-host"] = host.Name
		configMap.Data = map[string]string{
			"config.yaml": buf.String(),
		}

		return ctrl.SetControllerReference(host, configMap, r.Scheme)
	}); err != nil {
		kdexv1alpha1.SetConditions(
			&host.Status.Conditions,
			kdexv1alpha1.ConditionStatuses{
				Degraded:    metav1.ConditionTrue,
				Progressing: metav1.ConditionFalse,
				Ready:       metav1.ConditionFalse,
			},
			kdexv1alpha1.ConditionReasonReconcileError,
			err.Error(),
		)
		if err := r.Status().Update(ctx, host); err != nil {
			return err
		}
		return err
	}

	return nil
}

func (r *KDexHostReconciler) createOrUpdateHostControllerResource(
	ctx context.Context,
	host *kdexv1alpha1.KDexHost,
) error {
	hostController := &kdexv1alpha1.KDexHostController{
		ObjectMeta: metav1.ObjectMeta{
			Name:      host.Name,
			Namespace: host.Namespace,
		},
	}

	if _, err := ctrl.CreateOrUpdate(ctx, r.Client, hostController, func() error {
		hostController.Spec = kdexv1alpha1.KDexHostControllerSpec{
			HostRef: corev1.LocalObjectReference{
				Name: host.Name,
			},
		}
		return ctrl.SetControllerReference(host, hostController, r.Scheme)
	}); err != nil {
		kdexv1alpha1.SetConditions(
			&host.Status.Conditions,
			kdexv1alpha1.ConditionStatuses{
				Degraded:    metav1.ConditionTrue,
				Progressing: metav1.ConditionFalse,
				Ready:       metav1.ConditionFalse,
			},
			kdexv1alpha1.ConditionReasonReconcileError,
			err.Error(),
		)
		if err := r.Status().Update(ctx, host); err != nil {
			return err
		}
		return err
	}

	return nil
}

func (r *KDexHostReconciler) createOrUpdateDeployment(
	ctx context.Context,
	host *kdexv1alpha1.KDexHost,
) error {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      host.Name,
			Namespace: host.Namespace,
		},
	}

	if _, err := ctrl.CreateOrUpdate(
		ctx,
		r.Client,
		deployment,
		func() error {
			deployment.Annotations = host.Annotations
			deployment.Labels = host.Labels
			if deployment.Labels == nil {
				deployment.Labels = make(map[string]string)
			}
			deployment.Labels["app.kubernetes.io/name"] = "kdex-web"
			deployment.Labels["kdex.dev/focus-host"] = host.Name
			deployment.Spec = r.Configuration.FocusController.Deployment
			deployment.Spec.Selector.MatchLabels = make(map[string]string)
			deployment.Spec.Selector.MatchLabels["app.kubernetes.io/name"] = "kdex-web"
			deployment.Spec.Selector.MatchLabels["kdex.dev/focus-host"] = host.Name
			deployment.Spec.Template.Labels = make(map[string]string)
			deployment.Spec.Template.Labels["app.kubernetes.io/name"] = "kdex-web"
			deployment.Spec.Template.Labels["kdex.dev/focus-host"] = host.Name

			foundFocalHost := false
			foundServiceName := false
			for idx, value := range deployment.Spec.Template.Spec.Containers[0].Args {
				if strings.Contains(value, "--focal-host") {
					deployment.Spec.Template.Spec.Containers[0].Args[idx] = "--focal-host=" + host.Name
					foundFocalHost = true
				}
				if strings.Contains(value, "--service-name") {
					deployment.Spec.Template.Spec.Containers[0].Args[idx] = "--service-name=" + host.Name
					foundServiceName = true
				}
			}
			for idx, value := range deployment.Spec.Template.Spec.Containers[0].Command {
				if strings.Contains(value, "--focal-host") {
					deployment.Spec.Template.Spec.Containers[0].Command[idx] = "--focal-host=" + host.Name
					foundFocalHost = true
				}
				if strings.Contains(value, "--service-name") {
					deployment.Spec.Template.Spec.Containers[0].Command[idx] = "--service-name=" + host.Name
					foundServiceName = true
				}
			}
			if !foundFocalHost {
				deployment.Spec.Template.Spec.Containers[0].Args = append(deployment.Spec.Template.Spec.Containers[0].Args, "--focal-host="+host.Name)
			}
			if !foundServiceName {
				deployment.Spec.Template.Spec.Containers[0].Args = append(deployment.Spec.Template.Spec.Containers[0].Args, "--service-name="+host.Name)
			}

			deployment.Spec.Template.Spec.Containers[0].Name = host.Name
			deployment.Spec.Template.Spec.Containers[0].Ports[0].Name = host.Name
			deployment.Spec.Template.Spec.ServiceAccountName = host.Name

			for idx, volume := range deployment.Spec.Template.Spec.Volumes {
				if volume.Name == "config" {
					deployment.Spec.Template.Spec.Volumes[idx].VolumeSource.ConfigMap.Name = host.Name
				}
			}

			return ctrl.SetControllerReference(host, deployment, r.Scheme)
		},
	); err != nil {
		kdexv1alpha1.SetConditions(
			&host.Status.Conditions,
			kdexv1alpha1.ConditionStatuses{
				Degraded:    metav1.ConditionTrue,
				Progressing: metav1.ConditionFalse,
				Ready:       metav1.ConditionFalse,
			},
			kdexv1alpha1.ConditionReasonReconcileError,
			err.Error(),
		)

		if err := r.Status().Update(ctx, host); err != nil {
			return err
		}

		return err
	}

	return nil
}

func (r *KDexHostReconciler) createOrUpdateRole(ctx context.Context, host *kdexv1alpha1.KDexHost) error {
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      host.Name,
			Namespace: host.Namespace,
		},
	}

	if _, err := ctrl.CreateOrUpdate(
		ctx,
		r.Client,
		role,
		func() error {
			role.Annotations = host.Annotations
			role.Labels = host.Labels
			if role.Labels == nil {
				role.Labels = make(map[string]string)
			}
			role.Labels["app.kubernetes.io/name"] = "kdex-web"
			role.Labels["kdex.dev/focus-host"] = host.Name

			role.Rules = []rbacv1.PolicyRule{}

			for _, rule := range r.Configuration.FocusController.RolePolicyRules {
				role.Rules = append(role.Rules, rbacv1.PolicyRule{
					APIGroups: rule.APIGroups,
					Resources: rule.Resources,
					Verbs:     rule.Verbs,
				})
			}

			return ctrl.SetControllerReference(host, role, r.Scheme)
		},
	); err != nil {
		kdexv1alpha1.SetConditions(
			&host.Status.Conditions,
			kdexv1alpha1.ConditionStatuses{
				Degraded:    metav1.ConditionTrue,
				Progressing: metav1.ConditionFalse,
				Ready:       metav1.ConditionFalse,
			},
			kdexv1alpha1.ConditionReasonReconcileError,
			err.Error(),
		)

		if err := r.Status().Update(ctx, host); err != nil {
			return err
		}

		return err
	}

	return nil
}

func (r *KDexHostReconciler) createOrUpdateRoleBinding(ctx context.Context, host *kdexv1alpha1.KDexHost) error {
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      host.Name,
			Namespace: host.Namespace,
		},
	}

	if _, err := ctrl.CreateOrUpdate(
		ctx,
		r.Client,
		roleBinding,
		func() error {
			roleBinding.Annotations = host.Annotations
			roleBinding.Labels = host.Labels
			if roleBinding.Labels == nil {
				roleBinding.Labels = make(map[string]string)
			}
			roleBinding.Labels["app.kubernetes.io/name"] = "kdex-web"
			roleBinding.Labels["kdex.dev/focus-host"] = host.Name
			roleBinding.RoleRef = rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "Role",
				Name:     host.Name,
			}
			roleBinding.Subjects = []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      host.Name,
					Namespace: host.Namespace,
				},
			}

			return ctrl.SetControllerReference(host, roleBinding, r.Scheme)
		},
	); err != nil {
		kdexv1alpha1.SetConditions(
			&host.Status.Conditions,
			kdexv1alpha1.ConditionStatuses{
				Degraded:    metav1.ConditionTrue,
				Progressing: metav1.ConditionFalse,
				Ready:       metav1.ConditionFalse,
			},
			kdexv1alpha1.ConditionReasonReconcileError,
			err.Error(),
		)

		if err := r.Status().Update(ctx, host); err != nil {
			return err
		}

		return err
	}

	return nil
}

func (r *KDexHostReconciler) createOrUpdateService(
	ctx context.Context,
	host *kdexv1alpha1.KDexHost,
) error {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      host.Name,
			Namespace: host.Namespace,
		},
	}

	if _, err := ctrl.CreateOrUpdate(
		ctx,
		r.Client,
		service,
		func() error {

			service.Annotations = host.Annotations
			service.Labels = host.Labels
			if service.Labels == nil {
				service.Labels = make(map[string]string)
			}
			service.Labels["app.kubernetes.io/name"] = "kdex-web"
			service.Labels["kdex.dev/focus-host"] = host.Name
			service.Spec = r.Configuration.FocusController.Service
			service.Spec.Selector["app.kubernetes.io/name"] = "kdex-web"
			service.Spec.Selector["kdex.dev/focus-host"] = host.Name

			for idx, value := range service.Spec.Ports {
				if value.Name == "webserver" {
					service.Spec.Ports[idx].Name = host.Name
					service.Spec.Ports[idx].TargetPort.StrVal = host.Name
				}
			}

			return ctrl.SetControllerReference(host, service, r.Scheme)
		},
	); err != nil {
		kdexv1alpha1.SetConditions(
			&host.Status.Conditions,
			kdexv1alpha1.ConditionStatuses{
				Degraded:    metav1.ConditionTrue,
				Progressing: metav1.ConditionFalse,
				Ready:       metav1.ConditionFalse,
			},
			kdexv1alpha1.ConditionReasonReconcileError,
			err.Error(),
		)

		if err := r.Status().Update(ctx, host); err != nil {
			return err
		}

		return err
	}

	return nil
}

func (r *KDexHostReconciler) createOrUpdateServiceAccount(ctx context.Context, host *kdexv1alpha1.KDexHost) error {
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      host.Name,
			Namespace: host.Namespace,
		},
	}

	if _, err := ctrl.CreateOrUpdate(
		ctx,
		r.Client,
		serviceAccount,
		func() error {

			serviceAccount.Annotations = host.Annotations
			serviceAccount.Labels = host.Labels
			if serviceAccount.Labels == nil {
				serviceAccount.Labels = make(map[string]string)
			}
			serviceAccount.Labels["app.kubernetes.io/name"] = "kdex-web"
			serviceAccount.Labels["kdex.dev/focus-host"] = host.Name

			return ctrl.SetControllerReference(host, serviceAccount, r.Scheme)
		},
	); err != nil {
		kdexv1alpha1.SetConditions(
			&host.Status.Conditions,
			kdexv1alpha1.ConditionStatuses{
				Degraded:    metav1.ConditionTrue,
				Progressing: metav1.ConditionFalse,
				Ready:       metav1.ConditionFalse,
			},
			kdexv1alpha1.ConditionReasonReconcileError,
			err.Error(),
		)

		if err := r.Status().Update(ctx, host); err != nil {
			return err
		}

		return err
	}

	return nil
}
