# How to Access StatGate System

Your StatGate platform is fully deployed on Kubernetes. Here are all the ways to access it:

## Quick Start (Easiest)

### 1. Access Services via Port-Forwarding

Open PowerShell/Terminal and run these commands:

```bash
# Access Helpdesk UI (most stable)
kubectl port-forward -n statgate svc/statgate-helpdesk-ui 3005:3005
# Then open: http://localhost:3005

# Access StaChat Frontend (real-time collaboration)
kubectl port-forward -n statgate svc/statchat-frontend 3009:3009
# Then open: http://localhost:3009

# Access PMS UI (Projects Management)
kubectl port-forward -n statgate svc/statgate-pms-ui 3010:3010
# Then open: http://localhost:3010

# Access RMS UI (Research Management)
kubectl port-forward -n statgate svc/statgate-rms-ui 3011:3011
# Then open: http://localhost:3011

# Access Governance UI
kubectl port-forward -n statgate svc/statgate-governance-ui 3012:3012
# Then open: http://localhost:3012

# Access Analytics API (Python Flask)
kubectl port-forward -n statgate svc/statgate-analytics 5000:5000
# Then open: http://localhost:5000

# Access Helpdesk API
kubectl port-forward -n statgate svc/statgate-helpdesk-api 5000:5000
# Then open: http://localhost:5000/api-docs
```

### 2. Multiple Port-Forwards (Recommended for Development)

Run this in separate terminal windows:

**Terminal 1:**
```bash
kubectl port-forward -n statgate svc/statgate-helpdesk-ui 3005:3005
```

**Terminal 2:**
```bash
kubectl port-forward -n statgate svc/statchat-frontend 3009:3009
```

**Terminal 3:**
```bash
kubectl port-forward -n statgate svc/statgate-governance-api 8080:8080
```

Then open your browser and visit:
- http://localhost:3005 (Helpdesk)
- http://localhost:3009 (StaChat)
- http://localhost:8080/health (Governance API health check)

---

## Production Access (Ingress + DNS)

### 1. Set Up Local DNS (macOS/Linux/Windows)

Edit your hosts file:

**Linux/macOS:**
```bash
sudo nano /etc/hosts
# Add:
127.0.0.1 statgate.example.com
```

**Windows:**
```
C:\Windows\System32\drivers\etc\hosts
# Add:
127.0.0.1 statgate.example.com
```

### 2. Enable Ingress Controller (Docker Desktop)

```bash
# Install NGINX Ingress
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.8.0/deploy/static/provider/cloud/deploy.yaml

# Wait for it to be ready
kubectl wait --for=condition=ready pod -l app.kubernetes.io/component=controller -n ingress-nginx --timeout=300s
```

### 3. Deploy Ingress

```bash
kubectl apply -f k8s/08-ingress.yaml
```

### 4. Access via Browser

Then visit:
```
http://statgate.example.com/registry
http://statgate.example.com/helpdesk
http://statgate.example.com/pms
http://statgate.example.com/governance
http://statgate.example.com/statchat
http://statgate.example.com/analytics
```

---

## Cloud Deployment Access

### AWS EKS
```bash
# Get the Load Balancer URL
kubectl get svc -n ingress-nginx

# Add DNS record pointing to the Load Balancer
# Access via: https://your-domain.com/helpdesk
```

### Azure AKS
```bash
# Get Public IP
kubectl get svc -n ingress-nginx

# Configure Azure DNS to point to Public IP
# Access via: https://your-domain.com/helpdesk
```

### DigitalOcean
```bash
# Load Balancer auto-assigned
kubectl get svc -n ingress-nginx

# Create DNS A record in DigitalOcean control panel
# Access via: https://your-domain.com/helpdesk
```

---

## Database Access

### PostgreSQL (for developers/DBAs)

```bash
# Port-forward PostgreSQL
kubectl port-forward -n statgate svc/postgres 5432:5432

# Connect from any SQL client:
# Host: localhost
# Port: 5432
# User: postgres
# Password: statgate_k8s_password_prod (from .env)

# Or via psql CLI:
psql -h localhost -U postgres -d postgres
```

### Redis (for cache inspection)

```bash
# Port-forward Redis
kubectl port-forward -n statgate svc/redis 6379:6379

# Connect from redis-cli:
redis-cli -h localhost -p 6379

# Or use online client:
# redis-commander (Docker): docker run -p 8081:8081 rediscommander/redis-commander
```

---

## API Access (Postman/cURL)

### Registry API
```bash
# Get health status
curl http://localhost:9090/ready

# Or via port-forward:
kubectl port-forward -n statgate svc/statgate-registry-api 9090:9090
curl http://localhost:9090/ready
```

### Helpdesk API
```bash
# Get API docs
curl http://localhost:5000/api-docs

# Or via Swagger UI:
# Port-forward then open: http://localhost:5000/api-docs
kubectl port-forward -n statgate svc/statgate-helpdesk-api 5000:5000
```

### PMS API
```bash
# Health check
kubectl port-forward -n statgate svc/statgate-pms-api 8080:8080
curl http://localhost:8080/health
```

---

## Monitoring & Logs

