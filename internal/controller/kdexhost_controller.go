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
	"os"
	"strings"
	"sync"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	"kdex.dev/crds/configuration"
	nexuswebhook "kdex.dev/nexus/internal/webhook"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	hostFinalizerName = "kdex.dev/kdex-nexus-host-finalizer"
	kdexWeb           = "kdex-web"
)

// KDexHostReconciler reconciles a KDexHost object
type KDexHostReconciler struct {
	client.Client
	Configuration configuration.NexusConfiguration
	RequeueDelay  time.Duration
	Scheme        *runtime.Scheme

	mu                    sync.RWMutex
	memoizedConfiguration string
	memoizedDeployment    *appsv1.DeploymentSpec
	memoizedService       *corev1.ServiceSpec
}

// +kubebuilder:rbac:groups=apps,resources=deployments,                             verbs=get;list;watch;create;update;patch;delete

// +kubebuilder:rbac:groups=batch,resources=jobs,                                   verbs=get;list;watch;create;update;patch;delete

// +kubebuilder:rbac:groups=core,resources=configmaps,                              verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,                                    verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=secrets,                                 verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=serviceaccounts,                         verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,                                verbs=get;list;watch;create;update;patch;delete

// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=httproutes,         verbs=get;list;watch;create;update;patch;delete

// +kubebuilder:rbac:groups=kdex.dev,resources=kdexhosts,                           verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexhosts/status,                    verbs=get;update;patch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexhosts/finalizers,                verbs=update
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexhostcontrollers,                 verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexhostcontrollers/status,          verbs=get;update;patch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexhostcontrollers/finalizers,      verbs=update
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexhostpackagereferences,           verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexhostpackagereferences/status,    verbs=get;update;patch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexhostpackagereferences/finalizers,verbs=update
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexpagebindings,                    verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexpagebindings/status,             verbs=get;update;patch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexpagebindings/finalizers,         verbs=update
// +kubebuilder:rbac:groups=kdex.dev,resources=kdextranslations,                    verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kdex.dev,resources=kdextranslations/status,             verbs=get;update;patch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdextranslations/finalizers,         verbs=update

// +kubebuilder:rbac:groups=kdex.dev,resources=kdexapps,                            verbs=get;list;watch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexclusterapps,                     verbs=get;list;watch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexpagefooters,                     verbs=get;list;watch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexclusterpagefooters,              verbs=get;list;watch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexpageheaders,                     verbs=get;list;watch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexclusterpageheaders,              verbs=get;list;watch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexpagenavigations,                 verbs=get;list;watch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexclusterpagenavigations,          verbs=get;list;watch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexscriptlibraries,                 verbs=get;list;watch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexclusterscriptlibraries,          verbs=get;list;watch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexthemes,                          verbs=get;list;watch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexclusterthemes,                   verbs=get;list;watch

// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,                  verbs=get;list;watch;create;update;patch;delete

// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles,       verbs=get;list;watch
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterrolebindings,verbs=get;list;watch;create;update;patch;delete

