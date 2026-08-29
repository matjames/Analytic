#!/bin/bash
# Deploy StatGate to Kubernetes

set -e

NAMESPACE="statgate"
DEPLOY_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "=========================================="
echo "StatGate Kubernetes Deployment"
echo "=========================================="

# Check kubectl is installed
if ! command -v kubectl &> /dev/null; then
    echo "❌ kubectl not found. Install it first."
    exit 1
fi

echo ""
echo "📦 Creating namespace and ConfigMaps..."
kubectl apply -f "$DEPLOY_DIR/00-namespace.yaml"

echo ""
echo "🗄️  Deploying PostgreSQL..."
kubectl apply -f "$DEPLOY_DIR/01-postgres.yaml"

echo ""
echo "⏳ Waiting for PostgreSQL to be ready..."
kubectl wait --for=condition=ready pod -l app=postgres -n $NAMESPACE --timeout=300s || true

echo ""
echo "🔴 Deploying Redis..."
kubectl apply -f "$DEPLOY_DIR/02-redis.yaml"

echo ""
echo "⏳ Waiting for Redis to be ready..."
kubectl wait --for=condition=ready pod -l app=redis -n $NAMESPACE --timeout=300s || true

echo ""
echo "🔗 Deploying Registry (API + UI)..."
kubectl apply -f "$DEPLOY_DIR/03-registry.yaml"

echo ""
echo "💬 Deploying StatChat (Backend + Frontend)..."
kubectl apply -f "$DEPLOY_DIR/04-statchat.yaml"

echo ""
echo "🎫 Deploying Helpdesk (API + UI)..."
kubectl apply -f "$DEPLOY_DIR/05-helpdesk.yaml"

echo ""
echo "📊 Deploying PMS & RMS (APIs + UIs)..."
kubectl apply -f "$DEPLOY_DIR/06-pms-rms.yaml"

echo ""
echo "⚖️  Deploying Governance & Analytics..."
kubectl apply -f "$DEPLOY_DIR/07-governance-analytics.yaml"

echo ""
echo "🌐 Deploying Ingress..."
kubectl apply -f "$DEPLOY_DIR/08-ingress.yaml"

echo ""
echo "=========================================="
echo "✅ Deployment Complete!"
echo "=========================================="

echo ""
echo "📊 Checking Deployment Status:"
kubectl get deployments -n $NAMESPACE

echo ""
echo "🔗 Service Endpoints:"
kubectl get svc -n $NAMESPACE

echo ""
echo "📝 Next Steps:"
echo "  1. Monitor pod status: kubectl get pods -n $NAMESPACE -w"
echo "  2. View logs: kubectl logs -n $NAMESPACE -f deployment/statgate-registry-api"
echo "  3. Port-forward UI: kubectl port-forward -n $NAMESPACE svc/statgate-registry-ui 3007:3007"
echo "  4. Configure Ingress hostname in /etc/hosts or DNS"
echo "  5. Access via: http://statgate.example.com/registry"

echo ""
echo "🔧 Troubleshooting:"
echo "  - Check pod events: kubectl describe pod -n $NAMESPACE <pod-name>"
echo "  - View resource usage: kubectl top nodes"
echo "  - Delete deployment: kubectl delete namespace $NAMESPACE"
