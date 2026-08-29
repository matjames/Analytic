# StatGate Report Builder

A Go-based REST API server with an interactive web dashboard for reports. Features YAML-based report definitions, parameterized queries, and responsive visualizations. Demo site - report.statgate.io

## Project Structure

```
dwh_visualization/
├── server/
│   ├── main.go                    # HTTP server setup, BASE_PATH handling
│   ├── handlers.go                # Report & filter API handlers
│   ├── schema_handlers.go         # Schema introspection & builder API handlers
│   ├── models.go                  # Data structures (Query, Aggregation, GroupBy, Calculate)
│   ├── reports.go                 # Report loading, filter application, parallel execution
│   ├── database.go                # PostgreSQL connection pool, filter binding
│   ├── db_retry.go                # Database connection retry with backoff
│   ├── query_builder.go           # SQL generation entry point
│   ├── query_builder_v2.go        # Squirrel-based SQL generation
│   ├── expression_parser.go       # BODMAS expression parsing for column math
│   ├── column_resolver.go         # Column type detection and safe wrapping
│   ├── period_expansion.go        # Multi-period query expansion (months, weeks)
│   ├── transformer.go             # Data transformation for component types
│   ├── schema.go                  # Database schema introspection
│   ├── yaml_generator.go          # YAML report generation from builder
│   ├── auth.go                    # Keycloak JWT authentication
│   ├── cache.go                   # Component result caching (3-day TTL)
│   ├── middleware.go              # Rate limiting, panic recovery
│   ├── metrics.go                 # In-memory request metrics
│   ├── request_context.go         # Per-request context (filters, warnings)
│   ├── warnings.go                # Component warning collection
│   ├── sql_logger.go              # SQL query logging
│   ├── server_logger.go           # Server activity logging
│   └── *_test.go                  # Unit tests
│   ├── configs/                      # YAML report definitions (auto-discovered)
│   │   ├── Reports/                  # Builder-authored report pack
│   │   └── (schema/domain areas as configured)
├── public/
│   ├── index.html                 # Main dashboard UI
│   ├── builder.html               # Visual YAML report builder
│   ├── metrics.html               # Usage metrics dashboard
│   ├── sw.js                      # Service worker (PWA offline support)
│   ├── manifest.json              # PWA manifest
│   ├── js/
│   │   ├── app.js                 # Dashboard application logic
│   │   ├── components.js          # Chart/table/map rendering
│   │   ├── metrics.js             # Metrics dashboard
│   │   ├── apiCache.js            # Frontend API response cache
│   │   ├── auth.js                # Authentication UI logic
│   │   ├── authFetch.js           # Auth-aware fetch wrapper
│   │   ├── config.js              # Frontend configuration
│   │   ├── filterManager.js       # Filter state management
│   │   ├── pwa.js                 # PWA registration and updates
│   │   └── builder/               # Report builder modules
│   │       ├── main.js            # Builder initialization
│   │       ├── state.js           # Global state management
│   │       ├── api.js             # Backend API calls
│   │       ├── components.js      # Component editing UI
│   │       ├── sections.js        # Section management
│   │       ├── query-form.js      # Query builder form
│   │       ├── preview.js         # YAML preview
│   │       ├── preview-tab.js     # Live report preview
│   │       ├── publish.js         # Publish to server
│   │       ├── validation.js      # Form validation
│   │       ├── autocomplete.js    # Column name autocomplete
│   │       ├── keywords.js        # SQL keyword definitions
│   │       ├── recent-tables.js   # Recently used tables
│   │       ├── utils.js           # Builder utilities
│   │       └── yaml-handler.js    # YAML import/export
│   ├── css/
│   │   ├── style.css              # Main styles
│   │   ├── builder.css            # Builder-specific styles
│   │   ├── Aimara.css             # Aimara design system styles
│   │   └── _buttons.css           # Button component styles
│   ├── assets/                    # Icons, logos, GeoJSON data
│   └── lib/                       # Third-party libraries
├── .env.example                   # Environment config template
├── go.mod / go.sum                # Go dependencies
└── generated_sql.log              # SQL query audit trail (auto-generated)
```

## Quick Start

### Prerequisites

- Go 1.21 or later
- PostgreSQL database
- Bash terminal
- Create a `.env` file with database credentials by copying the .env.example file

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | Server port |
| `BASE_PATH` | (none) | Subpath for reverse proxy deployment |
| `ALLOWED_ORIGIN` | (none) | CORS origin restriction (enables credentials) |
| `COMPONENT_TIMEOUT_SECONDS` | `30` | Max time per component query |
| `RATE_LIMIT_PER_SEC` | `100` | Request rate limit |
| `RATE_LIMIT_BURST` | `200` | Maximum burst size |
| `AUTH_MODE` | `off` | `off` (dev) or `on` (production, requires Keycloak) |

