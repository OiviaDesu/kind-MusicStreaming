# Complete Testing Instructions

## Prerequisites

Ensure you have the following installed:
- Go 1.22.0+
- Docker 17.03+
- kubectl v1.11.3+
- Kind (for local cluster testing)

## Step 1: Build the Operator

```bash
cd /home/oiviadesu/OiviaKind

# Generate code and manifests
make manifests
make generate

# Format code
go fmt ./...

# Run linter checks
go vet ./...
```

Expected output: No errors, all code properly formatted.

## Step 2: Run Unit Tests

### Option A: Using Make (Recommended)
The Makefile properly sets up the kubebuilder test environment:

```bash
make test
```

This command:
1. Regenerates code and manifests
2. Runs go fmt to format code
3. Runs go vet for static analysis
4. Sets up envtest with K8s v1.30.0
5. Executes all unit tests (excluding e2e)
6. Generates coverage profile

Expected output:
```
ok  	github.com/example/managedapp-operator/internal/builder       [coverage: XX.X%]
ok  	github.com/example/managedapp-operator/internal/status        [coverage: XX.X%]
ok  	github.com/example/managedapp-operator/internal/controller    [coverage: XX.X%]
```

### Option B: Manual Test Execution

If `make test` has issues, run tests individually:

```bash
# Set up envtest and get the path to kubebuilder assets
export KUBEBUILDER_ASSETS=$(go run sigs.k8s.io/controller-runtime/tools/setup-envtest@latest use 1.30.0 -p path)

# Run builder tests
go test -v -coverprofile=builder.out ./internal/builder

# Run status tests
go test -v -coverprofile=status.out ./internal/status

# Run controller tests
go test -v -coverprofile=controller.out ./internal/controller
```

### Option C: Run Specific Tests

```bash
# Run only condition management tests
go test -run TestSetCondition ./internal/status -v

# Run only resource builder tests  
go test -run TestResourceBuilder ./internal/builder -v

# Run only MusicService controller tests
go test -run "MusicService" ./internal/controller -v
```

## Step 3: Coverage Analysis

After tests complete, analyze coverage:

```bash
# View coverage summary
go tool cover -func=cover.out | tail -20

# Generate HTML coverage report
go tool cover -html=cover.out -o coverage.html

# Open in browser
open coverage.html  # macOS
xdg-open coverage.html  # Linux
```

Target coverage: > 80% for critical paths

## Step 4: Deploy to Kind and Test Integration

```bash
# Create Kind cluster (if not exists)
make kind-create

# Build and deploy operator
make deploy-kind

# Wait for operator deployment
kubectl rollout status deployment/music-operator-controller-manager -n music-operator-system

# Apply sample MusicService
kubectl apply -f config/samples/musicservice_sample.yaml

# Verify resources created
kubectl get musicservices
kubectl get statefulsets
kubectl get services
kubectl get pods
```

Expected state:
- 1 MusicService created: miku-stream
- 3 app replicas running
- 2 database replicas running (if database enabled)
- All StatefulSets showing desired replicas ready

## Step 5: Verify Database Replication

```bash
# Check if replication secret exists
kubectl get secret miku-stream-db-replication -o yaml

# Verify master database is running
kubectl get pod miku-stream-db-master-0
kubectl exec pod/miku-stream-db-master-0 -- mysql -u root -p$ROOT_PASSWORD -e "SELECT VERSION();"

# Check replica status
kubectl exec pod/miku-stream-db-replica-0 -- mysql -u root -p$ROOT_PASSWORD -e "SHOW SLAVE STATUS\G"

# Expected output for replicas:
# Slave_IO_Running: Yes
# Slave_SQL_Running: Yes
# Seconds_Behind_Master: 0
```

## Step 6: Verify Status Updates

```bash
# Check MusicService status
kubectl get musicservice miku-stream -o yaml | grep -A 20 "^status:"

# Expected status shows:
# - phase: Available
# - readyReplicas: 3
# - conditions: Available=True, Reconciled=True
# - database.masterReady: true
# - database.replicasReady: 2
```

