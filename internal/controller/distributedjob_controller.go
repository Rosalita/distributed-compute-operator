/*
Copyright 2026.

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

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	hpcv1 "github.com/Rosalita/distributed-compute-operator/api/v1"
)

// DistributedJobReconciler reconciles a DistributedJob object
type DistributedJobReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=hpc.rosalita.github.io,resources=distributedjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=hpc.rosalita.github.io,resources=distributedjobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=hpc.rosalita.github.io,resources=distributedjobs/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *DistributedJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// 1. Fetch the DistributedJob instance
	var job hpcv1.DistributedJob
	if err := r.Get(ctx, req.NamespacedName, &job); err != nil {
		if apierrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Return and don't requeue
			log.Info("DistributedJob resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get DistributedJob")
		return ctrl.Result{}, err
	}

	// 2. Check if the Headless Service already exists, if not create it
	svcName := job.Name + "-svc"
	var svc corev1.Service
	if err := r.Get(ctx, types.NamespacedName{Name: svcName, Namespace: job.Namespace}, &svc); err != nil {
		if apierrors.IsNotFound(err) {
			// Define a new Headless Service
			newSvc, err := r.serviceForDistributedJob(&job)
			if err != nil {
				log.Error(err, "Failed to define new Headless Service for DistributedJob")
				return ctrl.Result{}, err
			}
			log.Info("Creating a new Headless Service", "Service.Namespace", newSvc.Namespace, "Service.Name", newSvc.Name)
			if err := r.Create(ctx, newSvc); err != nil {
				log.Error(err, "Failed to create new Headless Service", "Service.Namespace", newSvc.Namespace, "Service.Name", newSvc.Name)
				return ctrl.Result{}, err
			}
			// Service created successfully - return and requeue
			return ctrl.Result{Requeue: true}, nil
		}
		log.Error(err, "Failed to get Headless Service")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DistributedJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&hpcv1.DistributedJob{}).
		Owns(&corev1.Service{}).
		Named("distributedjob").
		Complete(r)
}

// serviceForDistributedJob returns a headless service for the DistributedJob
func (r *DistributedJobReconciler) serviceForDistributedJob(job *hpcv1.DistributedJob) (*corev1.Service, error) {
	labelSelector := map[string]string{"app": "distributedjob", "job_name": job.Name}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      job.Name + "-svc",
			Namespace: job.Namespace,
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "None", // This makes it a Headless Service!
			Selector:  labelSelector,
		},
	}
	// Set DistributedJob instance as the owner and controller
	if err := ctrl.SetControllerReference(job, svc, r.Scheme); err != nil {
		return nil, err
	}
	return svc, nil
}