func (r *KDexHostReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	log := logf.FromContext(ctx)

	var host kdexv1alpha1.KDexHost
	if err := r.Get(ctx, req.NamespacedName, &host); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if host.Status.Attributes == nil {
		host.Status.Attributes = make(map[string]string)
	}

	// Defer status update
	defer func() {
		host.Status.ObservedGeneration = host.Generation
		if updateErr := r.Status().Update(ctx, &host); updateErr != nil {
			err = updateErr
			res = ctrl.Result{}
		}

		log.V(2).Info("status", "status", host.Status, "err", err, "res", res)
	}()

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
			hostController := &kdexv1alpha1.KDexHostController{}
			err := r.Get(ctx, req.NamespacedName, hostController)
			if err == nil {
				if hostController.DeletionTimestamp.IsZero() {
					if err := r.Delete(ctx, hostController); err != nil {
						return ctrl.Result{}, err
					}
				}
				// KDexHostController still exists. We wait.
				return ctrl.Result{Requeue: true}, nil
			}
			if !errors.IsNotFound(err) {
				return ctrl.Result{}, err
			}

			deployment := &appsv1.Deployment{}
			err = r.Get(ctx, req.NamespacedName, deployment)
			if err == nil {
				if deployment.DeletionTimestamp.IsZero() {
					if err := r.Delete(ctx, deployment); err != nil {
						return ctrl.Result{}, err
					}
				}
				// Deployment still exists. We wait.
				return ctrl.Result{Requeue: true}, nil
			}
			if !errors.IsNotFound(err) {
				return ctrl.Result{}, err
			}

			// Deployment is gone. Clean up RBAC finalizers.
			if err := r.cleanupRbacFinalizers(ctx, &host); err != nil {
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

	themeObj, shouldReturn, r1, err := ResolveKDexObjectReference(ctx, r.Client, &host, &host.Status.Conditions, host.Spec.ThemeRef, r.RequeueDelay)
	if shouldReturn {
		return r1, err
	}

	if themeObj != nil {
		host.Status.Attributes["theme.generation"] = fmt.Sprintf("%d", themeObj.GetGeneration())
	}

	scriptLibraryObj, shouldReturn, r1, err := ResolveKDexObjectReference(ctx, r.Client, &host, &host.Status.Conditions, host.Spec.ScriptLibraryRef, r.RequeueDelay)
	if shouldReturn {
		return r1, err
	}

	if scriptLibraryObj != nil {
		host.Status.Attributes["scriptLibrary.generation"] = fmt.Sprintf("%d", scriptLibraryObj.GetGeneration())
	}

	return ctrl.Result{}, r.innerReconcile(ctx, &host)
}

// SetupWithManager sets up the controller with the Manager.
func (r *KDexHostReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if os.Getenv("ENABLE_WEBHOOKS") != FALSE {
		err := ctrl.NewWebhookManagedBy(mgr).
			For(&kdexv1alpha1.KDexHost{}).
			WithDefaulter(&nexuswebhook.KDexHostDefaulter{}).
			WithValidator(&nexuswebhook.KDexHostValidator{}).
			Complete()

		if err != nil {
			return err
		}
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&kdexv1alpha1.KDexHost{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&kdexv1alpha1.KDexHostController{}).
		Owns(&rbacv1.ClusterRoleBinding{}).
		Watches(
			&kdexv1alpha1.KDexScriptLibrary{},
			MakeHandlerByReferencePath(r.Client, r.Scheme, &kdexv1alpha1.KDexHost{}, &kdexv1alpha1.KDexHostList{}, "{.Spec.ScriptLibraryRef}")).
		Watches(
			&kdexv1alpha1.KDexClusterScriptLibrary{},
			MakeHandlerByReferencePath(r.Client, r.Scheme, &kdexv1alpha1.KDexHost{}, &kdexv1alpha1.KDexHostList{}, "{.Spec.ScriptLibraryRef}")).
		Watches(
			&kdexv1alpha1.KDexTheme{},
			MakeHandlerByReferencePath(r.Client, r.Scheme, &kdexv1alpha1.KDexHost{}, &kdexv1alpha1.KDexHostList{}, "{.Spec.ThemeRef}")).
		Watches(
			&kdexv1alpha1.KDexClusterTheme{},
			MakeHandlerByReferencePath(r.Client, r.Scheme, &kdexv1alpha1.KDexHost{}, &kdexv1alpha1.KDexHostList{}, "{.Spec.ThemeRef}")).
		WithOptions(controller.TypedOptions[reconcile.Request]{
			LogConstructor: LogConstructor("kdexhost", mgr),
		}).
		Named("kdexhost").
		Complete(r)
}

func (r *KDexHostReconciler) cleanupRbacFinalizers(ctx context.Context, host *kdexv1alpha1.KDexHost) error {
	// ClusterRoleBinding
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{}
	clusterRoleBinding.Name = fmt.Sprintf("%s-%s", host.Name, host.Namespace)
	if err := r.Get(ctx, types.NamespacedName{Name: clusterRoleBinding.Name}, clusterRoleBinding); err == nil {
		if controllerutil.RemoveFinalizer(clusterRoleBinding, hostFinalizerName) {
			if err := r.Update(ctx, clusterRoleBinding); err != nil {
				return err
			}
		}
	} else if !errors.IsNotFound(err) {
		return err
	}

	// ServiceAccount
	serviceAccount := &corev1.ServiceAccount{}
	if err := r.Get(ctx, types.NamespacedName{Name: host.Name, Namespace: host.Namespace}, serviceAccount); err == nil {
		if controllerutil.RemoveFinalizer(serviceAccount, hostFinalizerName) {
			if err := r.Update(ctx, serviceAccount); err != nil {
				return err
			}
		}
	} else if !errors.IsNotFound(err) {
		return err
	}

	return nil
}

func (r *KDexHostReconciler) getMemoizedConfiguration() (string, error) {
	r.mu.RLock()

	if r.memoizedConfiguration != "" {
		r.mu.RUnlock()
		return r.memoizedConfiguration, nil
	}

	r.mu.RUnlock()
	r.mu.Lock()
	defer r.mu.Unlock()

	codecs := serializer.NewCodecFactory(r.Scheme)

	info, ok := runtime.SerializerInfoForMediaType(codecs.SupportedMediaTypes(), "application/yaml")
	if !ok {
		return "", fmt.Errorf("no YAML serializer found")
	}

	encoder := codecs.EncoderForVersion(info.Serializer, configuration.GroupVersion)

	var buf bytes.Buffer
	if err := encoder.Encode(&r.Configuration, &buf); err != nil {
		return "", fmt.Errorf("failed to encode object to YAML: %w", err)
	}

	r.memoizedConfiguration = buf.String()

	return r.memoizedConfiguration, nil
}

func (r *KDexHostReconciler) getMemoizedDeployment() *appsv1.DeploymentSpec {
	r.mu.RLock()

	if r.memoizedDeployment != nil {
		r.mu.RUnlock()
		return r.memoizedDeployment
	}

	r.mu.RUnlock()
	r.mu.Lock()
	defer r.mu.Unlock()

	r.memoizedDeployment = r.Configuration.HostController.Deployment.DeepCopy()

	return r.memoizedDeployment
}

func (r *KDexHostReconciler) getMemoizedService() *corev1.ServiceSpec {
	r.mu.RLock()

	if r.memoizedService != nil {
		r.mu.RUnlock()
		return r.memoizedService
	}

	r.mu.RUnlock()
	r.mu.Lock()
	defer r.mu.Unlock()

	r.memoizedService = r.Configuration.HostController.Service.DeepCopy()

	return r.memoizedService
}

func (r *KDexHostReconciler) innerReconcile(ctx context.Context, host *kdexv1alpha1.KDexHost) error {
	configMapOp, err := r.createOrUpdateConfigMap(ctx, host)
	if err != nil {
		return err
	}

	serviceAccountOp, err := r.createOrUpdateServiceAccount(ctx, host)
	if err != nil {
		return err
	}

	clusterRoleBindingOp, err := r.createOrUpdateClusterRoleBinding(ctx, host)
	if err != nil {
		return err
	}

	deploymentOp, err := r.createOrUpdateDeployment(ctx, host)
	if err != nil {
		return err
	}

	serviceOp, err := r.createOrUpdateService(ctx, host)
	if err != nil {
		return err
	}

	hostControllerOp, err := r.createOrUpdateHostControllerResource(ctx, host)
	if err != nil {
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

	log := logf.FromContext(ctx)

	log.V(1).Info(
		"reconciled",
		"configMapOp", configMapOp,
		"serviceAccountOp", serviceAccountOp,
		"clusterRoleBindingOp", clusterRoleBindingOp,
		"deploymentOp", deploymentOp,
		"serviceOp", serviceOp,
		"hostControllerOp", hostControllerOp,
	)

	return nil
}

func (r *KDexHostReconciler) createOrUpdateConfigMap(
	ctx context.Context,
	host *kdexv1alpha1.KDexHost,
) (controllerutil.OperationResult, error) {
	configString, err := r.getMemoizedConfiguration()
	if err != nil {
		return controllerutil.OperationResultNone, err
	}

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      host.Name,
			Namespace: host.Namespace,
		},
	}

	op, err := ctrl.CreateOrUpdate(ctx, r.Client, configMap, func() error {
		if configMap.CreationTimestamp.IsZero() {
			if configMap.Annotations == nil {
				configMap.Annotations = make(map[string]string)
			}
			for key, value := range host.Annotations {
				configMap.Annotations[key] = value
			}
			if configMap.Labels == nil {
				configMap.Labels = make(map[string]string)
			}
			for key, value := range host.Labels {
				configMap.Labels[key] = value
			}

			configMap.Labels["app.kubernetes.io/name"] = kdexWeb
			configMap.Labels["kdex.dev/instance"] = host.Name
		}

		configMap.Data = map[string]string{
			"config.yaml": configString,
		}

		return ctrl.SetControllerReference(host, configMap, r.Scheme)
	})

	if err != nil {
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
		return controllerutil.OperationResultNone, err
	}

	return op, nil
}

func (r *KDexHostReconciler) createOrUpdateHostControllerResource(
	ctx context.Context,
	host *kdexv1alpha1.KDexHost,
) (controllerutil.OperationResult, error) {
	hostController := &kdexv1alpha1.KDexHostController{
		ObjectMeta: metav1.ObjectMeta{
			Name:      host.Name,
			Namespace: host.Namespace,
		},
	}

	op, err := ctrl.CreateOrUpdate(ctx, r.Client, hostController, func() error {
		if hostController.CreationTimestamp.IsZero() {
			if hostController.Annotations == nil {
				hostController.Annotations = make(map[string]string)
			}
			for key, value := range host.Annotations {
				hostController.Annotations[key] = value
			}
			if hostController.Labels == nil {
				hostController.Labels = make(map[string]string)
			}
			for key, value := range host.Labels {
				hostController.Labels[key] = value
			}

			hostController.Labels["app.kubernetes.io/name"] = kdexWeb
			hostController.Labels["kdex.dev/instance"] = host.Name
		}

		hostController.Spec = host.Spec

		return ctrl.SetControllerReference(host, hostController, r.Scheme)
	})

	if err != nil {
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

		return controllerutil.OperationResultNone, err
	}

	return op, nil
}

func (r *KDexHostReconciler) createOrUpdateDeployment(
	ctx context.Context,
	host *kdexv1alpha1.KDexHost,
) (controllerutil.OperationResult, error) {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      host.Name,
			Namespace: host.Namespace,
		},
	}

	op, err := ctrl.CreateOrUpdate(
		ctx,
		r.Client,
		deployment,
		func() error {
			if deployment.CreationTimestamp.IsZero() {
				if deployment.Annotations == nil {
					deployment.Annotations = make(map[string]string)
				}
				for key, value := range host.Annotations {
					deployment.Annotations[key] = value
				}
				if deployment.Labels == nil {
					deployment.Labels = make(map[string]string)
				}
				for key, value := range host.Labels {
					deployment.Labels[key] = value
				}

				deployment.Labels["app.kubernetes.io/name"] = kdexWeb
				deployment.Labels["kdex.dev/instance"] = host.Name

				deployment.Spec = *r.getMemoizedDeployment().DeepCopy()

				if deployment.Spec.Selector == nil {
					deployment.Spec.Selector = &metav1.LabelSelector{
						MatchLabels: map[string]string{},
					}
				}
				deployment.Spec.Selector.MatchLabels["app.kubernetes.io/name"] = kdexWeb
				deployment.Spec.Selector.MatchLabels["kdex.dev/instance"] = host.Name

				if deployment.Spec.Template.Labels == nil {
					deployment.Spec.Template.Labels = make(map[string]string)
				}
				deployment.Spec.Template.Labels["app.kubernetes.io/name"] = kdexWeb
				deployment.Spec.Template.Labels["kdex.dev/instance"] = host.Name

				deployment.Spec.Template.Spec = *r.getMemoizedDeployment().Template.Spec.DeepCopy()
			}

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
			deployment.Spec.Template.Spec.ServiceAccountName = host.Name

			for idx, volume := range deployment.Spec.Template.Spec.Volumes {
				if volume.Name == "config" {
					deployment.Spec.Template.Spec.Volumes[idx].ConfigMap.Name = host.Name
				}
			}

			return ctrl.SetControllerReference(host, deployment, r.Scheme)
		},
	)

	if err != nil {
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

		return controllerutil.OperationResultNone, err
	}

	return op, nil
}

