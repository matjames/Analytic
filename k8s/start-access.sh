#!/bin/bash
# StatGate System Access Script - macOS/Linux

echo "========================================"
echo "StatGate System Access"
echo "========================================"
echo ""
echo "Starting port-forwards in background..."
echo ""

# Function to cleanup on exit
cleanup() {
    echo ""
    echo "Stopping all port-forwards..."
    kill $(jobs -p) 2>/dev/null
    exit 0
}
trap cleanup EXIT

# Start all port-forwards in background
echo "Opening Helpdesk UI on http://localhost:3005"
kubectl port-forward -n statgate svc/statgate-helpdesk-ui 3005:3005 &

echo "Opening StaChat on http://localhost:3009"
kubectl port-forward -n statgate svc/statchat-frontend 3009:3009 &

echo "Opening PMS on http://localhost:3010"
kubectl port-forward -n statgate svc/statgate-pms-ui 3010:3010 &

echo "Opening RMS on http://localhost:3011"
kubectl port-forward -n statgate svc/statgate-rms-ui 3011:3011 &

echo "Opening Governance on http://localhost:3012"
kubectl port-forward -n statgate svc/statgate-governance-ui 3012:3012 &

echo "Opening Analytics on http://localhost:5000"
kubectl port-forward -n statgate svc/statgate-analytics 5000:5000 &

echo ""
echo "========================================"
echo "Port-forwards are running..."
echo "========================================"
echo ""
echo "Access your services:"
echo "  Helpdesk:   http://localhost:3005"
echo "  StaChat:    http://localhost:3009"
echo "  PMS:        http://localhost:3010"
echo "  RMS:        http://localhost:3011"
echo "  Governance: http://localhost:3012"
echo "  Analytics:  http://localhost:5000"
echo ""
echo "Press Ctrl+C to stop all port-forwards"
echo ""

# Wait for background jobs
wait
