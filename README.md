# Stationmaster

A Bring Your Own Server (BYOS) solution for TRMNL with a Go backend and static React frontend. 

## Features

- 🔐 **Multiple Authentication Methods**
  - Username/Password authentication
  - API Key authentication
  - OIDC/SSO integration
  - Proxy authentication support
  
- 👥 **User Management**
  - Multi-user support with admin roles
  - User registration (public or admin-only)
  - Password reset functionality
  - User profile management
  
- 📱 **Device Management**
  - TRMNL device integration
  - Firmware management and updates
  - Device scheduling and configuration
  
- 🔌 **Private Plugin System**
  - Custom liquid templates with multi-layout support
  - Webhook, polling, and merge data strategies
  - Built-in Monaco editor with syntax highlighting
  - Live preview with layout selection
  - Secure template validation and sandboxing
  
- 🎨 **Modern UI**
  - React + TypeScript frontend
  - Tailwind CSS styling
  - Dark/Light theme support
  - Responsive design
  
- 🗄️ **Database Support**
  - SQLite (default)
  - PostgreSQL (production)

## Quick Start

### Using Docker Compose

1. Clone the repository:
```bash
git clone https://github.com/rmitchellscott/stationmaster.git
cd stationmaster
```

2. Copy the example environment file:
```bash
cp .env.example .env
```

3. Edit `.env` and configure your settings (see Environment Variables section below)

4. Start the application:
```bash
docker-compose up -d
```

5. Access the application at http://localhost:8000

### Development Setup

1. Install dependencies:
```bash
# Backend
go mod download

# Frontend
cd ui
npm install
```

2. Start the development servers:
```bash
# Backend (in one terminal)
go run .

# Frontend (in another terminal)
cd ui
npm run dev
```

## Environment Variables

### Core Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8000` | Port for the web server |
| `GIN_MODE` | `release` | Gin framework mode (`debug`, `release`, `test`) |
| `DATA_DIR` | `/data` | Directory for data storage |

### Database Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_TYPE` | `sqlite` | Database type (`sqlite`, `postgres`) |
| `DB_HOST` | `localhost` | Database host (PostgreSQL only) |
| `DB_PORT` | `5432` | Database port (PostgreSQL only) |
| `DB_USER` | `stationmaster` | Database username (PostgreSQL only) |
| `DB_PASSWORD` | - | Database password (PostgreSQL only) |
| `DB_NAME` | `stationmaster` | Database name (PostgreSQL only) |
| `DB_SSLMODE` | `disable` | SSL mode for PostgreSQL |
| `DATABASE_URL` | `/data/stationmaster.db` | Complete database connection string |

### Authentication & Security

| Variable | Default | Description |
|----------|---------|-------------|
| `JWT_SECRET` | - | **REQUIRED** Secret key for JWT tokens |
| `SESSION_TIMEOUT` | `24h` | JWT token expiration time |
| `ALLOW_INSECURE` | `false` | Allow insecure connections (HTTP) |
| `AUTH_USERNAME` | - | Basic auth username for legacy auth |
| `AUTH_PASSWORD` | - | Basic auth password for legacy auth |
| `API_KEY` | - | Global API key for legacy auth |
| `BLOCK_PRIVATE_IPS` | `false` | Block requests to private IP addresses |
| `BLOCKED_DOMAINS` | - | Comma-separated list of domains to block |

### User Management

| Variable | Default | Description |
|----------|---------|-------------|
| `PUBLIC_REGISTRATION_ENABLED` | `false` | Controls user registration setting (overrides database and locks admin control when set) |
| `ADMIN_USERNAME` | - | Initial admin username |
| `ADMIN_PASSWORD` | - | Initial admin password |
| `ADMIN_EMAIL` | - | Initial admin email |
| `DISABLE_WELCOME_EMAIL` | `false` | Disable welcome emails for new users |

### OIDC/SSO Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `OIDC_ENABLED` | `false` | Enable OIDC authentication |
| `OIDC_ISSUER` | - | OIDC provider issuer URL |
| `OIDC_CLIENT_ID` | - | OIDC client ID |
| `OIDC_CLIENT_SECRET` | - | OIDC client secret |
| `OIDC_REDIRECT_URL` | - | OIDC callback URL |
| `OIDC_SCOPES` | `openid profile email` | OIDC scopes to request |
| `OIDC_SSO_ONLY` | `false` | Disable local login when OIDC is enabled |
| `OIDC_AUTO_CREATE_USERS` | `false` | Auto-create users from OIDC claims |
| `OIDC_ADMIN_GROUP` | - | OIDC group that grants admin privileges |
| `OIDC_SUCCESS_REDIRECT_URL` | - | Redirect URL after successful OIDC login |
| `OIDC_POST_LOGOUT_REDIRECT_URL` | - | Redirect URL after OIDC logout |
| `OIDC_BUTTON_TEXT` | - | Custom text for OIDC login button |
| `OIDC_DEBUG` | `false` | Enable OIDC debug logging |

### Proxy Authentication

