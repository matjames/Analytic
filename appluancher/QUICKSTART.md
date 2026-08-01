# Quick Start Guide

Get StatGate Uganda Limited Application Launcher running in minutes.

## Prerequisites

- Node.js 18+ installed
- npm or yarn
- Basic command line knowledge

## 5-Minute Setup

### 1. Install Dependencies

```bash
cd c:\Users\PC\Desktop\appluancher
npm install
```

### 2. Start Development Server

```bash
npm run dev
```

### 3. Open in Browser

Navigate to: **http://localhost:3000**

That's it! 🎉 The application is now running.

## What You'll See

- **Header**: StatGate Uganda branding with app launcher icon (3x3 grid) in top right
- **Dashboard**: Welcome section with system status
- **Application Cards**: 6 sample applications organized by category
  - Analytics: Statistical Analysis Suite, Data Quality Monitor
  - Database: Data Warehouse
  - Reporting: Report Builder
  - Administration: User Administration
  - Integration: Integration Hub

## First Steps

### 1. Click the App Launcher

Click the 3x3 grid icon in the top right corner of the header to see the app launcher modal.

### 2. Search Applications

Type in the search box to filter applications by name or description.

### 3. View App Details

Each application card shows:
- Application icon and name
- Description
- Status (operational, degraded, maintenance, offline)
- Version and last update info
- Tags and category

### 4. Launch an Application

Click the "Launch" button on any app card to navigate to that application.

### 5. User Profile Menu

Click on your profile picture (top right) to:
- View profile
- Access settings (admin only)
- Logout

## Available Roles

The demo includes a logged-in admin user with full access. Mock roles:

| Role | Access | Use For |
|------|--------|---------|
| Admin | All apps | Full system access |
| Manager | Analytics, Database, Reports, Integration | Management functions |
| Analyst | Analytics, Database, Reports, Quality | Data analysis |
| Viewer | Reports, Data | Read-only access |
| Guest | Public apps | Limited access |

## Useful Commands

```bash
# Development server (with hot reload)
npm run dev

# Build for production
npm run build

# Start production server
npm start

# Run linting
npm run lint

# Run tests
npm test

# Run tests in watch mode
npm run test:watch
```

## Environment Configuration

The app uses mock data by default. To change settings:

1. Edit `.env.local`
2. Set `NEXT_PUBLIC_MOCK_MODE=false` to use real API
3. Update `NEXT_PUBLIC_API_BASE_URL` to your API endpoint

## Docker Quick Start

### Run with Docker

```bash
# Build image
docker build -t statgate-launcher .

# Run container
docker run -p 3000:3000 statgate-launcher
```

### Run with Docker Compose

```bash
# Start all services
docker-compose up

# Access at http://localhost
```

## Project Structure

```
src/
├── components/       # React components
├── pages/           # Next.js pages
├── context/         # State management
├── hooks/           # Custom hooks
├── utils/           # Utilities & helpers
├── types/           # TypeScript types
└── styles/          # CSS
```

## Next Steps

1. **Read README.md** for full project documentation
2. **Check DEVELOPMENT.md** for coding guidelines
3. **Review DEPLOYMENT.md** for production deployment
4. **Explore components** in `src/components/`
5. **Modify mock data** in `src/utils/mockData.ts`

## Common Tasks

### Add a New Application

Edit `src/utils/mockData.ts`:

```typescript
{
  id: 'app-007',
  name: 'Your App Name',
  description: 'Description here',
  icon: '🎯',
  url: '/apps/your-app',
  category: 'tools',
  status: AppStatus.OPERATIONAL,
  requiredRoles: [UserRole.ANALYST, UserRole.MANAGER, UserRole.ADMIN],
}
```

### Change User Role

In `src/utils/mockData.ts`, modify `MOCK_USER`:

```typescript
export const MOCK_USER: User = {
  // ... other fields
  role: UserRole.ANALYST,  // Change this
  // ...
};
```

### Update Branding

Edit `.env.local`:
```
NEXT_PUBLIC_APP_NAME=Your Organization Name
```

### Customize Colors

Edit `tailwind.config.js` to change the primary color scheme.

## Troubleshooting

### Port 3000 Already in Use

```bash
# Windows - Find and kill process
netstat -ano | findstr :3000
taskkill /PID <PID> /F

# Or change port in package.json
# Update next dev to next dev -p 3001
```

### Dependencies Not Installing

```bash
# Clear npm cache
npm cache clean --force

# Delete node_modules and package-lock.json
rm -rf node_modules package-lock.json

# Reinstall
npm install
```

### Build Errors

```bash
# Clean build artifacts
rm -rf .next

# Rebuild
npm run build
```

## Learn More

- [Next.js Documentation](https://nextjs.org/docs)
- [React Documentation](https://react.dev)
- [Tailwind CSS](https://tailwindcss.com/docs)
- [TypeScript](https://www.typescriptlang.org/docs)

## Support

For help or questions:
- Check [README.md](README.md) for detailed documentation
- Review [DEVELOPMENT.md](DEVELOPMENT.md) for code guidelines
- Consult [DEPLOYMENT.md](DEPLOYMENT.md) for deployment help

## API Testing

Test the application endpoints:

```bash
# Get all applications
curl http://localhost:3000/api/applications

# Get health status
curl http://localhost:3000/api/health

# Filter by category
curl "http://localhost:3000/api/applications?category=analytics"

# Filter by role
curl "http://localhost:3000/api/applications?role=admin"
```

## Production Checklist

Before deploying to production:

- [ ] Replace mock data with real data source
- [ ] Configure real authentication
- [ ] Set up database
- [ ] Configure HTTPS/SSL
- [ ] Set up monitoring and logging
- [ ] Configure backup strategy
- [ ] Update environment variables
- [ ] Run full test suite
- [ ] Perform load testing
- [ ] Set up CI/CD pipeline

---

**Ready to build?** Start with `npm run dev` and happy coding! 🚀
