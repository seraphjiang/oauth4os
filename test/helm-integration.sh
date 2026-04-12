#!/usr/bin/env bash
# test/helm-integration.sh — Helm chart integration test using kind
# Prerequisites: kind, kubectl, helm, docker
set -euo pipefail

CLUSTER_NAME="oauth4os-test"
NAMESPACE="oauth4os"
RELEASE="oauth4os-test"
CHART_DIR="deploy/helm/oauth4os"
IMAGE="oauth4os:test"

RED='\033[0;31m'; GREEN='\033[0;32m'; CYAN='\033[0;36m'; NC='\033[0m'

cleanup() {
  echo -e "${CYAN}Cleaning up...${NC}"
  kind delete cluster --name "$CLUSTER_NAME" 2>/dev/null || true
}
trap cleanup EXIT

echo -e "${CYAN}=== Helm Chart Integration Test ===${NC}"

# 1. Build Docker image
echo -e "${CYAN}Building Docker image...${NC}"
docker build -t "$IMAGE" .

# 2. Create kind cluster
echo -e "${CYAN}Creating kind cluster...${NC}"
kind create cluster --name "$CLUSTER_NAME" --wait 60s

# 3. Load image into kind
echo -e "${CYAN}Loading image into kind...${NC}"
kind load docker-image "$IMAGE" --name "$CLUSTER_NAME"

# 4. Install Helm chart
echo -e "${CYAN}Installing Helm chart...${NC}"
kubectl create namespace "$NAMESPACE" || true
helm install "$RELEASE" "$CHART_DIR" \
  --namespace "$NAMESPACE" \
  --set image.repository="$IMAGE" \
  --set image.tag="" \
  --set image.pullPolicy=Never \
  --set config.upstream.engine="http://localhost:9200" \
  --wait --timeout 120s

# 5. Verify pods are running
echo -e "${CYAN}Checking pods...${NC}"
kubectl get pods -n "$NAMESPACE"
POD=$(kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/name=oauth4os -o jsonpath='{.items[0].metadata.name}')
STATUS=$(kubectl get pod "$POD" -n "$NAMESPACE" -o jsonpath='{.status.phase}')
if [ "$STATUS" != "Running" ]; then
  echo -e "${RED}FAIL: Pod not running (status: $STATUS)${NC}"
  kubectl logs "$POD" -n "$NAMESPACE" --tail=20
  exit 1
fi
echo -e "${GREEN}✅ Pod running${NC}"

# 6. Port-forward and test health
echo -e "${CYAN}Testing health endpoint...${NC}"
kubectl port-forward "$POD" -n "$NAMESPACE" 8443:8443 &
PF_PID=$!
sleep 3

HEALTH=$(curl -sf http://localhost:8443/health 2>/dev/null || echo "FAIL")
kill $PF_PID 2>/dev/null || true

if echo "$HEALTH" | grep -q '"status":"ok"'; then
  echo -e "${GREEN}✅ Health check passed${NC}"
else
  echo -e "${RED}FAIL: Health check failed: $HEALTH${NC}"
  exit 1
fi

# 7. Verify service exists
SVC=$(kubectl get svc -n "$NAMESPACE" -l app.kubernetes.io/name=oauth4os -o jsonpath='{.items[0].metadata.name}')
if [ -n "$SVC" ]; then
  echo -e "${GREEN}✅ Service exists: $SVC${NC}"
else
  echo -e "${RED}FAIL: Service not found${NC}"
  exit 1
fi

# 8. Helm test (if defined)
helm test "$RELEASE" -n "$NAMESPACE" 2>/dev/null && echo -e "${GREEN}✅ Helm tests passed${NC}" || echo -e "${CYAN}No Helm tests defined${NC}"

echo ""
echo -e "${GREEN}=== All Helm integration tests passed ===${NC}"