func (r *KDexHostReconciler) createOrUpdateClusterRoleBinding(
	ctx context.Context,
	host *kdexv1alpha1.KDexHost,
) (controllerutil.OperationResult, error) {
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-%s", host.Name, host.Namespace),
		},
	}

	op, err := ctrl.CreateOrUpdate(
		ctx,
		r.Client,
		clusterRoleBinding,
		func() error {
			if clusterRoleBinding.CreationTimestamp.IsZero() {
				if clusterRoleBinding.Annotations == nil {
					clusterRoleBinding.Annotations = make(map[string]string)
				}
				for key, value := range host.Annotations {
					clusterRoleBinding.Annotations[key] = value
				}
				if clusterRoleBinding.Labels == nil {
					clusterRoleBinding.Labels = make(map[string]string)
				}
				for key, value := range host.Labels {
					clusterRoleBinding.Labels[key] = value
				}

				clusterRoleBinding.Labels["app.kubernetes.io/name"] = kdexWeb
				clusterRoleBinding.Labels["kdex.dev/instance"] = host.Name
			}

			clusterRoleBinding.RoleRef = *r.Configuration.HostController.RoleRef.DeepCopy()
			clusterRoleBinding.Subjects = []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      host.Name,
					Namespace: host.Namespace,
				},
			}

			controllerutil.AddFinalizer(clusterRoleBinding, hostFinalizerName)
			return nil
		},
	)

	if err != nil {
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

		return controllerutil.OperationResultNone, err
	}

	return op, nil
}

