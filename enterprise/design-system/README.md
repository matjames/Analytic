# StatGate Enterprise Design System

A centralized UI library for all StatGate applications to maintain a consistent user experience.

## Components

| Component | Description |
|-----------|-------------|
| **Button** | Primary, secondary, success, danger, ghost variants with sizes and loading state |
| **Card** | Default, outlined, elevated variants with header, title, body sub-components |
| **Badge** | Primary, success, warning, error, neutral, info variants with dot indicator |
| **Input** | Text input with label and error state |
| **Select** | Dropdown with label, error, and options |
| **TextArea** | Multi-line input with label and error state |
| **Table** | Sortable data table with columns, accessors, and empty state |
| **Dialog** | Modal dialog with sizes, header, body, footer |
| **Notification** | Enterprise notification with priority, category, source app, deep links, actions, attachments |
| **NotificationBell** | Bell icon with unread count badge |
| **Layout** | App shell with sidebar, header, content areas |

## Design Tokens

- **Colors**: Primary, success, warning, error, neutral palettes
- **Spacing**: xs through 3xl scale
- **Typography**: Font families, sizes, weights
- **Radii**: sm through full
- **Shadows**: sm through xl
- **Themes**: Light and dark mode support
- **Accessibility**: Focus rings, reduced motion, high contrast

## Usage

```tsx
import { Button, Card, Badge, Input, Table, Dialog, Notification } from '@statgate/design-system';

// Button
<Button variant="primary" size="md" onClick={handleClick}>Save</Button>

// Card
<Card variant="elevated">
  <CardHeader><CardTitle>Project Overview</CardTitle></CardHeader>
  <CardBody>Content here</CardBody>
</Card>

// Badge
<Badge variant="success" dot>Active</Badge>

// Input
<Input label="Project Name" placeholder="Enter name" error={error} />

// Table
<Table
  columns={[{ key: 'name', header: 'Name', sortable: true }]}
  data={projects}
/>

// Dialog
<Dialog open={isOpen} onClose={close} title="Confirm" size="md">
  <p>Are you sure?</p>
</Dialog>

// Notification
<Notification
  id="n1"
  title="Project Created"
  body="Malaria Surveillance has been created."
  priority="high"
  category="project"
  sourceApp="pms"
  deepLink="/projects/PRJ-001"
  actions={[{ label: 'View', variant: 'primary' }]}
/>
```

## Installation

```bash
cd enterprise/design-system
npm install
npm run build
```

## Integration

Applications import components from this shared design system:

```bash
npm install @statgate/design-system
```

```tsx
import { Button, Card } from '@statgate/design-system';
```

## Dark Mode

```tsx
import { themes } from '@statgate/design-system';

// Apply theme
document.documentElement.classList.toggle('dark', theme.mode === 'dark');
```
