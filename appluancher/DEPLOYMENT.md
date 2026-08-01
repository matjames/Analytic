# StatGate Uganda Limited - Deployment Guide

Complete deployment instructions for StatGate Uganda Limited Application Launcher.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Local Development Deployment](#local-development-deployment)
3. [Docker Deployment](#docker-deployment)
4. [Kubernetes Deployment](#kubernetes-deployment)
5. [Cloud Platform Deployment](#cloud-platform-deployment)
6. [SSL/TLS Configuration](#ssltls-configuration)
7. [Monitoring & Logging](#monitoring--logging)
8. [Backup & Recovery](#backup--recovery)
9. [Troubleshooting](#troubleshooting)

---

## Prerequisites

### System Requirements

- **OS**: Ubuntu 20.04 LTS or CentOS 8+
- **CPU**: 2+ cores
- **RAM**: 4GB minimum, 8GB recommended
- **Storage**: 20GB available space
- **Network**: Stable internet connection, ports 80/443 accessible

### Required Software

- Docker 20.10+
- Docker Compose 2.0+
- Node.js 18.x
- npm 9.x or yarn 3.x

### Installation (Ubuntu 20.04)

```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo usermod -aG docker $USER

# Install Docker Compose
sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose

# Install Node.js
curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash -
sudo apt-get install -y nodejs

# Verify installations
docker --version
docker-compose --version
node --version
npm --version
```

---

## Local Development Deployment

### Quick Start

```bash
# 1. Clone repository
git clone <repository-url>
cd statgate-app-launcher

# 2. Install dependencies
npm install

# 3. Configure environment
cp .env.example .env.local
# Edit .env.local with your settings

# 4. Start development server
npm run dev

# 5. Access application
# Open http://localhost:3000 in your browser
```

### Development Server

The application will run in development mode with:
- Hot module reloading
- Mock data enabled
- Detailed error messages
- Source maps for debugging

### Build for Production

```bash
# Build application
npm run build

# Start production server
npm start

# Application available at http://localhost:3000
```

---

## Docker Deployment

### Single Container Deployment

#### Build Image

```bash
# Build image with tag
docker build -t statgate-launcher:1.0.0 .

# Tag for registry (if using container registry)
docker tag statgate-launcher:1.0.0 registry.example.com/statgate-launcher:1.0.0
```

#### Run Container

```bash
# Run container with environment variables
docker run -d \
  --name statgate-launcher \
  -p 3000:3000 \
  -e NODE_ENV=production \
  -e NEXT_PUBLIC_APP_NAME="StatGate Uganda Limited" \
  -e NEXT_PUBLIC_API_BASE_URL="http://localhost:3000" \
  statgate-launcher:1.0.0

# View logs
docker logs -f statgate-launcher

# Check container status
docker ps | grep statgate-launcher
```

#### Container Networking

```bash
# Create custom network
docker network create statgate-network

# Run container on custom network
docker run -d \
  --name statgate-launcher \
  --network statgate-network \
  -p 3000:3000 \
  statgate-launcher:1.0.0

# Container can communicate with other containers on network
# Example: curl http://launcher:3000 from other containers
```

### Multi-Container Deployment (Docker Compose)

#### Start Services

```bash
# Navigate to project root
cd /path/to/statgate-app-launcher

# Start all services in detached mode
docker-compose up -d

# View service status
docker-compose ps

# View logs (all services)
docker-compose logs -f

# View logs (specific service)
docker-compose logs -f launcher
```

#### Service Configuration

Edit `docker-compose.yml` to customize:

```yaml
services:
  launcher:
    environment:
      - NODE_ENV=production
      - NEXT_PUBLIC_API_BASE_URL=https://statgate.ug
    ports:
      - "3000:3000"  # Change port if needed
    volumes:
      - ./data:/app/data  # Persistent storage
    restart: unless-stopped
```

#### Scaling Services

```bash
# Scale a service to multiple replicas
docker-compose up -d --scale launcher=3

# Note: Manual port mapping required for multiple replicas
# Use load balancer (Nginx, HAProxy) for routing
```

#### Environment Configuration

Create `.env` file in project root:

```bash
# .env
COMPOSE_PROJECT_NAME=statgate
NODE_ENV=production
LAUNCHER_PORT=3000
NGINX_PORT=80
```

#### Stopping Services

```bash
# Stop all services (keeps containers)
docker-compose stop

# Start stopped services
docker-compose start

# Remove containers
docker-compose down

# Remove containers and volumes
docker-compose down -v
```

---

## Kubernetes Deployment

### Prerequisites

- Kubernetes cluster 1.20+
- kubectl configured with cluster access
- Docker images pushed to container registry

### Create Kubernetes Manifests

#### 1. Namespace

```yaml
# k8s/namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: statgate
  labels:
    app: statgate-launcher
```

#### 2. ConfigMap

```yaml
# k8s/configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: launcher-config
  namespace: statgate
data:
  NEXT_PUBLIC_APP_NAME: "StatGate Uganda Limited"
  NEXT_PUBLIC_API_BASE_URL: "https://statgate.ug"
  NEXT_PUBLIC_FEATURE_RBAC: "true"
  NODE_ENV: "production"
```

#### 3. Deployment

```yaml
# k8s/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: launcher
  namespace: statgate
  labels:
    app: launcher
spec:
  replicas: 3
  selector:
    matchLabels:
      app: launcher
  template:
    metadata:
      labels:
        app: launcher
    spec:
      containers:
      - name: launcher
        image: registry.example.com/statgate-launcher:1.0.0
        imagePullPolicy: IfNotPresent
        ports:
        - containerPort: 3000
          name: http
        envFrom:
        - configMapRef:
            name: launcher-config
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 3000
          initialDelaySeconds: 15
          periodSeconds: 20
        readinessProbe:
          httpGet:
            path: /health
            port: 3000
          initialDelaySeconds: 5
          periodSeconds: 10
```

#### 4. Service

```yaml
# k8s/service.yaml
apiVersion: v1
kind: Service
metadata:
  name: launcher
  namespace: statgate
  labels:
    app: launcher
spec:
  type: ClusterIP
  selector:
    app: launcher
  ports:
  - protocol: TCP
    port: 80
    targetPort: 3000
    name: http
```

#### 5. Ingress

```yaml
# k8s/ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: launcher-ingress
  namespace: statgate
  annotations:
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
spec:
  ingressClassName: nginx
  tls:
  - hosts:
    - statgate.ug
    secretName: launcher-tls
  rules:
  - host: statgate.ug
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: launcher
            port:
              number: 80
```

### Deploy to Kubernetes

```bash
# Create namespace
kubectl apply -f k8s/namespace.yaml

# Apply configuration
kubectl apply -f k8s/configmap.yaml

# Deploy application
kubectl apply -f k8s/deployment.yaml

# Create service
kubectl apply -f k8s/service.yaml

# Create ingress
kubectl apply -f k8s/ingress.yaml

# Verify deployment
kubectl get pods -n statgate
kubectl get svc -n statgate
kubectl get ingress -n statgate

# View logs
kubectl logs -n statgate deployment/launcher -f

# Scale deployment
kubectl scale deployment launcher -n statgate --replicas=5

# Update deployment
kubectl set image deployment/launcher \
  launcher=registry.example.com/statgate-launcher:1.1.0 \
  -n statgate
```

---

## Cloud Platform Deployment

### AWS Elastic Container Service (ECS)

#### Push Image to ECR

```bash
# Authenticate with ECR
aws ecr get-login-password --region us-east-1 | \
  docker login --username AWS --password-stdin 123456789.dkr.ecr.us-east-1.amazonaws.com

# Tag image
docker tag statgate-launcher:1.0.0 \
  123456789.dkr.ecr.us-east-1.amazonaws.com/statgate-launcher:1.0.0

# Push image
docker push 123456789.dkr.ecr.us-east-1.amazonaws.com/statgate-launcher:1.0.0
```

#### ECS Task Definition

```json
{
  "family": "statgate-launcher",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "512",
  "memory": "1024",
  "containerDefinitions": [
    {
      "name": "launcher",
      "image": "123456789.dkr.ecr.us-east-1.amazonaws.com/statgate-launcher:1.0.0",
      "portMappings": [
        {
          "containerPort": 3000,
          "hostPort": 3000,
          "protocol": "tcp"
        }
      ],
      "environment": [
        {
          "name": "NODE_ENV",
          "value": "production"
        },
        {
          "name": "NEXT_PUBLIC_APP_NAME",
          "value": "StatGate Uganda Limited"
        }
      ],
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/statgate-launcher",
          "awslogs-region": "us-east-1",
          "awslogs-stream-prefix": "ecs"
        }
      }
    }
  ]
}
```

#### Deploy with CLI

```bash
# Register task definition
aws ecs register-task-definition \
  --cli-input-json file://task-definition.json

# Create service
aws ecs create-service \
  --cluster statgate \
  --service-name launcher \
  --task-definition statgate-launcher:1 \
  --desired-count 3 \
  --launch-type FARGATE \
  --network-configuration "awsvpcConfiguration={subnets=[subnet-xxxxx],securityGroups=[sg-xxxxx],assignPublicIp=ENABLED}"
```

### Google Cloud Run

```bash
# Build and deploy to Cloud Run
gcloud run deploy statgate-launcher \
  --source . \
  --region us-central1 \
  --platform managed \
  --memory 512Mi \
  --cpu 1 \
  --allow-unauthenticated \
  --set-env-vars NODE_ENV=production

# Set up custom domain
gcloud run domain-mappings create \
  --service=statgate-launcher \
  --domain=statgate.ug \
  --region=us-central1
```

### Heroku

```bash
# Login to Heroku
heroku login

# Create Heroku app
heroku create statgate-launcher

# Set environment variables
heroku config:set NODE_ENV=production -a statgate-launcher

# Deploy using container
heroku container:login
docker tag statgate-launcher:1.0.0 registry.heroku.com/statgate-launcher/web:latest
docker push registry.heroku.com/statgate-launcher/web:latest
heroku container:release web -a statgate-launcher

# View logs
heroku logs --tail -a statgate-launcher
```

---

## SSL/TLS Configuration

### Self-Signed Certificates (Development)

```bash
# Generate private key
openssl genrsa -out docker/nginx/ssl/private-key.pem 2048

# Generate certificate
openssl req -new -x509 -key docker/nginx/ssl/private-key.pem \
  -out docker/nginx/ssl/certificate.pem -days 365 \
  -subj "/C=UG/ST=Kampala/L=Kampala/O=StatGate/CN=localhost"
```

### Let's Encrypt (Production)

#### Using Certbot

```bash
# Install Certbot
sudo apt-get install certbot python3-certbot-nginx -y

# Obtain certificate
sudo certbot certonly --standalone -d statgate.ug -d www.statgate.ug

# Copy certificates to docker volume
sudo cp /etc/letsencrypt/live/statgate.ug/fullchain.pem docker/nginx/ssl/certificate.pem
sudo cp /etc/letsencrypt/live/statgate.ug/privkey.pem docker/nginx/ssl/private-key.pem

# Auto-renew certificates
sudo systemctl enable certbot.timer
sudo systemctl start certbot.timer
```

#### Using Docker (docker-compose)

```yaml
services:
  certbot:
    image: certbot/certbot:latest
    volumes:
      - ./docker/nginx/ssl:/etc/letsencrypt
    command: certonly --standalone -d statgate.ug --agree-tos -m admin@statgate.ug
```

### Nginx SSL Configuration

Uncomment and configure in `docker/nginx/conf.d/default.conf`:

```nginx
server {
  listen 443 ssl http2;
  server_name statgate.ug www.statgate.ug;

  ssl_certificate /etc/nginx/ssl/certificate.pem;
  ssl_certificate_key /etc/nginx/ssl/private-key.pem;
  
  ssl_protocols TLSv1.2 TLSv1.3;
  ssl_ciphers HIGH:!aNULL:!MD5;
  ssl_prefer_server_ciphers on;
  
  # ... rest of configuration
}
```

---

## Monitoring & Logging

### Docker Logging

```bash
# View container logs
docker logs statgate-launcher

# Follow logs in real-time
docker logs -f statgate-launcher

# View last 100 lines
docker logs --tail 100 statgate-launcher

# View logs with timestamps
docker logs -t statgate-launcher
```

### Docker Compose Logging

```bash
# View all service logs
docker-compose logs

# Follow logs
docker-compose logs -f

# View specific service
docker-compose logs -f launcher

# View logs since specific time
docker-compose logs --since 2024-01-15T00:00:00Z
```

### Health Checks

```bash
# Check application health
curl http://localhost:3000/health

# Check through Nginx
curl http://localhost:80/health

# Docker health status
docker ps --format "table {{.Names}}\t{{.Status}}"
```

### Application Monitoring

Monitor key metrics:

```bash
# CPU and Memory Usage
docker stats statgate-launcher

# Network I/O
docker stats --no-stream

# Disk Usage
docker system df
```

### Structured Logging

Configure application logging in `src/utils/logger.ts`:

```typescript
// Example logging setup
const logger = {
  info: (message: string, data?: any) => console.log(JSON.stringify({ level: 'info', message, data })),
  error: (message: string, error?: any) => console.error(JSON.stringify({ level: 'error', message, error })),
  warn: (message: string, data?: any) => console.warn(JSON.stringify({ level: 'warn', message, data })),
};
```

### Log Aggregation

For production, use:
- **ELK Stack**: Elasticsearch, Logstash, Kibana
- **Splunk**: Commercial log aggregation
- **CloudWatch**: AWS native logging
- **Stackdriver**: Google Cloud logging
- **Azure Monitor**: Azure native monitoring

---

## Backup & Recovery

### Database Backups

```bash
# Backup container volumes
docker run --rm \
  -v statgate-launcher_data:/data \
  -v $(pwd)/backups:/backup \
  ubuntu tar czf /backup/backup-$(date +%Y%m%d).tar.gz -C /data .

# Restore from backup
docker run --rm \
  -v statgate-launcher_data:/data \
  -v $(pwd)/backups:/backup \
  ubuntu tar xzf /backup/backup-20240115.tar.gz -C /data
```

### Application Backups

```bash
# Backup entire application
tar -czf statgate-launcher-backup-$(date +%Y%m%d).tar.gz \
  --exclude=node_modules \
  --exclude=.next \
  --exclude=.git \
  .

# Restore from backup
tar -xzf statgate-launcher-backup-20240115.tar.gz
```

### Automated Backups (Cron)

```bash
# Add to crontab
crontab -e

# Daily backup at 2 AM
0 2 * * * docker run --rm -v statgate-launcher_data:/data -v /backup:/backup ubuntu tar czf /backup/backup-$(date +\%Y\%m\%d).tar.gz -C /data .
```

---

## Troubleshooting

### Container Won't Start

```bash
# Check logs for errors
docker logs statgate-launcher

# Inspect container
docker inspect statgate-launcher

# Check resource limits
docker stats statgate-launcher

# Try rebuilding
docker build --no-cache -t statgate-launcher:1.0.0 .
```

### Port Already in Use

```bash
# Find process using port 3000
sudo lsof -i :3000

# Kill process
kill -9 <PID>

# Or change port in docker-compose.yml
ports:
  - "3001:3000"  # Use port 3001 instead
```

### Out of Disk Space

```bash
# Check disk usage
docker system df

# Remove unused images
docker image prune -a

# Remove unused containers
docker container prune

# Remove unused volumes
docker volume prune

# Full cleanup
docker system prune -a --volumes
```

### Service Communication Issues

```bash
# Check network connectivity
docker network inspect statgate-network

# Test DNS resolution
docker exec statgate-launcher ping launcher

# Check exposed ports
docker port statgate-launcher
```

### Performance Issues

```bash
# Monitor real-time stats
docker stats --no-stream

# Increase resource limits
docker update --cpus="1" --memory="2g" statgate-launcher

# Check application logs for bottlenecks
docker logs -f statgate-launcher | grep error
```

---

## Next Steps

1. **Configure SSL/TLS certificates** for production
2. **Set up monitoring and alerting** for uptime
3. **Configure backup strategy** for data protection
4. **Implement CI/CD pipeline** for automated deployments
5. **Set up load balancing** for high availability
6. **Configure auto-scaling** based on demand

For additional support: admin@statgate.ug