See `.env.example` for database and Keycloak configuration.

### Local test database (optional)

For a **local PostgreSQL** database with DWH-like schemas and synthetic seed data (so reports and schema features work end-to-end without production data), use the Makefile target from the project root:

```bash
make test-db
```

Details, overrides (`TEST_DB_NAME`, `PGHOST`, `PGUSER`, …), and how this relates to `configs/` are documented in **[database/README.md](database/README.md)**.

### Build the server

```bash
go build -o report_server ./server
```

### Run

**Manual run**

```bash
# Default port 8080
./report_server

# Custom port
PORT=9000 ./report_server
```

Server runs on `http://localhost:8080` by default (must run from project root)

### Subpath Deployment (Reverse Proxy)

When deploying behind a reverse proxy at a subpath (e.g., `example.com/dashboards/viz`), set the `BASE_PATH` environment variable:

```bash
# Deploy at /dashboards/viz
BASE_PATH=/dashboards/viz ./report_server

# With custom port
BASE_PATH=/dashboards/viz PORT=9000 ./report_server
```

**How it works:**
- All API routes are prefixed: `/api/reports` → `/dashboards/viz/api/reports`
- HTML files get `window.BASE_PATH` injected for JavaScript
- No trailing slash in BASE_PATH

**Nginx example:**
```nginx
location /dashboards/viz/ {
    proxy_pass http://localhost:8080/dashboards/viz/;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
}
```

**Local development:** No `BASE_PATH` needed—works at root by default.

### PDF Generation Setup

The server uses **chromedp** (headless Chrome) to generate PDFs from report HTML. This requires Chrome/Chromium to be installed on the server.

**Install Chrome/Chromium:**

```bash
# Ubuntu/Debian
sudo apt-get update
sudo apt-get install -y chromium-browser

# RHEL/CentOS/Fedora
sudo dnf install chromium

# macOS (for local development)
brew install --cask google-chrome
```

**How it works:**
- Frontend captures charts (Chart.js) and maps (Leaflet) as images using `html2canvas`
- HTML with embedded images is sent to `POST /api/report/pdf`
- Server uses headless Chrome to render HTML and generate PDF
- Concurrent PDF generations limited to 5 (prevents resource exhaustion)

**Chrome flags used:**
- `--headless` - No GUI
- `--no-sandbox` - Required for Docker/containerized environments
- `--disable-dev-shm-usage` - Prevents shared memory issues on Linux

**Troubleshooting:**
- If PDFs fail, verify Chrome is installed: `which chromium-browser` or `which google-chrome`
- For Docker deployments, ensure Chrome is in the container image
- Check server logs for specific chromedp errors

## Authentication

The server supports JWT authentication via Keycloak. Configure using environment variables:

```bash
# Required for auth
KEYCLOAK_URL=https://auth.example.com    # Keycloak server URL
KEYCLOAK_REALM=MyRealm                   # Keycloak realm name
KEYCLOAK_CLIENT_ID=my-client             # Client ID for this app

# Auth mode (default: off)
AUTH_MODE=off   # Development: no authentication, publishing allowed
AUTH_MODE=on    # Production: requires valid JWT, publishing disabled
```

**Note:** When `AUTH_MODE=on`, the builder's Publish tab is hidden and the publish endpoint returns 403. All report changes must go through GitHub in production.

**How it works:**
- Tokens are validated against Keycloak's JWKS endpoint
- Public keys are cached for 1 hour
- `/api/auth/config` is always public (frontend needs it to configure auth)
- Admin-only endpoints (e.g., cache clear) require `admin` role

## Rate Limiting

The server includes rate limiting to prevent abuse:

```bash
# Default: 100 requests/second, burst of 200
RATE_LIMIT_PER_SEC=100   # Requests per second
RATE_LIMIT_BURST=200     # Maximum burst size
```

When rate limit is exceeded, the server returns `429 Too Many Requests`.

## API Endpoints

### Reports

```bash
GET /api/reports                        # List all available reports
GET /api/report/{id}                    # Get report by ID
GET /api/report/{id}?district=Central    # With district filter
GET /api/report/{id}?district=Central&year=2024&month=3  # Multiple filters
GET /api/report/{id}?week=2024W15       # Week filter (ISO format)
GET /api/report/{id}?quarter=1&year=2024  # Quarter filter
GET /api/report/source/{id}             # Get raw YAML for editing
POST /api/report/pdf                    # Generate PDF from HTML content
POST /api/report/builder-preview        # Preview report from builder
POST /api/report/publish                # Save report YAML to server
```

