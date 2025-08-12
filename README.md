# Stationmaster

A robust user management and authentication system built with Go and React. Based on the authentication system from Aviary, Stationmaster provides a clean foundation for building applications that require user management.

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
  
- 🔑 **API Key Management**
  - Generate and manage API keys
  - Key expiration and rotation
  - Usage tracking
  
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

3. Edit `.env` and set your configuration (especially `JWT_SECRET`)

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

## Configuration

### Environment Variables

Key configuration options:

- `MULTI_USER_MODE`: Enable multi-user mode (default: true)
- `PUBLIC_REGISTRATION_ENABLED`: Allow public user registration
- `JWT_SECRET`: Secret key for JWT tokens (required)
- `OIDC_ENABLED`: Enable OIDC authentication
- `PROXY_AUTH_ENABLED`: Enable proxy authentication
- `SMTP_ENABLED`: Enable email functionality for password resets

See `.env.example` for all available options.

### Database Configuration

By default, Stationmaster uses SQLite. For production, PostgreSQL is recommended:

```env
DATABASE_TYPE=postgres
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

### API Keys

- `GET /api/api-keys` - List user's API keys
- `POST /api/api-keys` - Create new API key
- `DELETE /api/api-keys/:id` - Delete API key

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

## License

MIT License - See LICENSE file for details

## Credits

Based on the authentication system from [Aviary](https://github.com/rmitchellscott/aviary)