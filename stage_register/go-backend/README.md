# Go Backend - Setup and Deployment Guide

This guide provides step-by-step instructions for setting up and deploying the Go backend API for the Facility Registry.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Development Setup](#development-setup)
- [Environment Configuration](#environment-configuration)
- [Database Setup](#database-setup)
- [Running the Application](#running-the-application)
- [Production Deployment](#production-deployment)
- [Service Management](#service-management)
- [Nginx Configuration](#nginx-configuration)
- [Troubleshooting](#troubleshooting)

---

## Prerequisites

Before you begin, ensure you have the following installed:

- **Go** (version 1.24.5 or higher)
  - Download from: https://golang.org/dl/
  - Verify installation: `go version`

- **PostgreSQL** (version 12 or higher)
  - Download from: https://www.postgresql.org/download/
  - Verify installation: `psql --version`

- **Git** (for cloning the repository)
  - Verify installation: `git --version`

- **Linux/Unix system** (for production deployment with systemd and nginx)

---

## Development Setup

Follow these steps to set up the project for local development:

### Step 1: Clone the Repository

```bash
git clone <repository-url>
cd facility-registry/go-backend
```

### Step 2: Initialize Go Module (if not already done)

```bash
go mod init go-backend
```

### Step 3: Install Dependencies

Install all required Go packages:

```bash
go get github.com/gin-gonic/gin
go get github.com/joho/godotenv
go get github.com/lib/pq
go get golang.org/x/crypto/bcrypt
go get github.com/golang-jwt/jwt/v5
go get github.com/gin-contrib/cors
```

### Step 4: Install Air (Hot Reload Tool)

Air allows automatic reloading during development:

```bash
go install github.com/air-verse/air@latest
```

**Note:** Make sure your `$GOPATH/bin` or `$GOBIN` is in your `$PATH` to use the `air` command.

### Step 5: Initialize Air Configuration

Generate the Air configuration file:

```bash
air init
```

This creates a `.air.toml` configuration file in your project root. You can customize it if needed.

### Step 6: Create Environment File

Create a `.env` file in the project root directory:

```bash
touch .env
```

See [Environment Configuration](#environment-configuration) section for required variables.

---

## Environment Configuration

Create a `.env` file in the project root with the following variables:

```env
# Server Configuration
PORT=9090

# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=your_db_user
DB_PASSWORD=your_db_password
DB_NAME=your_database_name

# JWT Secret (use a strong, random string in production)
JWT_SECRET=your-super-secret-jwt-key-change-in-production
```

### Environment Variables Explained

- **PORT**: The port number the server will listen on (default: 8080)
- **DB_HOST**: PostgreSQL server hostname (localhost for local development)
- **DB_PORT**: PostgreSQL server port (default: 5432)
- **DB_USER**: PostgreSQL username
- **DB_PASSWORD**: PostgreSQL password
- **DB_NAME**: Name of the PostgreSQL database
- **JWT_SECRET**: Secret key for signing JWT tokens (must be a strong random string)

**⚠️ Security Note:** Never commit the `.env` file to version control. Make sure it's listed in `.gitignore`.

---

## Database Setup

### Step 1: Create PostgreSQL Database

1. **Connect to PostgreSQL:**
   ```bash
   psql -U postgres
   ```

2. **Create a new database:**
   ```sql
   CREATE DATABASE your_database_name;
   ```

3. **Create a new user (optional, but recommended):**
   ```sql
   CREATE USER your_db_user WITH PASSWORD 'your_db_password';
   GRANT ALL PRIVILEGES ON DATABASE your_database_name TO your_db_user;
   ```

4. **Exit PostgreSQL:**
   ```sql
   \q
   ```

### Step 2: Run Database Migrations (if applicable)

If you have database migration scripts, run them to create the necessary tables:

```bash
# Example (adjust based on your migration setup):
psql -U your_db_user -d your_database_name -f migrations/schema.sql
```

### Step 3: Verify Database Connection

Ensure your database credentials in the `.env` file are correct. The application will attempt to connect when it starts.

---

## Running the Application

### Development Mode (with Hot Reload)

Use Air for automatic reloading during development:

```bash
air
```

The server will start and automatically reload when you make changes to the code.

### Development Mode (without Hot Reload)

Run the application directly:

```bash
go run .
```

The server will start on the port specified in your `.env` file (default: 9090).

### Verify the Application is Running

1. **Check the console output** - You should see:
   ```
   ✅ Database connected
   [GIN-debug] Listening and serving HTTP on :8080
   ```

2. **Test the API:**
   ```bash
   curl http://localhost:9090/register
   ```

---

## Production Deployment

Follow these steps to deploy the application on a Linux server:

### Step 1: Build the Application

Create a production binary:

```bash
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o app
```

This creates an executable file named `app` in your current directory.

### Step 2: Prepare Deployment Directory

Create the deployment directory and copy files:

```bash
sudo mkdir -p /opt/nhfr-api
sudo chown -R $USER:$USER /opt/nhfr-api
```

```bash
Copy binary
scp app root@64.226.104.49:/opt/nhfr-api/
```

```bash
create .env file on the server
sudo nano .env

PORT=9090

DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=xxx
DB_NAME=xxx
DB_SSLMODE=disable

JWT_SECRET=12345WQWTYUGBVNcders
JWT_EXPIRES_MINUTES=120
```

### Step 3: Make Binary Executable

```bash
cd /opt/nhfr-api
chmod +x app
```

Press `Ctrl+C` to stop it after verifying it runs correctly.

---

## Service Management

Set up the application as a systemd service for automatic startup and management.

### Step 1: Create systemd Service File

Create the service file:

```bash
sudo nano /etc/systemd/system/nhfr-api.service
```

### Step 2: Add Service Configuration

Paste the following configuration:

```ini
[Unit]
Description=Go Backend API Service
After=network.target

[Service]
User=www-data
WorkingDirectory=/opt/nhfr-api
ExecStart=/opt/nhfr-api/app
Restart=always
RestartSec=5
EnvironmentFile=/opt/nhfr-api/.env

[Install]
WantedBy=multi-user.target
```

### Step 3: Enable and Start the Service

1. **Reload systemd daemon:**
   ```bash
   sudo systemctl daemon-reload
   ```

2. **Enable the service** (starts on boot):
   ```bash
   sudo systemctl enable nhfr-api
   ```

3. **Start the service:**
   ```bash
   sudo systemctl start nhfr-api
   ```

4. **Check service status:**
   ```bash
   sudo systemctl status nhfr-api
   ```

### Step 4: View Service Logs

View real-time logs:
```bash
sudo journalctl -u nhfr-api -f
```

View recent logs:
```bash
sudo journalctl -u nhfr-api -n 100
```

### Common Service Management Commands

```bash
# Stop the service
sudo systemctl stop nhfr-api

# Restart the service
sudo systemctl restart nhfr-api

# Reload configuration (if service file changed)
sudo systemctl daemon-reload
sudo systemctl restart nhfr-api

# Disable auto-start on boot
sudo systemctl disable nhfr-api

# Check if service is running
sudo systemctl is-active nhfr-api
```

---

## Nginx Configuration

Set up Nginx as a reverse proxy in front of your Go API.

### Step 1: Install Nginx

```bash
sudo apt update
sudo apt install nginx -y
```

### Step 2: Check Nginx Status

```bash
sudo systemctl status nginx
```

Ensure Nginx is running. If not, start it:

```bash
sudo systemctl start nginx
sudo systemctl enable nginx
```

### Step 3: Configure Firewall

Allow Nginx through the firewall:

```bash
sudo ufw allow 'Nginx Full'
```

Or for specific ports:

```bash
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
```

### Step 4: Create Nginx Configuration

Create a new site configuration:

```bash
sudo nano /etc/nginx/sites-available/nhfr-api
```

### Step 5: Add Nginx Configuration

Paste the following configuration:

```nginx
server {
    listen 80;
    server_name 64.226.104.49;  # Replace with your domain or IP

    # Increase limits for large requests (optional)
    client_max_body_size 20M;

    location / {
        proxy_pass http://127.0.0.1:9090;  # Match your PORT in .env

        # Required headers
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # Timeouts (important for long-running requests)
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;

        # WebSocket support (if needed)
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

### Step 6: Enable the Site

1. **Create symbolic link:**
   ```bash
   sudo ln -s /etc/nginx/sites-available/nhfr-api /etc/nginx/sites-enabled/
   ```

2. **Test Nginx configuration:**
   ```bash
   sudo nginx -t
   ```

   You should see:
   ```
   nginx: the configuration file /etc/nginx/nginx.conf syntax is ok
   nginx: configuration file /etc/nginx/nginx.conf test is successful
   ```

3. **Reload Nginx:**
   ```bash
   sudo systemctl reload nginx
   ```

### Step 7: Verify Nginx is Working

Test the API through Nginx:

```bash
curl http://your-server-ip-or-domain/
```

### Deploy React App


```bash
yarn build
sudo mkdir -p /var/www/nhfr
sudo chown -R $USER:$USER /var/www/nhfr
scp -r build/* root@64.226.104.49:/var/www/nhfr/hmtl/

sudo nano /etc/nginx/sites-available/nhfr-app

server {
    listen 80;
    server_name 64.226.104.49;

    root /var/www/nhfr;
    index index.html;

    location / {
        try_files $uri /index.html;
    }
}

sudo ln -s /etc/nginx/sites-available/nhfr-app /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

This automatically configures HTTPS for your domain.

---

## Troubleshooting

### Database Connection Issues

**Problem:** Application fails to connect to database

**Solutions:**
1. Verify PostgreSQL is running:
   ```bash
   sudo systemctl status postgresql
   ```

2. Check database credentials in `.env` file

3. Test connection manually:
   ```bash
   psql -h localhost -U your_db_user -d your_database_name
   ```

4. Verify PostgreSQL is accepting connections in `pg_hba.conf`

5. Check firewall settings for PostgreSQL port (5432)

### Service Won't Start

**Problem:** systemd service fails to start

**Solutions:**
1. Check service status:
   ```bash
   sudo systemctl status go-api
   ```

2. View detailed logs:
   ```bash
   sudo journalctl -u go-api -n 50 --no-pager
   ```

3. Verify file permissions:
   ```bash
   ls -la /opt/go-api/
   ```

4. Test binary manually:
   ```bash
   cd /opt/go-api
   sudo -u www-data ./app
   ```

5. Check `.env` file exists and has correct format:
   ```bash
   sudo cat /opt/go-api/.env
   ```

### Nginx 502 Bad Gateway

**Problem:** Nginx returns 502 error

**Solutions:**
1. Verify Go API is running:
   ```bash
   sudo systemctl status go-api
   ```

2. Check if API is listening on the correct port:
   ```bash
   sudo netstat -tlnp | grep 8080
   # or
   sudo ss -tlnp | grep 8080
   ```

3. Test API directly (bypass Nginx):
   ```bash
   curl http://localhost:8080/
   ```

4. Check Nginx error logs:
   ```bash
   sudo tail -f /var/log/nginx/error.log
   ```

### Port Already in Use

**Problem:** Port 8080 (or your configured port) is already in use

**Solutions:**
1. Find process using the port:
   ```bash
   sudo lsof -i :8080
   # or
   sudo netstat -tlnp | grep 8080
   ```

2. Change PORT in `.env` file to an available port

3. Update Nginx configuration to match new port

4. Restart services

### Permission Denied Errors

**Problem:** Permission denied when running service

**Solutions:**
1. Verify file ownership:
   ```bash
   sudo chown -R www-data:www-data /opt/go-api
   ```

2. Ensure binary is executable:
   ```bash
   sudo chmod +x /opt/go-api/app
   ```

3. Check directory permissions:
   ```bash
   sudo chmod 755 /opt/go-api
   ```

### JWT Token Issues

**Problem:** Authentication tokens not working

**Solutions:**
1. Verify `JWT_SECRET` is set in `.env` file

2. Ensure `JWT_SECRET` is the same across all instances (if running multiple)

3. Check token expiration time in `utils/auth.go`

4. Verify request headers include `Authorization: Bearer <token>`

---

## Additional Resources

- [Go Documentation](https://golang.org/doc/)
- [Gin Framework Documentation](https://gin-gonic.com/docs/)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [systemd Service Documentation](https://www.freedesktop.org/software/systemd/man/systemd.service.html)
- [Nginx Documentation](https://nginx.org/en/docs/)

---

## Support

For issues or questions, please contact the development team or create an issue in the repository.
