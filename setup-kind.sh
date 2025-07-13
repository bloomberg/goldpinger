#!/bin/bash

# Prerequisites: Make sure you have kind and kubectl installed
# Install kind: https://kind.sigs.k8s.io/docs/user/quick-start/#installation
# Install kubectl: https://kubernetes.io/docs/tasks/tools/

# I recommend running each of these commands separately to understand what they do.

# 1. Create the cluster
echo "Creating goldpinger test cluster..."
kind create cluster --config kind-config-goldpinger.yaml

# 2. Wait for cluster to be ready
echo "Waiting for cluster to be ready..."
kubectl wait --for=condition=Ready nodes --all --timeout=300s

# 3. Deploy goldpinger
echo "Deploying goldpinger..."
kubectl apply -f goldpinger-deployment.yaml

# 4. Wait for goldpinger pods to be ready
echo "Waiting for goldpinger pods..."
kubectl wait --for=condition=Ready pod -l app=goldpinger --timeout=120s

# 5. Check deployment
echo "Checking goldpinger deployment..."
kubectl get pods -o wide -l app=goldpinger

# Check connectivity via CLI:
kubectl port-forward svc/goldpinger 8080:8080

# View logs from all goldpinger pods:
kubectl logs -l app=goldpinger --all-containers=true

# Check which nodes have goldpinger running:
kubectl get pods -l app=goldpinger -o wide

# Test connectivity manually:
kubectl exec -it <goldpinger-pod> -- wget -qO- http://goldpinger:8080/check_all

# 7. Port forward to access the UI
echo "Setting up port forwarding..."
kubectl port-forward svc/goldpinger 8080:8080 &
PF_PID=$!

echo "
Goldpinger UI available at: http://localhost:8080
Press Ctrl+C to stop port forwarding (PID: $PF_PID)
"

# Keep the port forward running
wait $PF_PID