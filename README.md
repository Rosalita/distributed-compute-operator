# distributed-compute-operator

# Introduction (What is this?) 
This is a custom Kubernetes Operator for orchestrating distributed workloads using leader-worker topologies and Headless Services.

This project serves as a demonstration of advanced Kubernetes internals, including the Operator pattern, declarative API design in Go, and stateful networking primitives.

# Usage (How do I use it?)
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

# Architecture (How does it work?)

Inside every Kubernetes operator is a Reconciler. The Reconciler is an endless loop (often called a control loop) that constantly asks three questions:
1. What is the Desired state?
2. What is the Actual state?
3. What needs to happen to make the Actual state match the desired state?

For example, if the manifest says there needs to be 3 worker replicas, but only 1 is running, the reconcilers job is to create 2 more. If something is deleted from the manifest, the reconcilers job is to clean up those pods.

## Operator

The Operator listens for events related to the `DistributedJob` Custom Resource Definition (CRD). When a `DistributedJob` is created, updated, or deleted, the controller's Reconcile loop is triggered to ensure the cluster's actual state matches the desired state.

## Headless Service

Every time a `DistributedJob` is created, the first thing that is created is a Headless Service. 

Normally a Kubernetes service acts as a load balancer. If you send traffic to it, it randomly forwards that traffic to one of the pods behind it. But in tightly-coupled distributed computing (like MPI (Message Passing Interface - a protocol used to link multiple computers together to solve heavy computational problems) - or deep learning workloads (a specialised AI technique to learn complex patterns from data)), the workers don't want load balancing. Worker #1 needs to know exactly how to talk directly to worker #2. 

By creating a "Headless" service (which means `ClusterIP:None` is set) Kubernetes skips the load balancing and instead creates a predictable DNS record for every single pod. This allows worker pods to discover and talk directly to one another.

## DistributedJob Controller

The distributed controller fetches the `DistributedJob` from the cluster. Then checks if a headless service already exists. If no headless service exists, it creates one and links it to the `DistributedJob` using `SetControllerReference`. This means that if the user deletes the job, Kubernetes automatically cleans up the headless service.

Because the controller is going to manage Kubernetes services, we have to tell Kubebuilder to generate the permissions (RBAC - role-based access control) to allow our operator to do that. The `Owns()` function in `SetupWithManager` tells the controller to "watch" services that are owned by a `DistributedJob` if someone accidentally deletes the Headless service, this ensures the controller loop will immediately recreate it.

For every valid `DistributedJob` resource, this operator automatically provisions a tightly coupled topology:
1. A **Leader Pod** responsible for coordinating the distributed workload.
2. A configurable number of **Worker Pods** for parallel execution.
3. A **Headless Service** to provide predictable DNS records (e.g., `worker-0`, `worker-1`) to enable direct pod-to-pod communication (bypassing load balancing), which is essential for MPI (Message Passing Interface) and other tightly coupled HPC workloads.

The controller uses `OwnerReferences` to ensure that all generated child resources (Pods and Services) are automatically garbage-collected by Kubernetes if the parent `DistributedJob` is deleted.

# Development Guide (How was it built?)
Kubebuilder was used to scaffold this project.

## Running Kubebuilder on Windows 
Since native Windows support was dropped in recent versions of kubebuilder, and I sometimes do development work on a Windows gaming PC, the easiest way to work around  lack of Windows support was to run kubebuilder in temporary Linux container with Docker Desktop.

To start a temporary Go Linux container and mount the project directory:
```bash
docker run --rm -it -v "${PWD}:/workspace" -w /workspace golang:latest bash
```
This lets kubebuilder to run in the container and output to the local workspace.

The project was initialised with
```bash
./kubebuilder init --domain rosalita.github.io --repo github.com/Rosalita/distributed-compute-operator
```

A `DistributedJob` CRD (Custom Resource Definition) was also created. Using the following command, kubebuilder automatically scaffolds a Go struct to represent the Custom Resource's Spec (the desired state). By default, it inserts a dummy field called Foo just to give you an example of what a field looks like and how to use JSON tags.
```bash
./kubebuilder create api --group hpc --version v1 --kind DistributedJob
```

The schema in `api/v1/distributedjob_types.go` was updated:
- **Spec:** Added `WorkerReplicas`, `Image`, and an optional `Command` list.
- **Status:** Added `Phase` and `ActiveWorkers` to track the job's progress.

Whenever the API schema or Kubebuilder markers are modified, two important `make` commands must be run inside the container to synchronize the project:

- **`make generate`**: Updates the autogenerated Go code. Specifically, it parses the custom resource structs and generates the `DeepCopy` methods in `zz_generated.deepcopy.go`. Kubernetes requires all API objects to implement these methods so they can be safely duplicated in memory.
- **`make manifests`**: Generates the actual Kubernetes YAML files. It reads the Go structs and the special `// +kubebuilder:...` marker comments to automatically build the CustomResourceDefinition (CRD) YAML manifests in `config/crd/bases/`, as well as any RBAC roles and Webhook configurations.

## Enhancing Custom Resource Definition (CRD) with markers
When `kubectl get pods` is run, a nice table with columns like `NAME`, `READY`, `STATUS`, `RESTARTS` and `AGE` is displayed. Kubebuilder can use special comments (called "markers") to tell Kubernetes to display additional useful fields in the table.

When a marker is defined, the default table layout is thrown away and only the `NAME` column is kept. If markers are added for `PHASE`, `WORKERS`, `ACTIVE` `AGE` This means when `kubectl get distributedjobs` is run it will display something like

```
NAME      PHASE     WORKERS   ACTIVE   AGE
my-job    Running   3         3        2m15s
```
After adding custom markers, `make manifests` needs to be run.

# License

Copyright 2026. Licensed under the MIT License. See LICENSE in the project root for license information.