## Step 7: Test Auto-Scaling

```bash
# Check HPA is created
kubectl get hpa

# Expected output:
# miku-stream-autoscaler: 2-10 replicas
# miku-stream-db-replica-autoscaler: 1-5 replicas (if DB enabled)

# Monitor CPU metrics
kubectl top pod
kubectl top nodes

# Generate load (optional, to trigger scaling)
# kubectl exec -it pod/miku-stream-0 -- /bin/sh
```

## Step 8: Cleanup

```bash
# Delete sample resources
kubectl delete -f config/samples/musicservice_sample.yaml

# Uninstall operator
make undeploy

# Delete Kind cluster  
make kind-delete
```

## Troubleshooting

### Issue: `make test` hangs
**Solution:**
```bash
# Check if envtest is properly installed
go run sigs.k8s.io/controller-runtime/tools/setup-envtest@latest list

# Manually set assets path and run
export KUBEBUILDER_ASSETS="/path/to/kubebuilder/assets"
go test -timeout 120s ./internal/builder
```

### Issue: Tests timeout
**Solution:**
```bash
# Increase timeout
go test -timeout 300s ./...

# Run with verbose logging
go test -v -timeout 300s ./internal/controller
```

### Issue: Pod not starting
**Solution:**
```bash
# Check pod logs
kubectl logs -f deployment/music-operator-controller-manager -n music-operator-system

# Check events
kubectl get events --field-selector involvedObject.kind=Pod

# Describe pod for detailed info
kubectl describe pod music-operator-controller-manager-xxxxx -n music-operator-system
```

### Issue: Database not syncing
**Solution:**
```bash
# Check replication secret exists
kubectl get secret miku-stream-db-replication

# Verify slave init script ran
kubectl logs pod/miku-stream-db-replica-0

# Manually start replication if needed
kubectl exec pod/miku-stream-db-replica-0 -- mysql -u root -p$PASSWORD << EOF
START SLAVE;
SHOW SLAVE STATUS\G
EOF
```

## Expected Test Results Summary

When all tests pass successfully:

1. **Unit Tests**
   - ✅ Builder creates valid StatefulSets/Services
   - ✅ Status manager properly sets conditions
   - ✅ Controller reconciliation works end-to-end

2. **Integration Tests (Kind)**
   - ✅ Operator deploys successfully
   - ✅ MusicService resources created immediately
   - ✅ App StatefulSet scales to desired replicas
   - ✅ Database master/replicas running (if enabled)
   - ✅ Replication syncing data (if enabled)
   - ✅ HPA managing scaling (if autoscaling enabled)
   - ✅ Status conditions updating properly
   - ✅ No errors in controller logs

3. **Coverage Goals**
   - ✅ Core reconciliation logic: >90%
   - ✅ Status management: >85%
   - ✅ Resource builders: >80%
   - ✅ Overall project: >80%

## Quick Start Command

Run all tests in sequence:

```bash
# Clean build and test in one command
cd /home/oiviadesu/OiviaKind && \
  make manifests && \
  make generate && \
  go fmt ./... && \
  go vet ./... && \
  make test && \
  make kind-create && \
  make deploy-kind && \
  kubectl apply -f config/samples/musicservice_sample.yaml && \
  echo "All tests completed! Verify with: kubectl get musicservices"
```

## Documentation References

- [Kubebuilder Testing Guide](https://book.kubebuilder.io/testing.html)
- [Ginkgo Documentation](http://onsi.github.io/ginkgo/)
- [Gomega Matchers](http://onsi.github.io/gomega/)
- [Go Testing Best Practices](https://golang.org/doc/effective_go#testing)

## Summary

The test infrastructure provides:
- ✅ Unit tests for all components
- ✅ Integration tests with real Kubernetes API
- ✅ Coverage reporting
- ✅ CI/CD-ready test pipeline
- ✅ Easy debugging with verbose output
- ✅ Production-quality verification

All tests follow Go and kubebuilder best practices, ensuring the MusicService operator is reliable and production-ready.
