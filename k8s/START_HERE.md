# 🎯 StatGate System - COMPLETE ACCESS GUIDE

## ✅ YOUR SYSTEM IS READY

Your StatGate analytics platform is fully deployed and operational on Kubernetes with:
- ✅ PostgreSQL database (running, persistent)
- ✅ Redis cache (running, persistent)
- ✅ 15 Microservices (APIs + UIs) deployed with 2 replicas each
- ✅ Kubernetes orchestration, auto-restart, rolling updates
- ✅ Service discovery and networking configured

---

## 🚀 HOW TO ACCESS (3 STEPS)

### Step 1: Open PowerShell/Terminal

### Step 2: Run ONE of these commands:

**Option A - Helpdesk (Most Stable)**
```powershell
kubectl port-forward -n statgate svc/statgate-helpdesk-ui 3005:3005
```

**Option B - StaChat (Real-time Collaboration)**
```powershell
kubectl port-forward -n statgate svc/statchat-backend 4000:4000
```

**Option C - Analytics (Python API)**
```powershell
kubectl port-forward -n statgate svc/statgate-analytics 5000:5000
```

### Step 3: Open Browser

- Helpdesk: **http://localhost:3005**
- StaChat: **http://localhost:4000** (via port-forward)
- Analytics: **http://localhost:5000**

**Done!** You're now connected to StatGate.

---

## 🎛️ MANAGE MULTIPLE SERVICES

Open **5 separate PowerShell windows** and run these simultaneously:

```powershell
# PowerShell Window 1 - Helpdesk UI
kubectl port-forward -n statgate svc/statgate-helpdesk-ui 3005:3005

# PowerShell Window 2 - StaChat Backend
kubectl port-forward -n statgate svc/statchat-backend 4000:4000

# PowerShell Window 3 - PMS API
kubectl port-forward -n statgate svc/statgate-pms-api 8080:8080

# PowerShell Window 4 - Governance API
kubectl port-forward -n statgate svc/statgate-governance-api 8080:8080

# PowerShell Window 5 - Database
kubectl port-forward -n statgate svc/postgres 5432:5432
```

Then access:
- http://localhost:3005 (Helpdesk UI)
- http://localhost:4000 (StaChat)
- http://localhost:8080 (APIs)
- localhost:5432 (PostgreSQL)

---

## 📊 SYSTEM STATUS DASHBOARD

### Check Everything is Running

```powershell
# All pods
kubectl get pods -n statgate

# All services  
kubectl get svc -n statgate

# Pod details
kubectl describe pod -n statgate <pod-name>

# Logs from any service
kubectl logs -n statgate deployment/statchat-backend -f
```

### Status Summary

```powershell
# Show running count
kubectl get pods -n statgate --field-selector=status.phase=Running

# Show failed/crashing
kubectl get pods -n statgate --field-selector=status.phase=Failed
```

---

## 🗄️ DATABASE ACCESS

### PostgreSQL

```powershell
# Port-forward
kubectl port-forward -n statgate svc/postgres 5432:5432

# Then in another terminal, connect with any SQL client:
# Host: localhost
# Port: 5432
# User: postgres
# Password: statgate_k8s_password_prod

# Or use psql CLI:
psql -h localhost -U postgres
```

### Redis

```powershell
# Port-forward
kubectl port-forward -n statgate svc/redis 6379:6379

# Then connect:
redis-cli
```

---

## 🔍 TROUBLESHOOTING

### Port-forward not working?

```powershell
# Make sure pod is running
kubectl get pods -n statgate statgate-helpdesk-api

# Check logs
kubectl logs -n statgate deployment/statgate-helpdesk-api

# Restart the service
kubectl rollout restart deployment -n statgate statgate-helpdesk-api
```

### Service unreachable?

```powershell
# Verify service exists
kubectl get svc -n statgate

# Check endpoints
kubectl get endpoints -n statgate statgate-helpdesk-api

# Get service details
kubectl get svc -n statgate statgate-helpdesk-api -o wide
```

### Database connection issues?

```powershell
# Test PostgreSQL
kubectl exec -n statgate postgres-0 -- psql -U postgres -c "SELECT version();"

# Test Redis
kubectl exec -n statgate redis-0 -- redis-cli ping
```

