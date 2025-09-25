package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type MicroFrontEndCommonReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *MicroFrontEndCommonReconciler) GetNavigation(
	ctx context.Context,
	log logr.Logger,
	navigationRef corev1.LocalObjectReference,
	object *ClientObjectWithConditions,
) (*kdexv1alpha1.MicroFrontEndPageNavigation, ctrl.Result, error) {
	var navigation kdexv1alpha1.MicroFrontEndPageNavigation
	navigationName := types.NamespacedName{
		Name:      navigationRef.Name,
		Namespace: object.GetNamespace(),
	}
	if err := r.Get(ctx, navigationName, &navigation); err != nil {
		if errors.IsNotFound(err) {
			log.Error(err, "referenced MicroFrontEndPageNavigation not found", "name", navigationRef.Name)
			apimeta.SetStatusCondition(
				object.Conditions,
				*kdexv1alpha1.NewCondition(
					kdexv1alpha1.ConditionTypeReady,
					metav1.ConditionFalse,
					kdexv1alpha1.ConditionReasonReconcileError,
					fmt.Sprintf("referenced MicroFrontEndPageNavigation %s not found", navigationRef.Name),
				),
			)
			if err := r.Status().Update(ctx, object.Object); err != nil {
				return nil, ctrl.Result{}, err
			}

			return nil, ctrl.Result{RequeueAfter: 15 * time.Second}, nil
		}

		log.Error(err, "unable to fetch MicroFrontEndPageNavigation", "name", navigationRef.Name)
		return nil, ctrl.Result{}, err
	}

	return &navigation, ctrl.Result{}, nil
}
