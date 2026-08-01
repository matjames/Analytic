# StatGate Uganda Limited - Application Launcher

Central hub and home portal for StatGate Uganda Limited's statistical tools, databases, and data-driven applications.

## Features

✨ **Core Features:**
- **Central Application Hub**: Navigate all StatGate Uganda Limited applications from one unified dashboard
- **App Launcher Component**: Quick-access 3x3 grid launcher in the header for instant cross-navigation
- **Role-Based Access Control (RBAC)**: Dynamic application visibility based on user roles and permissions
- **Responsive Design**: Optimized for desktop, tablet, and mobile devices
- **System Status Monitoring**: Real-time health checks and status indicators for all applications
- **Professional UI/UX**: Clean, modern interface tailored for statistical operations
- **Error Boundaries**: Isolated error handling to prevent cascading failures
- **Multi-Container Deployment**: Docker & Docker Compose architecture for scalable deployments

## Tech Stack

### Frontend
- **Framework**: Next.js 14 with React 18
- **Styling**: Tailwind CSS 3
- **Type Safety**: TypeScript 5
- **Icons**: Lucide React
- **HTTP Client**: Axios

### Backend/Infrastructure
- **Runtime**: Node.js 18
- **Containerization**: Docker & Docker Compose
- **Reverse Proxy**: Nginx (load balancing & routing)
- **Database**: Ready for integration (PostgreSQL, MongoDB, etc.)

## Project Structure

```
statgate-app-launcher/
├── src/
│   ├── components/          # React components
│   │   ├── Header.tsx       # Navigation header with app launcher
│   │   ├── AppLauncher.tsx  # 3x3 grid app launcher modal
│   │   ├── AppCard.tsx      # Individual application card
│   │   ├── Dashboard.tsx    # Main dashboard with app directory
│   │   └── ErrorBoundary.tsx # Error handling wrapper
│   ├── pages/               # Next.js pages
│   │   ├── _app.tsx         # App wrapper with providers
│   │   ├── _document.tsx    # HTML document structure
│   │   └── index.tsx        # Home page
│   ├── context/             # React Context providers
│   │   ├── AuthContext.tsx  # Authentication state & RBAC
│   │   └── LauncherContext.tsx # Applications state management
│   ├── hooks/               # Custom React hooks
│   ├── utils/               # Utility functions
│   │   ├── mockData.ts      # Mock data & test fixtures
│   │   └── helpers.ts       # Helper functions
│   ├── types/               # TypeScript type definitions
│   └── styles/              # CSS stylesheets
├── public/                  # Static assets
│   └── icons/               # SVG icons
├── docker/
│   └── nginx/               # Nginx configuration
├── docker-compose.yml       # Multi-container orchestration
├── Dockerfile               # Application container image
├── tsconfig.json            # TypeScript configuration
├── tailwind.config.js       # Tailwind CSS configuration
├── next.config.js           # Next.js configuration
└── package.json             # Dependencies & scripts
```

## Getting Started

### Prerequisites
- Node.js 18+ 
- npm or yarn
- Docker & Docker Compose (for containerized deployment)

### Local Development

1. **Install dependencies**:
   ```bash
   npm install
   ```

2. **Configure environment**:
   ```bash
   # .env.local is already configured with defaults
   # Modify as needed for your environment
   ```

3. **Start development server**:
   ```bash
   npm run dev
   ```
   
   Application will be available at `http://localhost:3000`

4. **Build for production**:
   ```bash
   npm run build
   npm start
   ```

## Docker Deployment

### Single Container

Build and run the launcher application:

```bash
# Build image
docker build -t statgate-launcher:latest .

# Run container
docker run -p 3000:3000 -e NODE_ENV=production statgate-launcher:latest
```

### Multi-Container (Docker Compose)

Deploy the complete system with Nginx, launcher, and application placeholders:

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f launcher

# Stop services
docker-compose down
```

**Services:**
- **Launcher** (port 3000): Main application hub
- **Nginx** (port 80/443): Reverse proxy & load balancer
- **Stats Suite** (port 3001): Analytics application placeholder
- **Warehouse** (port 3002): Database application placeholder
- **Reports** (port 3003): Reporting application placeholder
- **Admin** (port 3004): Administration application placeholder
- **Integration** (port 3005): Integration hub placeholder

## Architecture

### Component Hierarchy

```
_app.tsx (Providers)
├── AuthProvider (Auth state & RBAC)
├── LauncherProvider (Apps state)
└── ErrorBoundary (Error handling)
    └── Header (Navigation & App Launcher)
        ├── App Launcher Modal
        └── User Menu
    └── Dashboard (Main content)
        └── AppCard[] (Application tiles)
```

### Data Flow

1. **Authentication**: User logs in → AuthContext updates user state & permissions
2. **App Loading**: LauncherContext fetches applications on mount
3. **RBAC Filtering**: Applications filtered based on user role
4. **App Launch**: User clicks app → Analytics tracked → Navigation triggered
5. **Error Handling**: Component errors caught by ErrorBoundary → UI fallback shown

## Configuration

### Environment Variables

```env
# Application
NEXT_PUBLIC_APP_NAME=StatGate Uganda Limited
NEXT_PUBLIC_API_BASE_URL=http://localhost:3000

