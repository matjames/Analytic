# Kubernetes Deployment Guide for StatGate Analytics Platform

## Overview
This directory contains complete Kubernetes manifests to deploy the StatGate analytics system on any Kubernetes cluster.

## Architecture

**Infrastructure:**
- PostgreSQL StatefulSet (primary database, 1 replica)
- Redis StatefulSet (event bus & caching, 1 replica)

**Microservices (12+ services with 2 replicas each):**
- **Registry**: Field Operations API (Go) + UI (React)
- **StaChat**: Real-time collaboration backend (Go) + frontend (React)
- **Helpdesk**: Operations support API (Node.js) + UI (React)
- **PMS**: Projects Management System API (Go) + UI (React)
- **RMS**: Research Management System API (Go) + UI (React)
- **Governance**: Institutional governance API (Go) + UI (React)
- **Analytics**: Data analytics engine (Python Flask)

**External Access:**
- Ingress (NGINX) with TLS support for HTTP routing
- Services (ClusterIP for internal, NodePort for external)

## Files

| File | Purpose |
|------|---------|
| `00-namespace.yaml` | Namespace, ConfigMap (env vars), Secrets (passwords) |
| `01-postgres.yaml` | PostgreSQL StatefulSet + PersistentVolumeClaim |
| `02-redis.yaml` | Redis StatefulSet + PersistentVolumeClaim |
| `03-registry.yaml` | Registry API (Go) + UI (React) Deployments + Services |
| `04-statchat.yaml` | StaChat Backend (Go) + Frontend (React) Deployments + Services |
| `05-helpdesk.yaml` | Helpdesk API (Node.js) + UI (React) Deployments + Services |
| `06-pms-rms.yaml` | PMS/RMS APIs (Go) + UIs (React) Deployments + Services |
| `07-governance-analytics.yaml` | Governance API (Go) + UI, Analytics (Python) Deployments + Services |
| `08-ingress.yaml` | NGINX Ingress with path-based routing |
| `deploy.sh` | Automated deployment script |

## Prerequisites

1. **Kubernetes Cluster** (1.20+)
   - Docker Desktop (local): `Settings > Kubernetes > Enable Kubernetes`
   - Cloud: AWS EKS, Azure AKS, GCP GKE, DigitalOcean, etc.

2. **kubectl** CLI tool
   ```bash
   kubectl version --client
   ```

3. **StorageClass** (for PersistentVolumes)
   - Local clusters: included by default
   - Cloud: auto-provisioned by cloud provider

4. **Docker Images** (pre-built and available)
   - All images in local Docker daemon or private registry
   - Example: `analytic-statgate-registry-api:latest`

5. **(Optional) NGINX Ingress Controller** (for external access)
   ```bash
   # Install NGINX Ingress
   kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.8.0/deploy/static/provider/cloud/deploy.yaml
   ```

6. **(Optional) cert-manager** (for HTTPS/TLS)
   ```bash
   # Install cert-manager
   kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.12.0/cert-manager.yaml
   ```

## Quick Start

### Step 1: Verify Kubernetes is Running
```bash
kubectl cluster-info
kubectl get nodes
```

### Step 2: Deploy All Services
```bash
# Make script executable
chmod +x deploy.sh

# Run automated deployment
./deploy.sh
```

Or manually apply in order:
```bash
kubectl apply -f 00-namespace.yaml
kubectl apply -f 01-postgres.yaml
kubectl wait --for=condition=ready pod -l app=postgres -n statgate --timeout=300s
kubectl apply -f 02-redis.yaml
kubectl wait --for=condition=ready pod -l app=redis -n statgate --timeout=300s
kubectl apply -f 03-registry.yaml
kubectl apply -f 04-statchat.yaml
kubectl apply -f 05-helpdesk.yaml
kubectl apply -f 06-pms-rms.yaml
kubectl apply -f 07-governance-analytics.yaml
kubectl apply -f 08-ingress.yaml
```

### Step 3: Monitor Rollout
```bash
# Watch all pods
kubectl get pods -n statgate -w

# Check deployment status
kubectl rollout status deployment/statgate-registry-api -n statgate
kubectl rollout status deployment/statchat-backend -n statgate
```

### Step 4: Access Services

**Port-Forward (development/testing):**
```bash
# Registry UI
kubectl port-forward -n statgate svc/statgate-registry-ui 3007:3007
# Access: http://localhost:3007

# Registry API
kubectl port-forward -n statgate svc/statgate-registry-api 9090:9090
# Access: http://localhost:9090/api

# Helpdesk UI
kubectl port-forward -n statgate svc/statgate-helpdesk-ui 3005:3005
# Access: http://localhost:3005
```