func (r *KDexHostReconciler) createOrUpdateService(
	ctx context.Context,
	host *kdexv1alpha1.KDexHost,
) (controllerutil.OperationResult, error) {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      host.Name,
			Namespace: host.Namespace,
		},
	}

	op, err := ctrl.CreateOrUpdate(
		ctx,
		r.Client,
		service,
		func() error {
			if service.CreationTimestamp.IsZero() {
				if service.Annotations == nil {
					service.Annotations = make(map[string]string)
				}
				for key, value := range host.Annotations {
					service.Annotations[key] = value
				}
				if service.Labels == nil {
					service.Labels = make(map[string]string)
				}
				for key, value := range host.Labels {
					service.Labels[key] = value
				}

				service.Labels["app.kubernetes.io/name"] = kdexWeb
				service.Labels["kdex.dev/instance"] = host.Name

				service.Spec = *r.getMemoizedService().DeepCopy()

				if service.Spec.Selector == nil {
					service.Spec.Selector = make(map[string]string)
				}

				service.Spec.Selector["app.kubernetes.io/name"] = kdexWeb
				service.Spec.Selector["kdex.dev/instance"] = host.Name
			}

			return ctrl.SetControllerReference(host, service, r.Scheme)
		},
	)

	if err != nil {
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

		return controllerutil.OperationResultNone, err
	}

	return op, nil
}

func (r *KDexHostReconciler) createOrUpdateServiceAccount(
	ctx context.Context,
	host *kdexv1alpha1.KDexHost,
) (controllerutil.OperationResult, error) {
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      host.Name,
			Namespace: host.Namespace,
		},
	}

	op, err := ctrl.CreateOrUpdate(
		ctx,
		r.Client,
		serviceAccount,
		func() error {
			if serviceAccount.CreationTimestamp.IsZero() {
				if serviceAccount.Annotations == nil {
					serviceAccount.Annotations = make(map[string]string)
				}
				for key, value := range host.Annotations {
					serviceAccount.Annotations[key] = value
				}
				if serviceAccount.Labels == nil {
					serviceAccount.Labels = make(map[string]string)
				}
				for key, value := range host.Labels {
					serviceAccount.Labels[key] = value
				}

				serviceAccount.Labels["app.kubernetes.io/name"] = kdexWeb
				serviceAccount.Labels["kdex.dev/instance"] = host.Name
			}

			controllerutil.AddFinalizer(serviceAccount, hostFinalizerName)
			return ctrl.SetControllerReference(host, serviceAccount, r.Scheme)
		},
	)

	if err != nil {
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

		return controllerutil.OperationResultNone, err
	}

	return op, nil
}