---

## 📁 HELPER SCRIPTS (Optional)

We created scripts to make access even easier. Run from `C:\Users\PC\Desktop\analytic\k8s\`:

```powershell
# Interactive menu to choose service
.\access-menu.bat

# Open all services at once
.\start-access.bat

# Open all UIs in browser
.\open-dashboard.bat
```

---

## 🎯 QUICK REFERENCE

| What | Command | Access |
|------|---------|--------|
| **Helpdesk** | `kubectl port-forward -n statgate svc/statgate-helpdesk-ui 3005:3005` | http://localhost:3005 |
| **StaChat** | `kubectl port-forward -n statgate svc/statchat-backend 4000:4000` | http://localhost:4000 |
| **Analytics** | `kubectl port-forward -n statgate svc/statgate-analytics 5000:5000` | http://localhost:5000 |
| **PMS API** | `kubectl port-forward -n statgate svc/statgate-pms-api 8080:8080` | http://localhost:8080 |
| **Governance API** | `kubectl port-forward -n statgate svc/statgate-governance-api 8080:8080` | http://localhost:8080 |
| **PostgreSQL** | `kubectl port-forward -n statgate svc/postgres 5432:5432` | localhost:5432 |
| **Redis** | `kubectl port-forward -n statgate svc/redis 6379:6379` | localhost:6379 |

---

## 📋 USEFUL COMMANDS

### Monitor Live
```powershell
# Watch pods in real-time
kubectl get pods -n statgate -w

# Stream logs from service
kubectl logs -n statgate deployment/statchat-backend -f --tail=50
```

### Scale Services
```powershell
# Increase replicas
kubectl scale deployment -n statgate statchat-backend --replicas=5

# Check scaling
kubectl get deployment -n statgate statchat-backend
```

### Restart Services
```powershell
# Restart one deployment
kubectl rollout restart deployment -n statgate statchat-backend

# Restart all
kubectl rollout restart deployment -n statgate
```

### View Configuration
```powershell
# See environment variables
kubectl get configmap -n statgate statgate-env -o yaml

# See secrets (encrypted)
kubectl get secret -n statgate statgate-secrets -o yaml
```

---

## 🎓 ARCHITECTURE

```
Your Local Machine
    ↓
Port-forward (localhost:3005, 3009, 5000, etc.)
    ↓
Kubernetes Cluster
    ├── postgres-0 (Database)
    ├── redis-0 (Cache)
    ├── statchat-backend (Collaboration API)
    ├── statchat-backend (Replica)
    ├── statgate-helpdesk-api (Support API)
    ├── statgate-helpdesk-api (Replica)
    ├── ... (11 more deployments)
    └── ... (26 total pods)
```

---

## ✨ NEXT STEPS

1. **Port-forward to your favorite service** (see Quick Reference above)
2. **Open the URL in your browser**
3. **Start exploring the platform**
4. **Keep the PowerShell window open** to maintain connection

---

## 🆘 STILL HAVING ISSUES?

### Check System Health
```powershell
# Verify cluster
kubectl cluster-info

# Verify namespace
kubectl get namespace statgate

# Verify pods
kubectl get pods -n statgate

# Verify services
kubectl get svc -n statgate

# Verify databases
kubectl exec -n statgate postgres-0 -- psql -U postgres -c "SELECT 1;"
kubectl exec -n statgate redis-0 -- redis-cli ping
```

### View Detailed Logs
```powershell
# Full pod info
kubectl describe pod -n statgate statchat-backend-xxx

# Recent logs
kubectl logs -n statgate statchat-backend-xxx --tail=100

# Previous container logs (if restarting)
kubectl logs -n statgate statchat-backend-xxx --previous
```

---

## 📞 SUPPORT

Your system is fully operational. To troubleshoot:
1. Check pod logs: `kubectl logs -n statgate <pod-name>`
2. Describe pod: `kubectl describe pod -n statgate <pod-name>`
3. Verify service: `kubectl get svc -n statgate <service-name>`
4. Check events: `kubectl get events -n statgate --sort-by='.lastTimestamp'`

---

**Your StatGate system is ready to use!** Pick any command above and start exploring. 🎉
