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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	hpcv1 "github.com/Rosalita/distributed-compute-operator/api/v1"
)

var _ = Describe("DistributedJob Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		distributedjob := &hpcv1.DistributedJob{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind DistributedJob")
			err := k8sClient.Get(ctx, typeNamespacedName, distributedjob)
			if err != nil && errors.IsNotFound(err) {
				resource := &hpcv1.DistributedJob{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: hpcv1.DistributedJobSpec{
						WorkerReplicas: 1,
						Image:          "test-image:latest",
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &hpcv1.DistributedJob{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance DistributedJob")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &DistributedJobReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			req := reconcile.Request{
				NamespacedName: typeNamespacedName,
			}

			// 1. First Reconcile creates the Headless Service
			res, err := controllerReconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(res.RequeueAfter).To(Equal(time.Second))
			svc := &corev1.Service{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: resourceName + "-svc", Namespace: "default"}, svc)).To(Succeed())

			// 2. Second Reconcile creates the Leader Pod
			res, err = controllerReconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(res.RequeueAfter).To(Equal(time.Second))
			leaderPod := &corev1.Pod{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: resourceName + "-leader", Namespace: "default"}, leaderPod)).To(Succeed())

			// 3. Third Reconcile creates the Worker Pod (WorkerReplicas is 1)
			res, err = controllerReconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(res.RequeueAfter).To(Equal(time.Second))
			workerPod := &corev1.Pod{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: resourceName + "-worker-0", Namespace: "default"}, workerPod)).To(Succeed())

			// 4. Fourth Reconcile updates the Status
			res, err = controllerReconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(res.RequeueAfter).To(BeZero()) // No more resources to create!
			updatedJob := &hpcv1.DistributedJob{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, updatedJob)).To(Succeed())
			Expect(updatedJob.Status.Phase).To(Equal("Pending"))
			Expect(updatedJob.Status.ActiveWorkers).To(Equal(int32(0)))
		})
	})
})
