package controller

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ClientObjectWithConditions struct {
	client.Object
	Conditions *[]metav1.Condition
}
