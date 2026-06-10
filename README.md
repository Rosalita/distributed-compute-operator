# distributed-compute-operator
A custom Kubernetes Operator for orchestrating distributed workloads.

A Kubernetes Operator for orchestrating distributed workloads using leader-worker topologies and Headless Services.

This project serves as a demonstration of advanced Kubernetes internals, including the Operator pattern, declarative API design in Go, and stateful networking primitives.

## Architecture

The Operator listens for events related to the `DistributedJob` Custom Resource Definition (CRD). When a `DistributedJob` is created, updated, or deleted, the controller's Reconcile loop is triggered to ensure the cluster's actual state matches the desired state.

For every valid `DistributedJob` resource, this operator automatically provisions a tightly coupled topology:
1. A **Leader Pod** responsible for coordinating the distributed workload.
2. A configurable number of **Worker Pods** for parallel execution.
3. A **Headless Service** to provide predictable DNS records (e.g., `worker-0`, `worker-1`) to enable direct pod-to-pod communication (bypassing load balancing), which is essential for MPI (Message Passing Interface) and other tightly coupled HPC workloads.

The controller uses `OwnerReferences` to ensure that all generated child resources (Pods and Services) are automatically garbage-collected by Kubernetes if the parent `DistributedJob` is deleted.

## Prerequisites

- Go v1.20+
- Docker
- kubectl
- A local Kubernetes cluster (Docker Desktop, kind, or minikube)
- Kubebuilder v3+

## Getting Started

### 1. Clone the repository
```bash
git clone https://github.com/Rosalita/distributed-compute-operator.git
cd distributed-compute-operator
```

### 2. Install the CRDs into your cluster
Make sure your local Kubernetes cluster is running, then apply the Custom Resource Definitions:
```bash
make install
```

### 3. Run the controller locally
You can run the controller directly on your host machine (outside the cluster) for easy debugging and development:
```bash
make run
```

### 4. Create a DistributedJob instance
In a new terminal window, apply the sample custom resource:
```bash
kubectl apply -f config/samples/hpc_v1_distributedjob.yaml
```

You can then observe the Operator creating the associated Pods and Headless Service:
```bash
kubectl get distributedjobs
kubectl get pods
kubectl get services
```

## Cleaning Up

To remove the DistributedJob instance and let Kubernetes garbage collection remove the associated resources:
```bash
kubectl delete -f config/samples/hpc_v1_distributedjob.yaml
```

To uninstall the CRDs from your cluster:
```bash
make uninstall
```

## License

Copyright 2026. Licensed under the MIT License. See LICENSE in the project root for license information.
