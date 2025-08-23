# Private Plugin System

Stationmaster's Private Plugin System enables users to create custom plugins using Liquid templates and the TRMNL design framework. This powerful system allows for personalized content creation with built-in security, validation, and multi-layout support.

## 🎯 Overview

Private plugins are user-created content generators that render dynamic information on TRMNL devices. Unlike system plugins, private plugins are:

- **User-created**: Built using a visual editor with liquid templates
- **Secure**: Templates are validated and sandboxed for safety
- **Multi-layout**: Support all TRMNL screen layouts automatically  
- **Flexible**: Three data strategies for different use cases
- **Framework-integrated**: Automatically uses TRMNL's CSS/JS framework

## 🏗️ Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   User Creates  │    │  Template Engine │    │ TRMNL Framework │
│  Liquid Template│────│   (Security +    │────│   (CSS/JS)      │
│                 │    │   Validation)    │    │                 │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                │
                                ▼
                    ┌──────────────────┐
                    │ HTML → PNG       │
                    │ Browserless      │
                    │ Rendering        │
                    └──────────────────┘
                                │
                                ▼
                    ┌──────────────────┐
                    │ Device Display   │
                    │ (E-ink Optimized)│
                    └──────────────────┘
```

### Core Components

1. **Liquid Template Engine**: Processes user templates with security restrictions
2. **Validation System**: Ensures templates are safe and properly containerized
3. **Data Strategies**: Three methods for getting data into templates
4. **Multi-layout Support**: Automatic adaptation to all TRMNL screen sizes
5. **TRMNL Framework Integration**: Consistent styling and responsive design

## 📊 Data Strategies

### Webhook Strategy
Push data to your plugin from external systems.

- **Use case**: Real-time dashboards, notifications, IoT sensors
- **How it works**: Generate unique webhook URL, receive POST requests
- **Data limit**: 2KB per request
- **Example**: `POST /api/webhooks/plugin/{token}` with JSON payload

```json
{
  "temperature": "72°F",
  "humidity": "45%",
  "status": "comfortable"
}
```

### Polling Strategy  
Pull data from external APIs at regular intervals.

- **Use case**: Weather, news feeds, API integrations
- **How it works**: Configure URLs and intervals, system polls automatically
- **Flexibility**: Multiple URLs, custom headers, retry logic
- **Example**: Poll weather API every 30 minutes

```json
{
  "urls": [
    {
      "url": "https://api.weather.com/current",
      "headers": {"Authorization": "Bearer token"},
      "interval": 1800
    }
  ]
}
```

### Merge Strategy
Combine with existing plugin data for enhanced functionality.

- **Use case**: Extend system plugins with custom formatting
- **How it works**: Access data from other plugins in your templates
- **Benefits**: Leverage existing data sources with custom presentation

## 🎨 Layout System

Private plugins automatically support all TRMNL layouts:

| Layout | Dimensions | CSS Class | Use Case |
|--------|------------|-----------|----------|
| **Full Screen** | 800×480 | `view--full` | Dashboards, detailed info |
| **Half Vertical** | 400×480 | `view--half_vertical` | Side panels, widgets |
| **Half Horizontal** | 800×240 | `view--half_horizontal` | Status bars, tickers |
| **Quadrant** | 400×240 | `view--quadrant` | Small widgets, icons |

Each plugin template is automatically rendered for the appropriate layout size and optimized for e-ink displays.

## 🛡️ Security Features

### Template Validation
All templates undergo security validation:

- **Script blocking**: No JavaScript execution allowed
- **Resource restrictions**: External CSS/images blocked  
- **XSS prevention**: Input sanitization and output encoding
- **Containerization**: Unique instance IDs prevent CSS conflicts

### Sandboxed Execution
Templates run in a restricted environment:

- **No file system access**: Templates cannot read/write files
- **Network restrictions**: Only configured data sources allowed
- **Memory limits**: Prevents resource exhaustion
- **Timeout protection**: Templates must render within time limits

## 🎯 Key Features

### Visual Template Editor
- **Monaco Editor**: Full-featured code editor with syntax highlighting
- **Real-time Validation**: Instant feedback on template errors
- **Live Preview**: See your plugin render in real-time
- **Multi-layout Testing**: Preview across all screen sizes

### TRMNL Framework Integration
- **Automatic CSS/JS**: Framework assets included automatically
- **Responsive Classes**: Use `text--huge`, `u--align-center`, etc.
- **E-ink Optimized**: Colors and animations optimized for display
- **Typography**: Consistent fonts and sizing across plugins

### Template Management
- **Version Control**: Track changes to your templates
- **Publishing**: Control when plugins go live
- **Sharing**: Export/import plugin configurations (future)
- **Statistics**: Usage and performance metrics

## 🚀 Quick Start

1. **Navigate to Plugin Management** → **Private Plugins** tab
2. **Click "Create Private Plugin"** to open the editor
3. **Choose a data strategy** (webhook, polling, or merge)
4. **Write your templates** using Liquid syntax and TRMNL classes
5. **Test with sample data** using the preview feature
6. **Validate templates** for security and best practices
7. **Save and create instances** to use on your devices

## 📖 Documentation

- **[User Guide](./private-plugins/USER_GUIDE.md)** - Step-by-step plugin creation
- **[Template Reference](./private-plugins/TEMPLATE_REFERENCE.md)** - Liquid syntax and TRMNL classes  
- **[API Documentation](./private-plugins/API.md)** - Technical API reference
- **[Examples](./private-plugins/examples/)** - Working example plugins

## 🔧 Environment Variables

Private plugin system configuration:

| Variable | Default | Description |
|----------|---------|-------------|
| `PRIVATE_PLUGINS_ENABLED` | `true` | Enable private plugin system |
| `WEBHOOK_TOKEN_LENGTH` | `32` | Length of generated webhook tokens |
| `TEMPLATE_VALIDATION_STRICT` | `true` | Strict template validation mode |
| `MAX_TEMPLATE_SIZE` | `50KB` | Maximum template size |
| `POLLING_MIN_INTERVAL` | `300` | Minimum polling interval (seconds) |
| `POLLING_MAX_URLS` | `5` | Maximum URLs per polling config |

## 🌟 Use Cases

### Business Dashboard
Combine webhook data with polling APIs to create comprehensive business dashboards showing KPIs, alerts, and real-time metrics.

### Home Automation
Display IoT sensor data, smart home status, and environmental information with real-time updates via webhooks.

### Personal Productivity
Create custom layouts for calendar events, task lists, weather, and other personal information with polling integrations.

### Content Aggregation
Combine multiple news feeds, social media, and content sources into personalized information displays.

## 🎨 Best Practices

### Template Design
- Use TRMNL CSS classes instead of inline styles
- Test all layout sizes during development
- Avoid animations (e-ink displays don't support them)
- Use appropriate contrast for readability

### Data Management
- Choose webhook for real-time data, polling for periodic updates
- Keep webhook payloads under 2KB for best performance
- Set reasonable polling intervals to avoid rate limiting
- Cache data appropriately to reduce API calls

### Security
- Validate all user inputs in webhook endpoints
- Use HTTPS for polling URLs when possible
- Never include sensitive data in templates
- Follow principle of least privilege for API access

## 🤝 Contributing

The private plugin system is designed to be extensible:

- **Template Functions**: Add new Liquid filters and functions
- **Data Sources**: Extend polling and webhook capabilities  
- **Validation Rules**: Enhance security and best practice checks
- **Framework Integration**: Improve TRMNL CSS/JS integration

## 📊 API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/private-plugins` | List user's private plugins |
| `POST` | `/api/private-plugins` | Create new private plugin |
| `PUT` | `/api/private-plugins/:id` | Update existing plugin |
| `DELETE` | `/api/private-plugins/:id` | Delete private plugin |
| `POST` | `/api/private-plugins/validate` | Validate templates |
| `POST` | `/api/private-plugins/test` | Test with sample data |
| `POST` | `/api/webhooks/plugin/:token` | Webhook data endpoint |

---

*The Private Plugin System provides a secure, flexible way to create custom content for TRMNL devices while maintaining the reliability and performance users expect.*