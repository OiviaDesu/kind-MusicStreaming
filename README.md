# MusicService Operator

<!--
Quick reading guide:
- For reconciliation flow, see internal/controller/musicservice_controller.go.
- For CRD fields, see api/v1/musicservice_types.go.
- For resource creation logic, see internal/builder/resource_builder.go.
- For autoscaling and HPA, see internal/reconciler/app.go.
-->

A Kubernetes operator for deploying and managing music streaming services with MariaDB master/replica databases. Built with operator-sdk and kubebuilder following cloud-native best practices.

## Description

The MusicService operator manages a complete music streaming platform on Kubernetes, featuring:

- **StatefulSet-based deployments** for reliable music streaming services
- **MariaDB master/replica architecture** with automatic replication setup
- **Auto-scaling capabilities** with HPA integration
- **Persistent storage** for music data and databases
- **Streaming configuration** with bitrate and connection limits
- **Replication management** with GTID-based automatic synchronization

The operator demonstrates Kubernetes operator patterns and cloud-native application design principles.

## Features

### Music Service Deployment
- Configurable replicas with StatefulSets
- Custom streaming bitrate (e.g., "320k", "192k")
- Maximum concurrent connections control
- Persistent volume claims for music storage
- Resource requests and limits settings
- Service exposure with custom ports

### Database Management
- **Master/Replica Architecture**: Deploy MariaDB with 1 master + N replicas
- **Automatic Replication**: GTID-based replication is configured on startup (controller-managed replication Secret)
- **Separate Services**: Headless service for master, ClusterIP for read replicas
- **Persistent Storage**: Each database instance gets its own PVC
- **Configurable**: Custom images, storage sizes, and passwords
- **Replica Autoscaling**: Optional HPA for read replicas



## Getting Started

### Prerequisites
- go version v1.22.0+
- docker version 17.03+
- kubectl version v1.11.3+
- Access to a Kubernetes v1.11.3+ cluster (or use Kind for local testing)

### Environment Setup

Before running the operator, set up your environment variables:

```sh
# Copy the example environment file
cp .env.example .env

# Edit .env with your actual values
# IMPORTANT: Never commit .env to git - it contains sensitive data!
nano .env
```

The `.env` file contains configuration for:
- Database passwords (master and replication)
- Operator namespace and webhook settings
- Metrics and health probe endpoints
- Logging configuration

### Quick Start with Kind

We provide convenient Make targets for local development:

```sh
# Create Kind cluster, build image, and deploy operator
make deploy-kind

# Apply sample MusicService
kubectl apply -f config/samples/musicservice_sample.yaml

# Watch your music service deployment
kubectl get musicservices -w
kubectl describe musicservice miku-stream

# Check all created resources
kubectl get statefulsets
kubectl get services
kubectl get pvc
kubectl get pods
```

### Sample MusicService

Here's an example with all features enabled:

```yaml
apiVersion: music.mixcorp.org/v1
kind: MusicService
metadata:
  name: miku-stream
  labels:
spec:
  replicas: 3
  image: nginx:alpine  # Replace with your music streaming app
  port: 8080
  
  storage:
    size: 50Gi
    updatePolicy: Recreate
  
  streaming:
    bitrate: "320k"
    maxConnections: 5000
  
  resources:
    requests:
      cpu: "500m"
      memory: "512Mi"
    limits:
      cpu: "1000m"
      memory: "1Gi"
  
  autoscaling:
    minReplicas: 2
    maxReplicas: 10
    targetCPUUtilizationPercentage: 70
    targetMemoryUtilizationPercentage: 80
  
  database:
    enabled: true
    replicas: 2  # Number of read replicas
    image: mariadb:10.11
    replication:
      enabled: true
      gtid: true
    storage:
      size: 20Gi
      updatePolicy: Recreate
    rootPassword: "miku-secret-pass"  # Use k8s secrets in production!
    autoscaling:
      minReplicas: 1
      maxReplicas: 5
      targetCPUUtilizationPercentage: 70
```

### Architecture

When you deploy a MusicService with database enabled, the operator creates:

**For the Music Service:**
- StatefulSet with N replicas
- Service (ClusterIP) exposing your configured port
- PersistentVolumeClaims for each pod

**For the Database:**
- `{name}-db-master` StatefulSet (1 replica) - Master database
- `{name}-db-replica` StatefulSet (N replicas) - Read replicas
- `{name}-db-master` Service (Headless) - Direct access to master
- `{name}-db-read` Service (ClusterIP) - Load-balanced read access
- PVCs for each database instance
- Init containers that auto-configure replication

### Status Monitoring

The operator maintains comprehensive status:

```yaml
status:
  observedGeneration: 2
  desiredReplicas: 3
  readyReplicas: 3
  phase: Available
  lastReconcileTime: "2026-02-02T10:30:00Z"
  conditions:
    - type: Available
      status: "True"
      reason: PodsReady
      message: "All replicas are ready"
  database:
    phase: Ready
    masterReady: true
    replicasReady: 2
    replicaEverCreated: true
    replicaDeletionDetected: false
    replicationReady: true
```

### To Deploy on the cluster
**Build and push your image to the location specified by `IMG`:**

```sh
make docker-build docker-push IMG=<some-registry>/musicservice:tag
```

**NOTE:** This image ought to be published in the personal registry you specified.
And it is required to have access to pull the image from the working environment.
Make sure you have the proper permission to the registry if the above commands don’t work.

**Install the CRDs into the cluster:**

```sh
make install
```

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy IMG=<some-registry>/musicservice:tag
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin
privileges or be logged in as admin.

**Create instances of your solution**
You can apply the samples (examples) from the config/samples:

```sh
kubectl apply -f config/samples/musicservice_sample.yaml

# Check status
kubectl get musicservices miku-stream -o yaml
kubectl get events --field-selector involvedObject.name=miku-stream
```

### To Uninstall
**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k config/samples/
```

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```

## Project Distribution

Following are the steps to build the installer and distribute this project to users.

1. Build the installer for the image built and published in the registry:

```sh
make build-installer IMG=<some-registry>/musicservice:tag
```

NOTE: The makefile target mentioned above generates an 'install.yaml'
file in the dist directory. This file contains all the resources built
with Kustomize, which are necessary to install this project without
its dependencies.

2. Using the installer

Users can just run kubectl apply -f <URL for YAML BUNDLE> to install the project, i.e.:

```sh
kubectl apply -f https://raw.githubusercontent.com/<org>/musicservice/<tag or branch>/dist/install.yaml
```

## Contributing

This project demonstrates Kubernetes operator patterns and cloud-native application design.
Contributions are welcome - feel free to add improvements or additional features!

Some ideas for enhancement:
- Implement automated backup for databases
- Add Prometheus metrics with custom gauges
- Create webhooks for validation
- Enhance replica failure detection and recovery

**NOTE:** Run `make help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

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

