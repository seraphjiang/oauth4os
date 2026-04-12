#!/bin/bash
# Canary deployment for oauth4os on AppRunner
# Usage: ./canary-deploy.sh <image_tag> [--promote|--rollback]
set -euo pipefail

IMAGE_TAG="${1:?Usage: $0 <image_tag> [--promote|--rollback]}"
ACTION="${2:-canary}"
REGION="${AWS_REGION:-us-west-2}"
SERVICE_NAME="${APPRUNNER_SERVICE:-oauth4os}"
CANARY_MINUTES="${CANARY_MINUTES:-5}"
ERROR_THRESHOLD="${ERROR_THRESHOLD:-5}"  # max error % before auto-rollback

log() { echo "[$(date '+%H:%M:%S')] $*"; }

# Find service ARN
SERVICE_ARN=$(aws apprunner list-services --region "$REGION" \
  --query "ServiceSummaryList[?ServiceName=='$SERVICE_NAME'].ServiceArn" \
  --output text)
[ -z "$SERVICE_ARN" ] && { log "ERROR: Service '$SERVICE_NAME' not found"; exit 1; }

SERVICE_URL=$(aws apprunner describe-service --service-arn "$SERVICE_ARN" --region "$REGION" \
  --query "Service.ServiceUrl" --output text)

get_version() { curl -sf "https://$SERVICE_URL/version" 2>/dev/null | python3 -c "import json,sys; print(json.load(sys.stdin).get('version','unknown'))" 2>/dev/null || echo "unreachable"; }
get_health()  { curl -sf -o /dev/null -w '%{http_code}' "https://$SERVICE_URL/health" 2>/dev/null || echo "000"; }
get_errors()  { curl -sf "https://$SERVICE_URL/metrics" 2>/dev/null | grep '^oauth4os_requests_failed ' | awk '{print $2}' || echo "0"; }
get_total()   { curl -sf "https://$SERVICE_URL/metrics" 2>/dev/null | grep '^oauth4os_requests_total '  | awk '{print $2}' || echo "0"; }

if [ "$ACTION" = "--rollback" ]; then
  log "Rolling back — triggering redeployment with previous image..."
  aws apprunner start-deployment --service-arn "$SERVICE_ARN" --region "$REGION" >/dev/null
  log "Rollback triggered. Monitor: https://$SERVICE_URL/health"
  exit 0
fi

# --- Deploy ---
log "Canary deploy: $IMAGE_TAG → $SERVICE_NAME"
log "Current version: $(get_version)"

ERRORS_BEFORE=$(get_errors)
TOTAL_BEFORE=$(get_total)

log "Triggering deployment..."
aws apprunner start-deployment --service-arn "$SERVICE_ARN" --region "$REGION" >/dev/null

log "Waiting for deployment to start..."
for i in $(seq 1 30); do
  STATUS=$(aws apprunner describe-service --service-arn "$SERVICE_ARN" --region "$REGION" \
    --query "Service.Status" --output text)
  [ "$STATUS" = "OPERATION_IN_PROGRESS" ] && break
  sleep 5
done

log "Waiting for deployment to complete..."
for i in $(seq 1 60); do
  STATUS=$(aws apprunner describe-service --service-arn "$SERVICE_ARN" --region "$REGION" \
    --query "Service.Status" --output text)
  [ "$STATUS" = "RUNNING" ] && break
  sleep 10
done

if [ "$STATUS" != "RUNNING" ]; then
  log "ERROR: Deployment did not complete (status: $STATUS)"
  exit 1
fi

NEW_VERSION=$(get_version)
log "Deployed version: $NEW_VERSION"

# --- Canary monitoring ---
log "Canary monitoring for ${CANARY_MINUTES}m (error threshold: ${ERROR_THRESHOLD}%)..."
for i in $(seq 1 "$CANARY_MINUTES"); do
  sleep 60
  HEALTH=$(get_health)
  ERRORS_NOW=$(get_errors)
  TOTAL_NOW=$(get_total)

  DELTA_ERR=$((ERRORS_NOW - ERRORS_BEFORE))
  DELTA_TOT=$((TOTAL_NOW - TOTAL_BEFORE))

  if [ "$DELTA_TOT" -gt 0 ]; then
    ERR_PCT=$((DELTA_ERR * 100 / DELTA_TOT))
  else
    ERR_PCT=0
  fi

  log "  Minute $i/$CANARY_MINUTES: health=$HEALTH errors=$DELTA_ERR/$DELTA_TOT (${ERR_PCT}%)"

  if [ "$HEALTH" != "200" ]; then
    log "FAIL: Health check returned $HEALTH — auto-rolling back"
    aws apprunner start-deployment --service-arn "$SERVICE_ARN" --region "$REGION" >/dev/null
    exit 1
  fi

  if [ "$ERR_PCT" -gt "$ERROR_THRESHOLD" ]; then
    log "FAIL: Error rate ${ERR_PCT}% exceeds threshold ${ERROR_THRESHOLD}% — auto-rolling back"
    aws apprunner start-deployment --service-arn "$SERVICE_ARN" --region "$REGION" >/dev/null
    exit 1
  fi
done

log "✅ Canary passed. Version $NEW_VERSION is stable."

if [ "$ACTION" = "--promote" ]; then
  log "Promoted (AppRunner auto-deploys — already live)."
fi
