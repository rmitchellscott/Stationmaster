# Plugin System

This directory contains the modular plugin system for Stationmaster, designed to support both simple image plugins and rich data plugins that can be converted from TRMNL's open source plugin library.

## Architecture

The plugin system supports two types of plugins:

### Image Plugins (`PluginTypeImage`)
- Return pre-rendered image URLs directly
- Simple configuration and fast response times
- Examples: `redirect`, `alias`, `core_proxy`

### Data Plugins (`PluginTypeData`)  
- Return structured data with HTML templates
- Rendered server-side to images for device display
- Rich content with dynamic styling
- Examples: `days_left_until`

## Adding a New Plugin

### 1. Create Plugin Directory
```bash
mkdir internal/plugins/my_plugin
```

### 2. Implement Plugin Interface

**For Image Plugin:**
```go
// internal/plugins/my_plugin/my_plugin.go
package my_plugin

import "github.com/rmitchellscott/stationmaster/internal/plugins"

type MyPlugin struct{}

func (p *MyPlugin) Type() string { return "my_plugin" }
func (p *MyPlugin) PluginType() plugins.PluginType { return plugins.PluginTypeImage }
func (p *MyPlugin) Name() string { return "My Plugin" }
func (p *MyPlugin) Description() string { return "Description of what it does" }

func (p *MyPlugin) ConfigSchema() string {
    return `{
        "type": "object",
        "properties": {
            "setting1": {"type": "string", "title": "Setting 1"}
        },
        "required": ["setting1"]
    }`
}

func (p *MyPlugin) Validate(settings map[string]interface{}) error {
    // Validation logic
    return nil
}

func (p *MyPlugin) Process(ctx plugins.PluginContext) (plugins.PluginResponse, error) {
    // Plugin logic
    imageURL := "https://example.com/image.png"
    filename := "my_plugin_output"
    refreshRate := 3600
    
    return plugins.CreateImageResponse(imageURL, filename, refreshRate), nil
}

// Register plugin
func init() {
    plugins.Register(&MyPlugin{})
}
```

**For Data Plugin:**
```go
// internal/plugins/my_plugin/my_plugin.go
package my_plugin

import "github.com/rmitchellscott/stationmaster/internal/plugins"

type MyDataPlugin struct{}

// Implement Plugin interface
func (p *MyDataPlugin) Type() string { return "my_data_plugin" }
func (p *MyDataPlugin) PluginType() plugins.PluginType { return plugins.PluginTypeData }
// ... other Plugin methods

// Implement DataPlugin interface
func (p *MyDataPlugin) RenderTemplate() string {
    return `
    <div style="text-align: center; padding: 20px;">
        <h1>{{.title}}</h1>
        <p>{{.message}}</p>
    </div>
    `
}

func (p *MyDataPlugin) DataSchema() string {
    return `{
        "type": "object", 
        "properties": {
            "title": {"type": "string"},
            "message": {"type": "string"}
        }
    }`
}

func (p *MyDataPlugin) Process(ctx plugins.PluginContext) (plugins.PluginResponse, error) {
    data := map[string]interface{}{
        "title": "Hello World",
        "message": "This is rendered server-side!",
    }
    
    return plugins.CreateDataResponse(data, p.RenderTemplate(), 3600), nil
}

func init() {
    plugins.Register(&MyDataPlugin{})
}
```

### 3. Add Import to main.go
```go
import (
    // ... existing imports
    _ "github.com/rmitchellscott/stationmaster/internal/plugins/my_plugin"
)
```

### 4. Test Your Plugin
```bash
go test ./internal/plugins
go build -o /dev/null .
```

## Converting TRMNL Plugins

To convert a plugin from [TRMNL's open source library](https://github.com/usetrmnl/plugins):

1. **Study the Ruby plugin's `locals` method** to understand data structure
2. **Create equivalent Go data structures**
3. **Convert API calls** from Ruby HTTP libraries to Go
4. **Translate ERB templates** to Go HTML templates
5. **Implement caching** if the original plugin uses it
6. **Set appropriate refresh rates** based on data update frequency

### Example: Weather Plugin

**Original Ruby structure:**
```ruby
def locals
  {
    temperature: weather_data["temp"],
    condition: weather_data["condition"],
    location: settings["location"]
  }
end
```

**Go equivalent:**
```go
func (p *WeatherPlugin) Process(ctx plugins.PluginContext) (plugins.PluginResponse, error) {
    location := ctx.GetStringSetting("location", "")
    weatherData, err := p.fetchWeather(location)
    if err != nil {
        return plugins.CreateErrorResponse(err.Error()), err
    }
    
    data := map[string]interface{}{
        "temperature": weatherData.Temperature,
        "condition":   weatherData.Condition, 
        "location":    location,
    }
    
    return plugins.CreateDataResponse(data, p.RenderTemplate(), 3600), nil
}
```

## Available Helper Functions

### Plugin Context Helpers
- `ctx.GetStringSetting(key, fallback)`
- `ctx.GetIntSetting(key, fallback)`
- `ctx.GetBoolSetting(key, fallback)`
- `ctx.GetFloatSetting(key, fallback)`
- `ctx.HasSetting(key)`

### Response Helpers
- `plugins.CreateImageResponse(imageURL, filename, refreshRate)`
- `plugins.CreateDataResponse(data, template, refreshRate)`
- `plugins.CreateErrorResponse(message)`

### Template Functions
Templates have access to helper functions:
- Date/time: `now`, `formatDate`, `formatTime`
- String: `upper`, `lower`, `title`, `trim`, `contains`, `replace`
- Math: `add`, `subtract`, `multiply`, `divide`, `round`, `percent`
- Formatting: `currency`, `number`, `bytes`

## Plugin Registry

The plugin registry is automatically populated when plugins are imported. Access via:

- `plugins.Get(type)` - Get specific plugin
- `plugins.GetAllTypes()` - Get all plugin types
- `plugins.GetAllInfo()` - Get all plugin metadata
- `plugins.Count()` - Get total plugin count

## API Endpoints

New endpoints are available for plugin management:

- `GET /api/plugins/info` - Detailed plugin information
- `GET /api/plugins/types` - Available plugin types
- `GET /api/plugins/types/:type` - Specific plugin info
- `POST /api/plugins/validate` - Validate plugin settings
- `GET /api/plugins/registry/stats` - Registry statistics

## Rendering System

Data plugins use a server-side HTML-to-image rendering system:

- **Renderer**: Headless Chrome via `go-rod/rod`
- **Templates**: Go HTML templates with helper functions
- **Output**: PNG images optimized for TRMNL devices
- **Storage**: Images cached in `/static/rendered/`
- **Options**: Configurable size, quality, DPI

The system automatically handles device-specific screen dimensions and generates high-quality images suitable for e-ink displays.