# Features
NEXT_PUBLIC_AUTH_ENABLED=true
NEXT_PUBLIC_FEATURE_RBAC=true
NEXT_PUBLIC_FEATURE_SEARCH=true

# Mock Mode (for development)
NEXT_PUBLIC_MOCK_MODE=true
```

### User Roles & Permissions

Defined in `src/utils/mockData.ts`:

| Role | Access | Permissions |
|------|--------|-------------|
| **Admin** | All apps | Full system access, user management |
| **Manager** | Analytics, Database, Reports, Integration | App management, reporting |
| **Analyst** | Analytics, Database, Reports, Quality | Analytics, data export |
| **Viewer** | Reports, Data View | View-only access |
| **Guest** | Public apps | Limited public access |

### Application Categories

- **Analytics**: Statistical analysis and modeling
- **Database**: Data storage and retrieval
- **Reporting**: Report creation and publishing
- **Administration**: System and user management
- **Tools**: Utility and support tools
- **Integration**: External integrations and APIs

## Performance Optimizations

- ⚡ **Code Splitting**: Automatic route-based code splitting
- 🎯 **Image Optimization**: Next.js image optimization for public assets
- 💾 **Caching**: Browser caching headers configured for static assets
- 🗜️ **Compression**: Gzip compression enabled in Nginx
- 🚀 **SSG/ISR**: Static generation where applicable

## Security Features

- 🔐 **RBAC**: Role-based application access control
- 🛡️ **Security Headers**: X-Frame-Options, X-Content-Type-Options, etc.
- 🔒 **HTTPS Ready**: SSL/TLS configuration available in Nginx
- 🚫 **XSS Protection**: React's built-in XSS prevention
- 📊 **CSRF Protection**: Can be added via middleware

## Monitoring & Health Checks

All containers include health checks:

```bash
# Check launcher health
curl http://localhost:3000/health

# Check Nginx health
curl http://localhost:80/health

# View Docker health status
docker ps --format "table {{.Names}}\t{{.Status}}"
```

## API Endpoints

### Analytics (Mock)

```
POST /api/analytics/app-launch
  - Track application launches
  - Body: { appId, appName, timestamp, userAgent, metadata }
```

### Applications (Mock)

```
GET /api/applications
  - Fetch all available applications
  - Returns: Application[]
```

### Health

```
GET /health
- Application health status
- Returns: { status: 'healthy' }
```

## Development

### Running Tests

```bash
npm run test
npm run test:watch
```

### Linting

```bash
npm run lint
```

### Building for Production

```bash
npm run build
npm start
```

## Troubleshooting

### Applications not loading

1. Check environment variables in `.env.local`
2. Verify `NEXT_PUBLIC_MOCK_MODE=true` for mock data
3. Check browser console for errors
4. Verify component error boundaries are working

### Docker containers not starting

```bash
# Check logs
docker-compose logs launcher

# Rebuild images
docker-compose build --no-cache

# Full restart
docker-compose down -v
docker-compose up --build
```

### Port conflicts

Modify `.env.local` or `docker-compose.yml` to use different ports:

```yaml
services:
  launcher:
    ports:
      - "8000:3000"  # Change host port from 3000 to 8000
```

## Contributing

### Code Style

- Use TypeScript for type safety
- Follow ESLint configuration
- Use Prettier for formatting
- Write descriptive commit messages

### Creating New Applications

To add a new application to the launcher:

1. Add entry to `MOCK_APPLICATIONS` in `src/utils/mockData.ts`
2. Create Docker container in `docker-compose.yml`
3. Update Nginx configuration in `docker/nginx/conf.d/default.conf`
4. Set appropriate `requiredRoles` for access control

## Deployment Guide

### AWS ECS/Fargate

```bash
# Build and push to ECR
aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin 123456789.dkr.ecr.us-east-1.amazonaws.com
docker build -t statgate-launcher .
docker tag statgate-launcher:latest 123456789.dkr.ecr.us-east-1.amazonaws.com/statgate-launcher:latest
docker push 123456789.dkr.ecr.us-east-1.amazonaws.com/statgate-launcher:latest
```

### Kubernetes

```bash
# Create deployment
kubectl create namespace statgate
kubectl apply -f k8s/deployment.yaml -n statgate
kubectl apply -f k8s/service.yaml -n statgate
```

### Traditional Server (Ubuntu/CentOS)

```bash
# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# Clone repository and start services
git clone <repository-url>
cd statgate-app-launcher
docker-compose up -d
```

## License

Proprietary - StatGate Uganda Limited 2024

## Support & Contact

For issues, feature requests, or support:
- Internal: [IT Support Email]
- Documentation: https://docs.statgate.ug
- Repository: [Internal GitLab/GitHub]

---

**Version**: 1.0.0  
**Last Updated**: August 2024  
**Maintained By**: StatGate Development Team