### Filters

```bash
GET /api/filters/districts              # Districts from database
GET /api/filters/regions                # Regions from database
GET /api/filters/facilities             # Facilities from database
GET /api/filters/years                  # Available years
GET /api/filters/months                 # Returns 1-12
GET /api/filters/quarters               # Returns 1-4
GET /api/filters/weeks                  # All available weeks
GET /api/filters/weeks?year=2024        # Weeks filtered by year
GET /api/filters/custom?column=X&table=Y  # Custom filter values
```

### Schema Introspection (Builder)

```bash
GET /api/schema/schemas                 # List database schemas
GET /api/schema/tables?schema=report    # Tables in schema
GET /api/schema/table/{schema}/{table}/columns  # Column info
```

### Query Builder

```bash
GET /api/builder/categories             # Report categories for publishing
POST /api/query/validate                # Validate query structure
POST /api/query/preview                 # Preview query results
POST /api/report/generate               # Generate YAML from builder
POST /api/report/parse                  # Parse existing YAML
```

### Cache

```bash
GET /api/cache/stats                    # View cache statistics
GET /api/cache/stats?entries=true       # Include individual cache entries
POST /api/cache/clear                   # Clear entire cache (admin only)
```

### Auth

```bash
GET /api/auth/config                    # Auth configuration (public, no auth required)
```

### Metrics

```bash
GET /api/metrics                        # Usage statistics (in-memory)
```

## Component Types

### 1. Text

- Renders HTML content
- Supports `{{value}}` placeholder populated from SQL query results
- Useful for summaries, KPIs, and recommendations

### 2. Line Chart

- Time series visualization
- Multiple data series support
- Chart.js powered

### 3. Bar Chart

- Category comparison
- Multiple data series support
- Chart.js powered

### 4. Bar-Line Chart (Combo)

- Combined bar and line visualization
- First dataset renders as bars, rest as lines
- Useful for showing values with trends

### 5. Table

- Data in tabular format
- Styled rows with alternating colors
- Headers from SQL query column names
- Use `table` for structured queries, `table_advanced` for raw SQL

### 6. Map

- Geographic visualization with markers
- Leaflet.js integration
- Color-coded markers based on values

### 7. Choropleth

- District-level heatmap visualization
- Uganda districts support
- Interactive tooltips
- Supports custom SQL with column mapping

### Chart Enhancements

#### Reference Lines (Bar/Line Charts)

Add horizontal reference lines for targets, thresholds, or baselines:

```yaml
- type: bar
  title: Fever Cases
  query:
    # ... query config
  referenceLines:
    - value: 15000
      label: Target
      color: '#22c55e'
      style: dashed
    - value: 10000
      label: Baseline
      color: '#9ca3af'
      style: dotted
```

#### Target Comparison (Text Components)

Compare KPI values against static targets:

```yaml
- type: text
  content: "<h2>Coverage: {{value}}%</h2>"
  target:
    value: 80
    label: "Goal"
  query:
    # ... query config
```

### Choropleth with Custom SQL

For complex choropleth queries, use raw SQL with column mapping:

```yaml
- type: choropleth
  title: District Cases
  sql: "SELECT district, SUM(cases) as total FROM schema.table GROUP BY district"
  filterTable: schema.table  # Table for filter application
  columnMapping:
    district: district       # Column containing district names
    value: total             # Column containing values
  data:
    geojson_path: /assets/uganda_districts.geojson
    color_scheme:
      - '#90ee90'
      - '#ffd700'
      - '#ff6347'
```

## Layout Types

### Single Column

- Full width component
- Used for tables, full-width text, or standalone visualizations

### Two Column

- Side-by-side layout
- Perfect for related visualizations
- Responsive: stacks on tablets (≤1024px) and mobile (≤768px)

### Three Column

- Three-component horizontal layout
- Great for comparing multiple datasets
- Responsive: stacks to 2 columns on tablets, 1 column on mobile

### Four Column

- Four-component horizontal layout
- Optimal for displaying multiple related metrics
- Responsive: stacks to 2 columns on tablets, 1 column on mobile

## Report Builder

The visual builder at `/builder` provides a tab-based workflow:

