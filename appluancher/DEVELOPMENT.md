# StatGate Uganda Limited - Development Guide

Guide for developers working on the StatGate Uganda Limited Application Launcher.

## Table of Contents

1. [Development Setup](#development-setup)
2. [Project Structure](#project-structure)
3. [Code Style & Standards](#code-style--standards)
4. [Component Development](#component-development)
5. [State Management](#state-management)
6. [API Integration](#api-integration)
7. [Testing](#testing)
8. [Debugging](#debugging)
9. [Common Tasks](#common-tasks)
10. [Contribution Workflow](#contribution-workflow)

---

## Development Setup

### Initial Setup

```bash
# 1. Clone repository
git clone <repository-url>
cd statgate-app-launcher

# 2. Install dependencies
npm install

# 3. Create environment file
cp .env.example .env.local

# 4. Start development server
npm run dev

# 5. Open browser
# Navigate to http://localhost:3000
```

### IDE Setup

#### VS Code Recommended Extensions

```json
{
  "recommendations": [
    "ms-vscode.vscode-typescript-next",
    "esbenp.prettier-vscode",
    "dbaeumer.vscode-eslint",
    "bradlc.vscode-tailwindcss",
    "ms-vscode-remote.remote-containers"
  ]
}
```

Install extensions:
```bash
code --install-extension ms-vscode.vscode-typescript-next
code --install-extension esbenp.prettier-vscode
code --install-extension dbaeumer.vscode-eslint
code --install-extension bradlc.vscode-tailwindcss
```

#### VS Code Settings

Create `.vscode/settings.json`:

```json
{
  "editor.formatOnSave": true,
  "editor.defaultFormatter": "esbenp.prettier-vscode",
  "editor.codeActionsOnSave": {
    "source.fixAll.eslint": true
  },
  "[typescript]": {
    "editor.defaultFormatter": "esbenp.prettier-vscode"
  },
  "typescript.tsdk": "node_modules/typescript/lib"
}
```

---

## Project Structure

### Directory Organization

```
src/
├── components/              # Reusable React components
│   ├── Header.tsx           # Main header with app launcher
│   ├── AppLauncher.tsx      # 3x3 grid app launcher modal
│   ├── AppCard.tsx          # Individual app card component
│   ├── Dashboard.tsx        # Main dashboard layout
│   ├── ErrorBoundary.tsx    # Error handling wrapper
│   └── index.ts             # Component exports
├── pages/                   # Next.js pages (auto-routed)
│   ├── _app.tsx             # App wrapper with providers
│   ├── _document.tsx        # HTML document structure
│   ├── index.tsx            # Home page (/)
│   ├── api/                 # API routes (/api/*)
│   │   ├── applications.ts  # GET /api/applications
│   │   ├── health.ts        # GET /api/health
│   │   └── analytics/
│   │       └── app-launch.ts # POST /api/analytics/app-launch
├── context/                 # React Context providers
│   ├── AuthContext.tsx      # Authentication & RBAC
│   └── LauncherContext.tsx  # Applications state
├── hooks/                   # Custom React hooks
│   └── index.ts             # useAuth, useLauncher, useResponsive, etc.
├── utils/                   # Utility functions & helpers
│   ├── mockData.ts          # Mock data & fixtures
│   └── helpers.ts           # Helper functions
├── types/                   # TypeScript type definitions
│   └── index.ts             # Application types
└── styles/                  # CSS stylesheets
    └── globals.css          # Global styles

public/
├── icons/                   # SVG icons
└── favicon.ico              # Site favicon

docker/
├── nginx/                   # Nginx configuration
│   ├── nginx.conf           # Main Nginx config
│   └── conf.d/
│       └── default.conf     # Virtual host config

Configuration Files:
├── tsconfig.json            # TypeScript config
├── tailwind.config.js       # Tailwind CSS config
├── next.config.js           # Next.js config
├── jest.config.js           # Jest testing config
├── .eslintrc.json           # ESLint config
├── .prettierrc               # Prettier config
├── package.json             # Dependencies
├── Dockerfile               # Container image
├── docker-compose.yml       # Multi-container config
├── .env.example             # Environment template
├── README.md                # Project readme
├── DEPLOYMENT.md            # Deployment guide
└── DEVELOPMENT.md           # This file
```

---

## Code Style & Standards

### TypeScript

Always use strict TypeScript:

```typescript
// ✅ Good - Strong typing
interface Props {
  apps: Application[];
  onLaunch: (app: Application) => Promise<void>;
  canAccess?: (app: Application) => boolean;
}

export const AppCard: React.FC<Props> = ({ apps, onLaunch, canAccess }) => {
  // Component implementation
};

// ❌ Avoid - Any types
export const AppCard: React.FC<any> = (props: any) => {
  // Don't use 'any'
};
```

### Naming Conventions

```typescript
// Components: PascalCase
export const AppLauncher: React.FC = () => {};

// Variables & Functions: camelCase
const getUserRole = (user: User): UserRole => {};
let isLoading = false;

// Constants: UPPER_SNAKE_CASE
const MAX_APPS_PER_PAGE = 12;
const APP_CATEGORIES = { /* ... */ };

// Private methods: _camelCase prefix
const _calculateAppScore = () => {};

// Interfaces: PascalCase with 'I' prefix or suffix
interface AppProps { }
interface Props { }
type AppCardProps = { }  // Or use 'Props' suffix
```

### File Naming

```
Components:          AppCard.tsx
Pages:               index.tsx (auto-routed)
API Routes:          /api/applications.ts
Hooks:               useAppHealth.ts (but group in index.ts)
Utils:               helpers.ts, mockData.ts
Context:             AuthContext.tsx
Types:               index.ts (grouped)
Styles:              globals.css
```

### ESLint & Prettier

```bash
# Format code
npm run lint

# Check formatting
npx prettier --check src/

# Auto-fix formatting
npx prettier --write src/
```

### Commit Messages

Use conventional commits:

```
feat: add app launcher modal
fix: resolve RBAC permission check
docs: update deployment guide
style: format component styles
refactor: extract helper functions
test: add AppCard component tests
chore: update dependencies
```

---

## Component Development

### Creating a New Component

```typescript
// src/components/MyComponent.tsx
import React from 'react';
import { MyComponentProps } from '@types/index';

/**
 * MyComponent - Brief description of component purpose
 *
 * @component
 * @example
 * <MyComponent title="Hello" onAction={() => console.log('Clicked')} />
 */
export const MyComponent: React.FC<MyComponentProps> = ({
  title,
  onAction,
}) => {
  const [state, setState] = React.useState(false);

  return (
    <div className="p-4 bg-white rounded-lg">
      <h2 className="text-lg font-bold">{title}</h2>
      <button onClick={onAction} className="mt-4 px-4 py-2 bg-blue-600 text-white rounded">
        Action
      </button>
    </div>
  );
};

export default MyComponent;
```

### Component Patterns

#### Controlled Component

```typescript
interface Props {
  value: string;
  onChange: (value: string) => void;
}

export const SearchInput: React.FC<Props> = ({ value, onChange }) => (
  <input
    value={value}
    onChange={(e) => onChange(e.target.value)}
    type="text"
    placeholder="Search..."
  />
);
```

#### Render Props

```typescript
interface Props {
  children: (data: { isLoading: boolean; data: any }) => React.ReactNode;
}

export const DataFetcher: React.FC<Props> = ({ children }) => {
  const [isLoading, setIsLoading] = React.useState(true);
  const [data, setData] = React.useState(null);

  React.useEffect(() => {
    // Fetch data
    setIsLoading(false);
  }, []);

  return <>{children({ isLoading, data })}</>;
};

// Usage
<DataFetcher>
  {({ isLoading, data }) => (
    <div>
      {isLoading ? <Spinner /> : <Content data={data} />}
    </div>
  )}
</DataFetcher>
```

#### Compound Component Pattern

```typescript
export const Dialog: React.FC<{ children: React.ReactNode }> = ({ children }) => (
  <div className="modal">{children}</div>
);

Dialog.Header = ({ children }) => <div className="modal-header">{children}</div>;
Dialog.Body = ({ children }) => <div className="modal-body">{children}</div>;
Dialog.Footer = ({ children }) => <div className="modal-footer">{children}</div>;

// Usage
<Dialog>
  <Dialog.Header>Title</Dialog.Header>
  <Dialog.Body>Content</Dialog.Body>
  <Dialog.Footer>Actions</Dialog.Footer>
</Dialog>
```

---

## State Management

### Using React Context

#### Creating a Context

```typescript
// src/context/MyContext.tsx
import React, { createContext, useContext } from 'react';

interface MyContextType {
  value: string;
  setValue: (value: string) => void;
}

const MyContext = createContext<MyContextType | undefined>(undefined);

export const MyProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [value, setValue] = React.useState('');

  return (
    <MyContext.Provider value={{ value, setValue }}>
      {children}
    </MyContext.Provider>
  );
};

export const useMyContext = (): MyContextType => {
  const context = useContext(MyContext);
  if (!context) {
    throw new Error('useMyContext must be used within MyProvider');
  }
  return context;
};
```

#### Using Contexts in App

```typescript
// src/pages/_app.tsx
import { AuthProvider } from '@context/AuthContext';
import { LauncherProvider } from '@context/LauncherContext';

export default function App({ Component, pageProps }: AppProps) {
  return (
    <AuthProvider mockMode={true}>
      <LauncherProvider mockMode={true}>
        <Component {...pageProps} />
      </LauncherProvider>
    </AuthProvider>
  );
}
```

### Local Component State

```typescript
// ✅ Good - For local component state
const [isOpen, setIsOpen] = React.useState(false);
const [formData, setFormData] = React.useState({ name: '', email: '' });

// Use for effects
React.useEffect(() => {
  // Side effects
}, [dependency]);

// Use useCallback for stable function references
const handleClick = React.useCallback(() => {
  setIsOpen(!isOpen);
}, [isOpen]);
```

---

## API Integration

### Making API Calls

```typescript
// src/utils/api.ts
import axios from 'axios';
import { Application } from '@types/index';

const apiClient = axios.create({
  baseURL: process.env.NEXT_PUBLIC_API_BASE_URL,
  timeout: 10000,
});

export const applicationAPI = {
  // Fetch all applications
  getAll: async (params?: { category?: string; role?: string }) => {
    const response = await apiClient.get<{ data: Application[] }>('/applications', {
      params,
    });
    return response.data.data;
  },

  // Fetch single application
  getById: async (id: string) => {
    const response = await apiClient.get<{ data: Application }>(
      `/applications/${id}`
    );
    return response.data.data;
  },

  // Create application (admin only)
  create: async (app: Omit<Application, 'id'>) => {
    const response = await apiClient.post<{ data: Application }>(
      '/applications',
      app
    );
    return response.data.data;
  },

  // Update application (admin only)
  update: async (id: string, app: Partial<Application>) => {
    const response = await apiClient.put<{ data: Application }>(
      `/applications/${id}`,
      app
    );
    return response.data.data;
  },

  // Delete application (admin only)
  delete: async (id: string) => {
    await apiClient.delete(`/applications/${id}`);
  },
};

// Error handling wrapper
export const handleApiError = (error: any): string => {
  if (axios.isAxiosError(error)) {
    return error.response?.data?.error || error.message;
  }
  return 'An unexpected error occurred';
};
```

### Using API in Components

```typescript
// src/components/AppList.tsx
import React from 'react';
import { applicationAPI, handleApiError } from '@utils/api';
import { Application } from '@types/index';

export const AppList: React.FC = () => {
  const [apps, setApps] = React.useState<Application[]>([]);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);

  React.useEffect(() => {
    const fetchApps = async () => {
      try {
        setLoading(true);
        setError(null);
        const data = await applicationAPI.getAll();
        setApps(data);
      } catch (err) {
        setError(handleApiError(err));
      } finally {
        setLoading(false);
      }
    };

    fetchApps();
  }, []);

  if (loading) return <div>Loading...</div>;
  if (error) return <div>Error: {error}</div>;

  return (
    <div>
      {apps.map((app) => (
        <div key={app.id}>{app.name}</div>
      ))}
    </div>
  );
};
```

---

## Testing

### Unit Tests

```typescript
// src/utils/__tests__/helpers.test.ts
import { canAccessApp, formatDate } from '@utils/helpers';
import { MOCK_USER, MOCK_APPLICATIONS } from '@utils/mockData';

describe('helpers', () => {
  describe('canAccessApp', () => {
    it('should return true when user has required role', () => {
      const app = MOCK_APPLICATIONS[0];
      const result = canAccessApp(app, MOCK_USER);
      expect(result).toBe(true);
    });

    it('should return false for guest user with restricted app', () => {
      const app = MOCK_APPLICATIONS[3]; // Admin only app
      const result = canAccessApp(app, null);
      expect(result).toBe(false);
    });
  });

  describe('formatDate', () => {
    it('should format date correctly', () => {
      const date = new Date('2024-01-15');
      const result = formatDate(date);
      expect(result).toMatch(/15.*Jan.*2024/);
    });
  });
});
```

### Component Tests

```typescript
// src/components/__tests__/AppCard.test.tsx
import React from 'react';
import { render, screen } from '@testing-library/react';
import { AppCard } from '../AppCard';
import { MOCK_APPLICATIONS } from '@utils/mockData';

describe('AppCard', () => {
  it('renders app information', () => {
    const app = MOCK_APPLICATIONS[0];
    render(<AppCard app={app} canAccess={true} />);

    expect(screen.getByText(app.name)).toBeInTheDocument();
    expect(screen.getByText(app.description)).toBeInTheDocument();
  });

  it('disables launch button when not accessible', () => {
    const app = MOCK_APPLICATIONS[0];
    render(<AppCard app={app} canAccess={false} />);

    const launchButton = screen.getByRole('button', { name: /launch/i });
    expect(launchButton).toBeDisabled();
  });
});
```

### Running Tests

```bash
# Run all tests
npm test

# Watch mode
npm run test:watch

# Coverage report
npm test -- --coverage
```

---

## Debugging

### Browser DevTools

1. **React DevTools**
   - Install from Chrome Web Store
   - Inspect component props and state
   - Track component re-renders

2. **Network Tab**
   - Monitor API calls
   - Check request/response payloads
   - Verify headers and status codes

3. **Console**
   - Check for errors
   - Use `console.log()` for debugging
   - Inspect application state

### VS Code Debugging

Create `.vscode/launch.json`:

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Next.js",
      "type": "node",
      "request": "launch",
      "skipFiles": ["<node_internals>/**"],
      "program": "${workspaceFolder}/node_modules/.bin/next",
      "args": ["dev"],
      "console": "integratedTerminal"
    }
  ]
}
```

### Debug Logging

```typescript
// Development logging helper
const DEBUG = process.env.NODE_ENV === 'development';

const log = {
  info: (message: string, data?: any) => {
    if (DEBUG) console.log(`[INFO] ${message}`, data);
  },
  error: (message: string, error?: any) => {
    console.error(`[ERROR] ${message}`, error);
  },
  warn: (message: string, data?: any) => {
    console.warn(`[WARN] ${message}`, data);
  },
};

// Usage
log.info('App loading', { appCount: apps.length });
log.error('Failed to fetch', error);
```

---

## Common Tasks

### Adding a New Application

1. **Update Mock Data**
```typescript
// src/utils/mockData.ts
export const MOCK_APPLICATIONS: Application[] = [
  // ... existing apps
  {
    id: 'app-007',
    name: 'New Application',
    description: 'Description here',
    icon: '🆕',
    url: '/apps/new-app',
    category: 'tools',
    status: AppStatus.OPERATIONAL,
    requiredRoles: [UserRole.ANALYST, UserRole.MANAGER, UserRole.ADMIN],
    version: '1.0.0',
  },
];
```

2. **Create Docker Container**
```yaml
# docker-compose.yml
new-app:
  container_name: statgate-new-app
  image: nginx:alpine
  ports:
    - "3006:80"
  networks:
    - statgate-network
```

3. **Update Nginx Config**
```nginx
# docker/nginx/conf.d/default.conf
location /apps/new-app {
  proxy_pass http://new-app;
}
```

### Implementing RBAC

```typescript
// Check permissions in component
import { useAuth } from '@context/AuthContext';

export const AdminPanel: React.FC = () => {
  const { hasRole, user } = useAuth();

  if (!hasRole('admin')) {
    return <div>Access Denied</div>;
  }

  return <div>Admin Content</div>;
};
```

### Adding New API Endpoint

1. **Create Route**
```typescript
// src/pages/api/my-endpoint.ts
export default function handler(req: NextApiRequest, res: NextApiResponse) {
  if (req.method !== 'GET') {
    return res.status(405).json({ error: 'Method not allowed' });
  }

  // Handle request
  res.status(200).json({ data: 'response' });
}
```

2. **Call from Client**
```typescript
const response = await fetch('/api/my-endpoint');
const data = await response.json();
```

### Styling a Component

Use Tailwind CSS classes:

```typescript
<div className="max-w-md mx-auto p-6 bg-white rounded-lg shadow-lg border border-gray-200">
  <h2 className="text-2xl font-bold text-gray-900 mb-4">Title</h2>
  <p className="text-gray-600 mb-6">Description</p>
  <button className="w-full bg-blue-600 text-white py-2 px-4 rounded-lg hover:bg-blue-700 transition-colors">
    Action
  </button>
</div>
```

---

## Contribution Workflow

### Creating a Feature Branch

```bash
# Update main branch
git checkout main
git pull origin main

# Create feature branch
git checkout -b feat/feature-name

# Make changes and commit
git add .
git commit -m "feat: add feature description"

# Push to remote
git push origin feat/feature-name
```

### Pull Request Checklist

- [ ] Branch created from `main` (or `develop`)
- [ ] Code follows style guidelines
- [ ] Tests written and passing
- [ ] No console errors or warnings
- [ ] Components properly typed with TypeScript
- [ ] Documentation updated (if applicable)
- [ ] Commit messages follow conventional commits
- [ ] PR description explains changes

### Code Review Process

1. **Request review** from team members
2. **Address feedback** from reviewers
3. **Run full test suite** before merge
4. **Squash commits** if necessary
5. **Merge to main** once approved

### Deployment from Feature

```bash
# After PR is merged to main
git checkout main
git pull origin main

# Build and test
npm install
npm run build
npm test

# Deploy to staging
docker build -t statgate-launcher:staging .
docker-compose up -d

# If successful, deploy to production
docker build -t statgate-launcher:1.0.0 .
docker push registry.example.com/statgate-launcher:1.0.0
```

---

## Performance Tips

### Component Optimization

```typescript
// Memoize expensive components
export const HeavyComponent = React.memo(({ data }: Props) => (
  <div>{/* Heavy rendering */}</div>
));

// Use useMemo for expensive calculations
const expensiveValue = React.useMemo(() => {
  return complexCalculation(data);
}, [data]);

// Use useCallback for stable function references
const handleClick = React.useCallback(() => {
  // Handler
}, [dependency]);
```

### Bundle Optimization

```bash
# Analyze bundle size
npm install -D @next/bundle-analyzer
```

Update `next.config.js`:
```javascript
const withBundleAnalyzer = require('@next/bundle-analyzer')({
  enabled: process.env.ANALYZE === 'true',
});

module.exports = withBundleAnalyzer(nextConfig);
```

Run analysis:
```bash
ANALYZE=true npm run build
```

---

## Resources

- [Next.js Documentation](https://nextjs.org/docs)
- [React Documentation](https://react.dev)
- [TypeScript Handbook](https://www.typescriptlang.org/docs)
- [Tailwind CSS](https://tailwindcss.com/docs)
- [Jest Testing](https://jestjs.io/docs/getting-started)

For questions or issues, reach out to the development team.
