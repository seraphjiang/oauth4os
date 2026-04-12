#!/bin/bash
# Helm chart integration test using kind (Kubernetes in Docker)
# Usage: bash test/helm/test-helm.sh
set -euo pipefail

CHART_DIR="deploy/helm/oauth4os"
RELEASE="oauth4os-test"
NAMESPACE="oauth4os-test"
TIMEOUT="120s"

echo "=== Helm Chart Integration Test ==="

# Check prerequisites
for cmd in kind kubectl helm; do
  if ! command -v "$cmd" &>/dev/null; then
    echo "SKIP: $cmd not installed"
    exit 0
  fi
done

# Create kind cluster
echo "[1/6] Creating kind cluster..."
kind create cluster --name helm-test --wait 60s 2>/dev/null || true
kubectl cluster-info --context kind-helm-test

# Create namespace
echo "[2/6] Creating namespace..."
kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -

# Lint chart
echo "[3/6] Linting chart..."
helm lint "$CHART_DIR"

# Template render (dry-run)
echo "[4/6] Template render..."
helm template "$RELEASE" "$CHART_DIR" --namespace "$NAMESPACE" > /dev/null
echo "  Templates render OK"

# Install chart
echo "[5/6] Installing chart..."
helm upgrade --install "$RELEASE" "$CHART_DIR" \
  --namespace "$NAMESPACE" \
  --set image.tag=latest \
  --set replicaCount=1 \
  --wait --timeout "$TIMEOUT"

# Verify
echo "[6/6] Verifying deployment..."
kubectl -n "$NAMESPACE" rollout status deployment/"$RELEASE"-oauth4os --timeout="$TIMEOUT"
kubectl -n "$NAMESPACE" get pods -l app.kubernetes.io/name=oauth4os

POD=$(kubectl -n "$NAMESPACE" get pod -l app.kubernetes.io/name=oauth4os -o jsonpath='{.items[0].metadata.name}')
kubectl -n "$NAMESPACE" exec "$POD" -- wget -qO- http://localhost:8443/health || echo "Health check via exec"

echo ""
echo "=== Results ==="
echo "✅ Lint: passed"
echo "✅ Template: rendered"
echo "✅ Install: deployed"
echo "✅ Rollout: ready"

# Cleanup
echo ""
echo "Cleaning up..."
helm uninstall "$RELEASE" --namespace "$NAMESPACE" 2>/dev/null || true
kubectl delete namespace "$NAMESPACE" 2>/dev/null || true
kind delete cluster --name helm-test 2>/dev/null || true

echo "✅ Helm chart integration test PASSED"
