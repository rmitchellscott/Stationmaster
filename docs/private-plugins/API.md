# Private Plugin API Documentation

Complete API reference for integrating with Stationmaster's private plugin system.

## 📋 Table of Contents

- [Authentication](#authentication)
- [Private Plugin Endpoints](#private-plugin-endpoints)
- [Webhook Endpoints](#webhook-endpoints)
- [Validation API](#validation-api)
- [Error Codes](#error-codes)
- [Rate Limiting](#rate-limiting)
- [SDK Examples](#sdk-examples)

## 🔐 Authentication

All private plugin API endpoints require authentication using one of the following methods:

### Session Authentication (Web UI)
Used by the web interface, handled automatically by browser sessions.

### API Key Authentication
Include API key in request headers:

```http
GET /api/private-plugins
Authorization: Bearer your-api-key-here
```

### User Token Authentication
Use JWT tokens from login endpoints:

```http
GET /api/private-plugins  
Authorization: Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9...
```

## 🔌 Private Plugin Endpoints

### List Private Plugins

Get all private plugins for the authenticated user.

```http
GET /api/private-plugins
```

**Response:**
```json
{
  "private_plugins": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "Weather Widget",
      "description": "Display current weather conditions",
      "markup_full": "<div id=\"plugin-{{ instance_id }}\">...</div>",
      "markup_half_vert": "",
      "markup_half_horiz": "",
      "markup_quadrant": "",
      "shared_markup": "",
      "data_strategy": "webhook",
      "polling_config": null,
      "form_fields": null,
      "version": "1.0.0",
      "is_published": true,
      "created_at": "2024-03-15T10:30:00Z",
      "updated_at": "2024-03-15T10:30:00Z",
      "webhook_url": "https://your-domain.com/api/webhooks/plugin/abc123def456"
    }
  ]
}
```

### Get Private Plugin

Retrieve a specific private plugin by ID.

```http
GET /api/private-plugins/{id}
```

**Parameters:**
- `id` (string) - UUID of the private plugin

**Response:**
```json
{
  "private_plugin": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "Weather Widget",
    "description": "Display current weather conditions",
    "markup_full": "<div id=\"plugin-{{ instance_id }}\">...</div>",
    "data_strategy": "webhook",
    "version": "1.0.0",
    "webhook_url": "https://your-domain.com/api/webhooks/plugin/abc123def456"
  }
}
```

### Create Private Plugin

Create a new private plugin.

```http
POST /api/private-plugins
Content-Type: application/json
```

**Request Body:**
```json
{
  "name": "My New Plugin",
  "description": "A custom plugin for my data",
  "markup_full": "<div id=\"plugin-{{ instance_id }}\" class=\"plugin-container view--full\">...</div>",
  "markup_half_vert": "",
  "markup_half_horiz": "",
  "markup_quadrant": "",
  "shared_markup": "",
  "data_strategy": "webhook",
  "polling_config": {
    "urls": [
      {
        "url": "https://api.example.com/data",
        "method": "GET",
        "headers": {
          "Authorization": "Bearer token"
        },
        "interval": 1800
      }
    ]
  },
  "form_fields": {},
  "version": "1.0.0"
}
```

**Response:**
```json
{
  "private_plugin": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "My New Plugin",
    "webhook_url": "https://your-domain.com/api/webhooks/plugin/abc123def456"
  }
}
```

**Validation Errors:**
```json
{
  "error": "Template validation failed",
  "validation_errors": [
    "full layout: Template must include a container div with id='plugin-{{ instance_id }}'",
    "full layout: Script tags are not allowed for security reasons"
  ],
  "validation_warnings": [
    "full layout: Inline styles found - consider using TRMNL framework classes"
  ]
}
```

### Update Private Plugin

Update an existing private plugin.

```http
PUT /api/private-plugins/{id}
Content-Type: application/json
```

**Parameters:**
- `id` (string) - UUID of the private plugin

**Request Body:** (Same as create request)

**Response:** (Same as create response)

### Delete Private Plugin

Delete a private plugin and all its instances.

```http
DELETE /api/private-plugins/{id}
```

**Parameters:**
- `id` (string) - UUID of the private plugin

**Response:**
```json
{
  "message": "Private plugin deleted successfully"
}
```

### Regenerate Webhook Token

Generate a new webhook token for a private plugin.

```http
POST /api/private-plugins/{id}/regenerate-token
```

**Parameters:**
- `id` (string) - UUID of the private plugin

**Response:**
```json
{
  "webhook_token": "new-token-here",
  "webhook_url": "https://your-domain.com/api/webhooks/plugin/new-token-here"
}
```

## 🔗 Webhook Endpoints

### Submit Webhook Data

Send data to a private plugin via webhook.

```http
POST /api/webhooks/plugin/{token}
Content-Type: application/json
```

**Parameters:**
- `token` (string) - Unique webhook token for the plugin

**Request Body:**
```json
{
  "temperature": "72°F",
  "humidity": "45%",
  "location": "Living Room",
  "status": "comfortable",
  "sensors": {
    "motion": false,
    "light": 350
  },
  "timestamp": "2024-03-15T10:30:00Z"
}
```

**Size Limit:** 2KB per request

**Response:**
```json
{
  "message": "Webhook data received successfully",
  "timestamp": "2024-03-15T10:30:00Z"
}
```

**Error Response:**
```json
{
  "error": "Webhook token not found",
  "code": "WEBHOOK_NOT_FOUND"
}
```

### Webhook Data Format

Webhook data should be valid JSON with the following guidelines:

- **Maximum size**: 2KB per request
- **Data types**: Supports strings, numbers, booleans, arrays, objects
- **Nesting**: Up to 5 levels deep
- **Reserved keys**: Avoid `user`, `device`, `timestamp`, `instance_id`

**Example Data Structures:**

**Simple Values:**
```json
{
  "status": "online",
  "count": 42,
  "enabled": true
}
```

**Nested Objects:**
```json
{
  "weather": {
    "current": {
      "temperature": 72,
      "humidity": 45,
      "condition": "sunny"
    },
    "forecast": [
      {"day": "tomorrow", "high": 75, "low": 65},
      {"day": "thursday", "high": 78, "low": 68}
    ]
  }
}
```

**Arrays:**
```json
{
  "news": [
    {
      "title": "Breaking News",
      "summary": "Important announcement",
      "published": "2024-03-15T10:00:00Z"
    }
  ],
  "tags": ["urgent", "breaking", "news"]
}
```

## ✅ Validation API

### Validate Plugin Templates

Validate plugin templates for security and best practices.

```http
POST /api/private-plugins/validate
Content-Type: application/json
```

**Request Body:**
```json
{
  "name": "Test Plugin",
  "description": "Testing validation",
  "markup_full": "<div id=\"plugin-{{ instance_id }}\">Content</div>",
  "markup_half_vert": "",
  "markup_half_horiz": "",
  "markup_quadrant": "",
  "shared_markup": "",
  "data_strategy": "webhook",
  "polling_config": null,
  "form_fields": null,
  "version": "1.0.0"
}
```

**Response (Success):**
```json
{
  "valid": true,
  "message": "Templates validated successfully",
  "warnings": [
    "full layout: Consider using TRMNL framework classes instead of inline styles"
  ],
  "errors": []
}
```

**Response (Failure):**
```json
{
  "valid": false,
  "message": "Template validation failed",
  "warnings": [],
  "errors": [
    "full layout: Template must include a container div with id='plugin-{{ instance_id }}'",
    "full layout: Script tags are not allowed for security reasons",
    "At least one layout template must be provided"
  ]
}
```

### Test Plugin with Sample Data

Test plugin rendering with sample data.

```http
POST /api/private-plugins/test
Content-Type: application/json
```

**Request Body:**
```json
{
  "plugin": {
    "name": "Test Plugin",
    "markup_full": "<div id=\"plugin-{{ instance_id }}\">{{ data.message }}</div>",
    "data_strategy": "webhook"
  },
  "layout": "full",
  "sample_data": {
    "message": "Hello, World!",
    "temperature": "72°F"
  },
  "device_width": 800,
  "device_height": 480
}
```

**Response:**
```json
{
  "message": "Plugin test completed successfully",
  "preview_url": "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAA...",
  "render_time_ms": 1250,
  "layout": "full",
  "dimensions": {
    "width": 800,
    "height": 480
  }
}
```

## 🔄 Polling Configuration

### Polling Config Schema

For plugins using the "polling" data strategy:

```json
{
  "urls": [
    {
      "url": "https://api.example.com/data",
      "method": "GET",
      "headers": {
        "Authorization": "Bearer token",
        "User-Agent": "Stationmaster/1.0"
      },
      "interval": 1800,
      "timeout": 30,
      "retry_count": 3,
      "retry_delay": 5
    }
  ],
  "merge_responses": true,
  "cache_duration": 300
}
```

**URL Configuration:**
- `url` (string, required) - API endpoint URL
- `method` (string) - HTTP method (GET, POST) - default: GET
- `headers` (object) - HTTP headers to send
- `interval` (number) - Polling interval in seconds - min: 300, max: 86400
- `timeout` (number) - Request timeout in seconds - default: 30
- `retry_count` (number) - Number of retries on failure - default: 3
- `retry_delay` (number) - Delay between retries in seconds - default: 5

**Global Configuration:**
- `merge_responses` (boolean) - Merge multiple URL responses - default: true
- `cache_duration` (number) - Response cache duration in seconds - default: 300

### Polling Response Format

Polling responses are merged and available as `data.*` in templates:

**Single URL Response:**
```json
// API Response
{
  "temperature": 72,
  "humidity": 45
}

// Available in template
{{ data.temperature }}  <!-- 72 -->
{{ data.humidity }}     <!-- 45 -->
```

**Multiple URL Responses (Merged):**
```json
// First API: {"weather": {"temp": 72}}
// Second API: {"news": [{"title": "..."}]}

// Available in template  
{{ data.weather.temp }}    <!-- 72 -->
{{ data.news[0].title }}   <!-- First news title -->
```

## ❌ Error Codes

### HTTP Status Codes

- `200` - Success
- `201` - Created successfully
- `400` - Bad request / Validation error
- `401` - Unauthorized
- `403` - Forbidden
- `404` - Not found
- `422` - Unprocessable entity (validation failed)
- `429` - Rate limit exceeded
- `500` - Internal server error

### Custom Error Codes

**Plugin Management:**
- `PLUGIN_NOT_FOUND` - Plugin ID doesn't exist or not owned by user
- `PLUGIN_VALIDATION_FAILED` - Template validation failed
- `PLUGIN_NAME_REQUIRED` - Plugin name is required
- `PLUGIN_TEMPLATES_REQUIRED` - At least one template must be provided

**Webhook Errors:**
- `WEBHOOK_NOT_FOUND` - Webhook token not found
- `WEBHOOK_PAYLOAD_TOO_LARGE` - Payload exceeds 2KB limit
- `WEBHOOK_INVALID_JSON` - Request body is not valid JSON
- `WEBHOOK_RATE_LIMITED` - Too many requests to webhook

**Template Validation:**
- `TEMPLATE_SYNTAX_ERROR` - Liquid template syntax error
- `TEMPLATE_SECURITY_VIOLATION` - Template contains blocked content
- `TEMPLATE_CONTAINER_MISSING` - Required container div missing
- `TEMPLATE_SIZE_EXCEEDED` - Template too large

**Polling Configuration:**
- `POLLING_CONFIG_INVALID` - Invalid polling configuration format
- `POLLING_URL_INVALID` - Polling URL is not valid
- `POLLING_INTERVAL_INVALID` - Polling interval out of range
- `POLLING_TOO_MANY_URLS` - Too many URLs in polling config

## 🛡️ Rate Limiting

### API Endpoints
- **Plugin CRUD**: 100 requests per minute per user
- **Validation**: 50 requests per minute per user  
- **Testing**: 20 requests per minute per user

### Webhook Endpoints
- **Data submission**: 300 requests per minute per token
- **Burst allowance**: 10 requests per second

### Rate Limit Headers

Responses include rate limiting information:

```http
HTTP/1.1 200 OK
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1710515400
```

### Rate Limit Exceeded Response

```json
{
  "error": "Rate limit exceeded",
  "code": "RATE_LIMIT_EXCEEDED",
  "retry_after": 60
}
```

## 🧰 SDK Examples

### JavaScript/Node.js

```javascript
class StationmasterPrivatePlugins {
  constructor(baseUrl, apiKey) {
    this.baseUrl = baseUrl;
    this.apiKey = apiKey;
  }

  async request(method, path, data = null) {
    const response = await fetch(`${this.baseUrl}${path}`, {
      method,
      headers: {
        'Authorization': `Bearer ${this.apiKey}`,
        'Content-Type': 'application/json'
      },
      body: data ? JSON.stringify(data) : null
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(`${error.error} (${error.code})`);
    }

    return response.json();
  }

  // List all private plugins
  async listPlugins() {
    const result = await this.request('GET', '/api/private-plugins');
    return result.private_plugins;
  }

  // Create a new plugin
  async createPlugin(pluginData) {
    const result = await this.request('POST', '/api/private-plugins', pluginData);
    return result.private_plugin;
  }

  // Send webhook data
  async sendWebhookData(token, data) {
    const result = await this.request('POST', `/api/webhooks/plugin/${token}`, data);
    return result;
  }

  // Validate templates
  async validatePlugin(pluginData) {
    const result = await this.request('POST', '/api/private-plugins/validate', pluginData);
    return result;
  }
}

// Usage
const client = new StationmasterPrivatePlugins('https://your-domain.com', 'your-api-key');

// Create a plugin
const plugin = await client.createPlugin({
  name: 'Temperature Monitor',
  description: 'Displays temperature data',
  markup_full: '<div id="plugin-{{ instance_id }}">{{ data.temp }}°F</div>',
  data_strategy: 'webhook',
  version: '1.0.0'
});

// Send data to plugin
await client.sendWebhookData(plugin.webhook_token, {
  temp: 72,
  humidity: 45,
  location: 'Living Room'
});
```

### Python

```python
import requests
import json

class StationmasterPrivatePlugins:
    def __init__(self, base_url, api_key):
        self.base_url = base_url
        self.api_key = api_key
        self.session = requests.Session()
        self.session.headers.update({
            'Authorization': f'Bearer {api_key}',
            'Content-Type': 'application/json'
        })

    def _request(self, method, path, data=None):
        url = f"{self.base_url}{path}"
        response = self.session.request(method, url, json=data)
        
        if not response.ok:
            error = response.json()
            raise Exception(f"{error['error']} ({error.get('code', 'UNKNOWN')})")
        
        return response.json()

    def list_plugins(self):
        result = self._request('GET', '/api/private-plugins')
        return result['private_plugins']

    def create_plugin(self, plugin_data):
        result = self._request('POST', '/api/private-plugins', plugin_data)
        return result['private_plugin']

    def send_webhook_data(self, token, data):
        result = self._request('POST', f'/api/webhooks/plugin/{token}', data)
        return result

    def validate_plugin(self, plugin_data):
        result = self._request('POST', '/api/private-plugins/validate', plugin_data)
        return result

# Usage
client = StationmasterPrivatePlugins('https://your-domain.com', 'your-api-key')

# Create a plugin
plugin = client.create_plugin({
    'name': 'Weather Station',
    'description': 'IoT weather station data',
    'markup_full': '<div id="plugin-{{ instance_id }}">{{ data.temperature }}°F</div>',
    'data_strategy': 'webhook',
    'version': '1.0.0'
})

# Send sensor data
client.send_webhook_data(plugin['webhook_token'], {
    'temperature': 72,
    'humidity': 45,
    'pressure': 30.15,
    'timestamp': '2024-03-15T10:30:00Z'
})
```

### cURL Examples

**Create Plugin:**
```bash
curl -X POST https://your-domain.com/api/private-plugins \
  -H "Authorization: Bearer your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Status Monitor",
    "description": "System status display",
    "markup_full": "<div id=\"plugin-{{ instance_id }}\">Status: {{ data.status }}</div>",
    "data_strategy": "webhook",
    "version": "1.0.0"
  }'
```

**Send Webhook Data:**
```bash
curl -X POST https://your-domain.com/api/webhooks/plugin/your-webhook-token \
  -H "Content-Type: application/json" \
  -d '{
    "status": "online",
    "uptime": "99.9%",
    "last_check": "2024-03-15T10:30:00Z"
  }'
```

**Validate Templates:**
```bash
curl -X POST https://your-domain.com/api/private-plugins/validate \
  -H "Authorization: Bearer your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Plugin",
    "markup_full": "<div id=\"plugin-{{ instance_id }}\">Test</div>",
    "data_strategy": "webhook"
  }'
```

## 📊 Admin Endpoints

These endpoints require admin privileges:

### Get Private Plugin Statistics

```http
GET /api/admin/private-plugins/stats
```

**Response:**
```json
{
  "stats": {
    "total_plugins": 25,
    "total_instances": 78,
    "webhook_plugins": 15,
    "polling_plugins": 8,
    "merge_plugins": 2,
    "published_plugins": 20,
    "total_webhook_requests_24h": 15420,
    "average_plugin_size": 2.3
  }
}
```

---

*This API documentation covers all aspects of integrating with the private plugin system. For implementation examples and best practices, see the [User Guide](./USER_GUIDE.md) and [Template Reference](./TEMPLATE_REFERENCE.md).*