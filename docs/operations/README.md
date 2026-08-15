# StatGate Operations & Observability

## Health, Liveness & Readiness Probes
- **`/health`**: Basic service uptime and component connectivity summary.
- **`/live`**: Kubernetes/Container liveness probe (returns 200 while the process is active).
- **`/ready`**: Kubernetes/Load balancer readiness probe (verifies database pool and Redis connectivity before admitting ingress traffic).
- **`/metrics`**: Real-time atomic metrics snapshot covering request volume, error rates, memory allocation, active goroutines, queue depths, and DB pool stats.

## Deployment & Configuration
- Strict separation between development (`STATGATE_ENV=development`) and production (`STATGATE_ENV=production`).
- Production deployments must supply `STATGATE_REGISTRY_JWT_SECRET` and `STATGATE_INTERNAL_API_KEY`.
