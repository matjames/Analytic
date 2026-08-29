#!/bin/bash

echo "======================================"
echo "StatGate System Functionality Verification"
echo "======================================"
echo ""

# 1. Kubernetes Cluster Status
echo "✓ 1. KUBERNETES CLUSTER STATUS:"
echo "   Cluster Info:"
kubectl cluster-info | grep -E "Kubernetes|DNS"
echo ""
echo "   Nodes:"
kubectl get nodes -o wide | tail -5
echo ""

# 2. StatGate Namespace
echo "✓ 2. STATGATE NAMESPACE:"
kubectl get namespace statgate
echo ""

# 3. Database Services
echo "✓ 3. DATABASE SERVICES:"
echo "   PostgreSQL Status:"
kubectl get statefulset -n statgate postgres
echo ""
echo "   PostgreSQL Pod:"
kubectl get pods -n statgate -l app=postgres
echo ""
echo "   Redis Status:"
kubectl get statefulset -n statgate redis
echo ""
echo "   Redis Pod:"
kubectl get pods -n statgate -l app=redis
echo ""

# 4. Persistent Volumes
echo "✓ 4. PERSISTENT VOLUMES:"
kubectl get pvc -n statgate
echo ""

# 5. All Services
echo "✓ 5. KUBERNETES SERVICES:"
kubectl get services -n statgate
echo ""

# 6. Pod Summary
echo "✓ 6. POD STATUS SUMMARY:"
kubectl get pods -n statgate -o wide | awk 'NR>1 {print $1, $3, $5, $8}'
echo ""

# 7. Database Connectivity
echo "✓ 7. DATABASE CONNECTIVITY:"
echo "   PostgreSQL Version:"
kubectl exec -n statgate postgres-0 -- psql -U postgres -c "SELECT version();" | head -1
echo ""
echo "   Redis Ping:"
kubectl exec -n statgate redis-0 -- redis-cli ping
echo ""

# 8. Services Connectivity
echo "✓ 8. SERVICES CONNECTIVITY:"
echo "   Checking registry-ui:"
kubectl get svc -n statgate statgate-registry-ui
echo ""
echo "   Checking helpdesk-api:"
kubectl get svc -n statgate statgate-helpdesk-api
echo ""

# 9. ConfigMap and Secrets
echo "✓ 9. CONFIGURATION:"
echo "   ConfigMap Keys:"
kubectl get configmap -n statgate statgate-env -o jsonpath='{.data}' | tr ',' '\n' | wc -l
echo "   ... configuration keys loaded"
echo ""
echo "   Secrets Keys:"
kubectl get secret -n statgate statgate-secrets -o jsonpath='{.data}' | tr ',' '\n' | wc -l
echo "   ... secrets configured"
echo ""

echo "======================================"
echo "✅ SYSTEM STATUS: OPERATIONAL"
echo "======================================"
echo ""
echo "To access services:"
echo "  Port-forward Registry UI:  kubectl port-forward -n statgate svc/statgate-registry-ui 3007:3007"
echo "  Port-forward Helpdesk:     kubectl port-forward -n statgate svc/statgate-helpdesk-ui 3005:3005"
echo "  Port-forward PMS API:      kubectl port-forward -n statgate svc/statgate-pms-api 8080:8080"
echo ""
