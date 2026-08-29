# StatGate System - Quick Commands Reference

## 🚀 Quick Start

### Fastest Way (Copy & Paste)
```powershell
# Open PowerShell and run:
kubectl port-forward -n statgate svc/statgate-helpdesk-ui 3005:3005
# Then open: http://localhost:3005
```

---

## 📊 All Services at Once

```powershell
# Run these in separate PowerShell windows:

# 1. Helpdesk
kubectl port-forward -n statgate svc/statgate-helpdesk-ui 3005:3005

# 2. StaChat
kubectl port-forward -n statgate svc/statchat-frontend 3009:3009

# 3. PMS
kubectl port-forward -n statgate svc/statgate-pms-ui 3010:3010

# 4. RMS
kubectl port-forward -n statgate svc/statgate-rms-ui 3011:3011

# 5. Governance
kubectl port-forward -n statgate svc/statgate-governance-ui 3012:3012

# 6. Analytics
kubectl port-forward -n statgate svc/statgate-analytics 5000:5000
```

**Then open in browser:**
- http://localhost:3005 (Helpdesk)
- http://localhost:3009 (StaChat)
- http://localhost:3010 (PMS)
- http://localhost:3011 (RMS)
- http://localhost:3012 (Governance)
- http://localhost:5000 (Analytics)

---

## 📋 System Status

```powershell
# Check all pods running
kubectl get pods -n statgate

# Check all services
kubectl get svc -n statgate

# View logs from any service
kubectl logs -n statgate deployment/statgate-helpdesk-api -f

# Check pod resource usage
kubectl top pods -n statgate
```

---

## 🗄️ Database Access

```powershell
# PostgreSQL
kubectl port-forward -n statgate svc/postgres 5432:5432
# Connect: psql -h localhost -U postgres

# Redis
kubectl port-forward -n statgate svc/redis 6379:6379
# Connect: redis-cli
```

---

## 🔧 Troubleshooting

```powershell
# Check if service is running
kubectl get pods -n statgate -l app=statgate-helpdesk-ui

# View detailed pod info
kubectl describe pod -n statgate <pod-name>

# View pod logs
kubectl logs -n statgate <pod-name>

# Restart a service
kubectl rollout restart deployment -n statgate statgate-helpdesk-api

# Get service details
kubectl get svc -n statgate statgate-helpdesk-ui -o wide
```

---

## 📁 Helper Scripts

We created these scripts for you:

- **start-access.bat** - Opens all services in separate windows
- **open-dashboard.bat** - Opens all services in browser tabs
- **access-menu.bat** - Interactive menu to choose which service to access
- **start-access.sh** - macOS/Linux version of start-access.bat

**Usage:**
```powershell
# Windows
cd C:\Users\PC\Desktop\analytic\k8s
.\start-access.bat

# Or run the menu
.\access-menu.bat
```

---

## 🌐 Service Details

| Service | Port | What It Is |
|---------|------|-----------|
| Helpdesk UI | 3005 | Web UI for operations support |
| Helpdesk API | 5000 | REST API for helpdesk functions |
| StaChat | 3009 | Real-time collaboration platform |
| PMS | 3010 | Projects Management System |
| RMS | 3011 | Research Management System |
| Governance | 3012 | Institutional governance platform |
| Analytics | 5000 | Python Flask analytics engine |
| PostgreSQL | 5432 | Database server |
| Redis | 6379 | Cache & event bus |

---

## ✅ Verification Commands

```powershell
# Verify cluster is ready
kubectl cluster-info

# Verify namespace exists
kubectl get namespace statgate

# Verify all pods running
kubectl get pods -n statgate --field-selector=status.phase=Running

# Verify databases are ready
kubectl exec -n statgate postgres-0 -- psql -U postgres -c "SELECT version();"
kubectl exec -n statgate redis-0 -- redis-cli ping

# Verify services are created
kubectl get svc -n statgate

# Check system health
kubectl describe node
```

---

## 🎯 Common Tasks

### Scale a Service (more replicas)
```powershell
kubectl scale deployment -n statgate statgate-helpdesk-api --replicas=5
```

### View Real-time Logs
```powershell
kubectl logs -n statgate deployment/statgate-helpdesk-api -f
```

### Restart a Service
```powershell
kubectl rollout restart deployment -n statgate statgate-helpdesk-api
```

### Delete Entire System
```powershell
kubectl delete namespace statgate
```

### Redeploy Everything
```powershell
cd k8s
kubectl apply -f 00-namespace.yaml
kubectl apply -f 01-postgres.yaml
kubectl apply -f 02-redis.yaml
# ... apply remaining files
```

---

## 💡 Pro Tips

1. **Use multiple terminal tabs** - Open one PowerShell tab per service for concurrent access

2. **Keep terminals organized** - Name each window by service for easy management

3. **Monitor while developing** - Run `kubectl get pods -n statgate -w` in one terminal to watch pods live

4. **Save bandwidth** - Only port-forward services you're currently using

5. **Database backups** - Regularly export PostgreSQL data:
   ```powershell
   kubectl exec -n statgate postgres-0 -- pg_dump -U postgres > backup.sql
   ```

---

## 📞 Need Help?

- Check pod logs: `kubectl logs -n statgate <pod-name>`
- Describe pod: `kubectl describe pod -n statgate <pod-name>`
- Check events: `kubectl get events -n statgate`
- View manifest: `kubectl get pod -n statgate <pod-name> -o yaml`