1. **Import YAML** — Load existing report for editing
2. **Edit Existing** — Select published report to modify
3. **Metadata** — Report info, filters, custom columns
4. **Sections** — Add/edit sections and components
5. **Preview** — Test report with live filters
6. **Publish** — Save to server filesystem

## Query Format

### Structured Query (Recommended)

```yaml
query:
  table: "report.cht_form_097b"
  aggregations:
    - column: "num_fever_u5_male + num_fever_u5_female"
      function: "sum" # sum, count, avg, max, min
      alias: "Fever Cases"
  groupBy:
    - field: "period_date"
      format: "month" # month, year, date, or blank
  calculate: # Optional: post-aggregation formula
    formula: "(positive / tested * 100)"
    roundTo: 1
    whenZero: 0
```

**Benefits:**

- No SQL syntax required
- Automatic type conversion for TEXT-stored numerics
- Built-in date formatting
- Type-safe validation
- 80% less verbose than raw SQL

### Advanced Filter Configuration

```yaml
# Custom column mappings for time filters
timeColumns:
  year: period_year
  month: period_month
  week: period_week

# Custom column mappings for location filters
locationColumns:
  district: district_name
  region: region_name
  facility: facility_name

# Custom dropdown filters
customFilters:
  - column: facility_type
    table: schema.table
    label: "Facility Type"
    type: select
    defaultValue: "HC III"  # optional; single-select filters use this or the first available value

# Auto-expand to N periods ending at selected period
periodLimit: 6
```

### Parameterized Query Features

All queries use PostgreSQL parameter binding (`$1`, `$2`, ...) to prevent SQL injection. District, year, month, quarter, and week filters are whitelist-validated before query execution. Invalid values are silently ignored.

### BODMAS Support in Column Expressions

Column expressions support full BODMAS operator precedence (`+`, `-`, `*`, `/`, parentheses). TEXT-stored numbers are automatically wrapped with safe COALESCE/CAST conversion.

```yaml
aggregations:
  - column: "(fever_male + fever_female) * ratio / total"
    function: "sum"
    alias: "Adjusted Cases"
```

## Testing

### Unit Tests

```bash
go test ./server                 # Run all tests
go test ./server -v              # Verbose output
go test ./server -cover          # Show coverage (~38%)
go test ./server -race           # Race condition detection
go test ./server -run TestName   # Run specific test
```

### API Testing

```bash
curl http://localhost:8080/api/reports
curl "http://localhost:8080/api/report/executive-overview?district=Central&year=2024&month=3"
```

## SQL Query Logging

All SQL queries are automatically logged to `generated_sql.log`:

- Generated queries (from structured query builder)
- Raw SQL (from legacy YAML components)
- Filtered queries (after district/year/month filters)
- Executed queries (with execution time and row count)

## Caching

The server implements a two-layer caching strategy to reduce database load and improve response times.

### Backend Cache (Server)

- **What's cached**: Individual component results (charts, tables, text)
- **Cache key**: `{reportID}:{sectionIdx}:{componentIdx}:{filters}`
- **TTL**: 3 days
- **Invalidation**: Automatic when report YAML file changes or TTL expires
- **Storage**: In-memory (cleared on server restart)

**API Endpoints:**
```bash
GET /api/cache/stats              # View cache statistics
GET /api/cache/stats?entries=true # Include individual cache entries
POST /api/cache/clear             # Clear entire cache
```

Cache stats are also displayed on the metrics page (`/metrics`).

### Frontend Cache (Browser)

- **What's cached**: API responses (reports, filters, schema)
- **TTL**: 5 minutes for report data, up to 1 day for stable data (filters, schema)
- **Storage**: In-memory (cleared on page refresh)

### How They Work Together

```
Browser Request
    ↓
[Frontend Cache] ── HIT ──→ Return instantly (no network)
    │ MISS
    ↓
[Backend Cache] ── HIT ──→ Return from memory (no DB query)
    │ MISS
    ↓
[Database] ──→ Query, cache result, return
```

This means:
- Repeated requests within 5 minutes hit frontend cache (instant)
- Requests after frontend cache expires hit backend cache (fast, no DB)
- Only cache misses on both layers trigger database queries

## Adding New Reports

1. Create `.yaml` file in `configs/` with report structure
2. Define sections with components using structured query format
3. Specify filters: `district`, `year`, `month`
4. No code changes required - API auto-discovers YAML reports
5. Frontend automatically renders any report structure

### Key Files for Development

- **server/models.go**: Data structures and query format
- **server/query_builder_v2.go**: SQL generation logic
- **server/reports.go**: Report loading and filter application
- **public/js/components.js**: Frontend rendering logic