| Variable | Default | Description |
|----------|---------|-------------|
| `PROXY_AUTH_ENABLED` | `false` | Enable proxy authentication |
| `PROXY_AUTH_HEADER` | `X-Remote-User` | Header containing authenticated username |

### SMTP Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `SMTP_ENABLED` | `false` | Enable SMTP for email functionality |
| `SMTP_HOST` | - | SMTP server hostname |
| `SMTP_PORT` | `587` | SMTP server port |
| `SMTP_USERNAME` | - | SMTP authentication username |
| `SMTP_PASSWORD` | - | SMTP authentication password |
| `SMTP_FROM` | - | From address for outgoing emails |
| `SMTP_TLS` | `true` | Use TLS for SMTP connection |
| `SITE_URL` | - | Base URL for email links |

### TRMNL Integration

| Variable | Default | Description |
|----------|---------|-------------|
| `TRMNL_MODEL_API_URL` | `https://usetrmnl.com/api/models` | API URL for device models |
| `TRMNL_FIRMWARE_API_URL` | `https://usetrmnl.com/api/firmware/latest` | API URL for firmware updates |
| `MODEL_POLLER` | `true` | Enable automatic model polling |
| `MODEL_POLLER_INTERVAL` | `1h` | Interval for model polling |
| `FIRMWARE_POLLER` | `true` | Enable automatic firmware polling |
| `FIRMWARE_POLLER_INTERVAL` | `1h` | Interval for firmware polling |
| `FIRMWARE_STORAGE_DIR` | `/data/firmware` | Directory for firmware storage |
| `FIRMWARE_AUTO_DOWNLOAD` | `true` | Automatically download new firmware |

### Private Plugin Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `PRIVATE_PLUGINS_ENABLED` | `true` | Enable private plugin system |
| `PLUGIN_WEBHOOK_TIMEOUT` | `30s` | Timeout for webhook data submissions |
| `PLUGIN_POLLING_INTERVAL` | `5m` | Default polling interval for external APIs |
| `PLUGIN_MAX_DATA_SIZE` | `2048` | Maximum data size in bytes for webhooks |
| `PLUGIN_TEMPLATE_TIMEOUT` | `10s` | Template rendering timeout |
| `PLUGIN_SANDBOX_ENABLED` | `true` | Enable template security sandboxing |

### Logging & Debugging

| Variable | Default | Description |
|----------|---------|-------------|
| `LOG_LEVEL` | `INFO` | Logging level (`DEBUG`, `INFO`, `WARN`, `ERROR`) |
| `LOG_FORMAT` | `text` | Log output format (`text`, `json`) |
| `DRY_RUN` | `false` | Enable dry-run mode (no actual changes) |

### File-based Secrets

All environment variables support file-based configuration by appending `_FILE` to the variable name. For example:
- `JWT_SECRET_FILE=/run/secrets/jwt_secret`
- `DB_PASSWORD_FILE=/run/secrets/db_password`

This is useful for Docker secrets or other secret management systems.

## Database Configuration

### SQLite (Default)
```env
DATABASE_URL=/data/stationmaster.db
```

### PostgreSQL (Production)
```env
DB_TYPE=postgres
DATABASE_URL=postgres://user:password@host/dbname?sslmode=disable
```

## API Documentation

### Authentication Endpoints

- `POST /api/auth/login` - User login
- `POST /api/auth/logout` - User logout
- `GET /api/auth/check` - Check authentication status
- `POST /api/auth/register/public` - Public registration
- `POST /api/auth/password-reset` - Request password reset

### User Management (Admin)

- `GET /api/users` - List all users
- `PUT /api/users/:id` - Update user
- `POST /api/users/:id/promote` - Promote to admin
- `DELETE /api/users/:id` - Delete user

### Profile Management

- `PUT /api/profile` - Update current user profile
- `POST /api/profile/password` - Change password
- `DELETE /api/profile` - Delete account

### Device Management

- `GET /api/devices` - List devices
- `POST /api/devices` - Add device
- `PUT /api/devices/:id` - Update device
- `DELETE /api/devices/:id` - Delete device

### Private Plugin System

- `GET /api/private-plugins` - List private plugins
- `POST /api/private-plugins` - Create private plugin
- `PUT /api/private-plugins/:id` - Update private plugin
- `DELETE /api/private-plugins/:id` - Delete private plugin
- `POST /api/private-plugins/:id/webhook` - Submit webhook data
- `GET /api/private-plugins/:id/render/:layout` - Render plugin template

For detailed documentation, see [docs/PRIVATE_PLUGINS.md](docs/PRIVATE_PLUGINS.md)

## Building from Source

### Backend
```bash
go build -o stationmaster .
```

### Frontend
```bash
cd ui
npm run build
```

### Docker Image
```bash
docker build -t stationmaster .
```

## Security Considerations

- **Always change the default `JWT_SECRET`** in production
- Use strong passwords for admin accounts
- Enable HTTPS in production (set `ALLOW_INSECURE=false`)
- Use PostgreSQL for production deployments
- Consider using file-based secrets for sensitive configuration

## License

MIT License - See LICENSE file for details