**Via Ingress (production):**
1. Configure DNS or /etc/hosts:
   ```bash
   echo "127.0.0.1 statgate.example.com" >> /etc/hosts
   ```

2. Get Ingress IP:
   ```bash
   kubectl get ingress -n statgate
   ```

3. Access services:
   ```
   http://statgate.example.com/registry
   http://statgate.example.com/registry-api
   http://statgate.example.com/helpdesk
   ```

## Scaling & Management

### Scale a Deployment
```bash
# Scale Registry API to 5 replicas
kubectl scale deployment/statgate-registry-api -n statgate --replicas=5

# Verify
kubectl get deployment -n statgate
```

### Update an Image
```bash
# Update Registry API with new image
kubectl set image deployment/statgate-registry-api \
  registry-api=analytic-statgate-registry-api:v2.0 \
  -n statgate

# Monitor rollout
kubectl rollout status deployment/statgate-registry-api -n statgate
```

### View Logs
```bash
# All pods of a service
kubectl logs -n statgate -l app=statgate-registry-api --tail=100

# Specific pod
kubectl logs -n statgate pod/statgate-registry-api-xyz123 -f

# Previous crashed pod
kubectl logs -n statgate pod/statgate-registry-api-xyz123 --previous
```

### Delete a Service
```bash
# Delete Registry deployment
kubectl delete deployment/statgate-registry-api -n statgate

# Delete entire namespace (all services)
kubectl delete namespace statgate
```

## Environment Configuration

Modify `00-namespace.yaml` to change:
- Database names, users, passwords (Secrets section)
- API endpoints, ports (ConfigMap section)
- Service URLs

Then reapply:
```bash
kubectl apply -f 00-namespace.yaml
kubectl rollout restart deployment -n statgate
```

## Resource Requests & Limits

All services have requests/limits defined:
- **Requests**: Minimum guaranteed resources (used for scheduling)
- **Limits**: Maximum allowed (pod killed if exceeded)

Adjust in each manifest based on cluster capacity:
```yaml
resources:
  requests:
    memory: "256Mi"
    cpu: "250m"
  limits:
    memory: "512Mi"
    cpu: "500m"
```

## Persistence

- **PostgreSQL**: Uses PersistentVolumeClaim (20Gi)
- **Redis**: Uses PersistentVolumeClaim (10Gi)
- **Stateless Services**: No persistent storage (scale horizontally)

## High Availability Setup

For production, scale up:
```bash
# 3+ replicas per service
kubectl scale deployment -n statgate --replicas=3 --all

# 3+ PostgreSQL/Redis replicas
# (requires StatefulSet updates in manifests)
```

## Troubleshooting

### Pods stuck in Pending
```bash
# Check events
kubectl describe pod -n statgate <pod-name>

# Check resources
kubectl top nodes
kubectl describe node <node-name>

# Check StorageClass
kubectl get storageclass
```

### Pods crashing (CrashLoopBackOff)
```bash
# View logs
kubectl logs -n statgate pod/<pod-name>

# Check previous logs
kubectl logs -n statgate pod/<pod-name> --previous

# Describe pod for events
kubectl describe pod -n statgate <pod-name>
```

### Services not communicating
```bash
# Test DNS resolution from pod
kubectl exec -n statgate pod/<pod-name> -- nslookup statgate-registry-api

# Test connectivity
kubectl exec -n statgate pod/<pod-name> -- wget -O- http://statgate-registry-api:9090/ready
```

### Ingress not working
```bash
# Check Ingress Controller
kubectl get pods -n ingress-nginx

# Check Ingress rules
kubectl get ingress -n statgate -o yaml
kubectl describe ingress -n statgate statgate-ingress

# Test backend service directly
kubectl port-forward -n statgate svc/statgate-registry-ui 3007:3007
```

## Cloud Deployment Examples

### AWS EKS
```bash
# Create cluster
eksctl create cluster --name statgate --region us-east-1 --nodes 3

# Deploy
kubectl apply -f 00-namespace.yaml
./deploy.sh
```

### Azure AKS
```bash
# Create cluster
az aks create --resource-group myGroup --name statgate --node-count 3

# Deploy
az aks get-credentials --resource-group myGroup --name statgate
./deploy.sh
```

### DigitalOcean
```bash
# Create cluster via web UI or doctl CLI
doctl kubernetes cluster create statgate

# Deploy
doctl kubernetes cluster kubeconfig save statgate
./deploy.sh
```

## Cleanup

```bash
# Delete namespace (all resources)
kubectl delete namespace statgate

# Verify deletion
kubectl get namespace statgate  # should fail
```

## Support

For issues or questions:
1. Check logs: `kubectl logs -n statgate deployment/<service>`
2. Describe resources: `kubectl describe pod -n statgate <pod-name>`
3. Check StatGate documentation in `/docs`
