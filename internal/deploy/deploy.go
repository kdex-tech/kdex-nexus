package deploy

import (
	"context"
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	"kdex.dev/crds/configuration"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Deployer struct {
	Client         client.Client
	Scheme         *runtime.Scheme
	Configuration  configuration.NexusConfiguration
	ServiceAccount string
}

func (d *Deployer) GetOrCreateDeployJob(ctx context.Context, function *kdexv1alpha1.KDexFunction) (*batchv1.Job, error) {
	// Create Job name
	jobName := fmt.Sprintf("%s-deployer-%d", function.Name, function.Generation)

	job := &batchv1.Job{}
	err := d.Client.Get(ctx, client.ObjectKey{Namespace: function.Namespace, Name: jobName}, job)
	if err == nil {
		return job, nil
	}
	if !errors.IsNotFound(err) {
		return nil, err
	}

	return nil, nil
}