### View Pod Logs
```bash
# View logs from a service
kubectl logs -n statgate deployment/statgate-helpdesk-api -f

# View logs from specific pod
kubectl logs -n statgate statgate-helpdesk-ui-7dbb57c6b-6mcm2 -f

# View last 100 lines
kubectl logs -n statgate deployment/statchat-backend --tail=100
```

### Monitor Pod Status
```bash
# Watch pods in real-time
kubectl get pods -n statgate -w

# Get detailed pod info
kubectl describe pod -n statgate statgate-helpdesk-api-84fcdf59bc-dxlv6
```

### Check Pod Resource Usage
```bash
# CPU and memory usage
kubectl top pods -n statgate

# Node usage
kubectl top nodes
```

---

## Service Discovery (Within Cluster)

From inside a pod, services are accessible via DNS:

```bash
# From inside any pod:
curl http://statgate-helpdesk-api:5000
curl http://statchat-backend:4000
curl http://statgate-pms-api:8080
curl http://postgres:5432  # PostgreSQL
curl http://redis:6379    # Redis
```

---

## Architecture Overview

```
┌─ Kubernetes Cluster ─────────────────────────────────┐
│                                                       │
│  ┌─ Persistent Services ────────────────────────┐   │
│  │ • PostgreSQL (postgres:5432)                 │   │
│  │ • Redis (redis:6379)                         │   │
│  └──────────────────────────────────────────────┘   │
│                                                       │
│  ┌─ Microservices (2 replicas each) ───────────┐   │
│  │ APIs (Backend):                              │   │
│  │ • Helpdesk API (statgate-helpdesk-api:5000) │   │
│  │ • PMS API (statgate-pms-api:8080)            │   │
│  │ • RMS API (statgate-rms-api:8080)            │   │
│  │ • Governance API (statgate-governance-api)   │   │
│  │ • StaChat Backend (statchat-backend:4000)    │   │
│  │ • Analytics (statgate-analytics:5000)        │   │
│  │                                              │   │
│  │ UIs (Frontend):                              │   │
│  │ • Helpdesk UI (statgate-helpdesk-ui:3005)   │   │
│  │ • PMS UI (statgate-pms-ui:3010)              │   │
│  │ • RMS UI (statgate-rms-ui:3011)              │   │
│  │ • Governance UI (statgate-governance-ui)     │   │
│  │ • StaChat Frontend (statchat-frontend:3009)  │   │
│  │ • Registry UI (statgate-registry-ui:3007)    │   │
│  └──────────────────────────────────────────────┘   │
│                                                       │
│  ┌─ Optional: Ingress (NGINX) ──────────────────┐   │
│  │ statgate.example.com/helpdesk                │   │
│  │ statgate.example.com/governance              │   │
│  │ statgate.example.com/statchat                │   │
│  └──────────────────────────────────────────────┘   │
│                                                       │
└───────────────────────────────────────────────────────┘
```

---

## Troubleshooting Connection Issues

### Port-forward not working?
```bash
# Make sure pod is running
kubectl get pods -n statgate statgate-helpdesk-ui

# Check pod logs
kubectl logs -n statgate statgate-helpdesk-ui-7dbb57c6b-6mcm2

# Restart port-forward in new terminal
kubectl port-forward -n statgate svc/statgate-helpdesk-ui 3005:3005
```

### Service unreachable?
```bash
# Check if service exists
kubectl get svc -n statgate

# Check service endpoints
kubectl get endpoints -n statgate statgate-helpdesk-ui

# Test connectivity from within cluster
kubectl exec -n statgate statgate-helpdesk-api-84fcdf59bc-dxlv6 -- curl http://statgate-helpdesk-ui:3005
```

### Database connection fails?
```bash
# Verify PostgreSQL is running
kubectl get pods -n statgate postgres-0

# Check credentials in secrets
kubectl get secret -n statgate statgate-secrets -o yaml

# Test database
kubectl exec -n statgate postgres-0 -- psql -U postgres -c "SELECT version();"
```

---

## Quick Reference

| Service | Port | URL | Purpose |
|---------|------|-----|---------|
| Helpdesk UI | 3005 | http://localhost:3005 | Operations support interface |
| Helpdesk API | 5000 | http://localhost:5000/api-docs | Support API docs |
| StaChat | 3009 | http://localhost:3009 | Real-time collaboration |
| PMS UI | 3010 | http://localhost:3010 | Projects management |
| RMS UI | 3011 | http://localhost:3011 | Research management |
| Governance | 3012 | http://localhost:3012 | Institutional governance |
| Analytics | 5000 | http://localhost:5000 | Data analytics |
| Registry UI | 3007 | http://localhost:3007 | Field operations registry |
| PostgreSQL | 5432 | localhost:5432 | Database |
| Redis | 6379 | localhost:6379 | Cache/event bus |

---

## Start Using Now!

1. **Open PowerShell/Terminal**
2. **Run one of these:**
   ```bash
   kubectl port-forward -n statgate svc/statgate-helpdesk-ui 3005:3005
   ```
3. **Open Browser:** http://localhost:3005
4. **Done!** You're now accessing StatGate
