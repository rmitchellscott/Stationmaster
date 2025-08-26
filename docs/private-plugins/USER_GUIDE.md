# Private Plugin User Guide

This guide walks you through creating, configuring, and managing private plugins in Stationmaster. By the end, you'll be able to create custom content for your TRMNL devices using liquid templates and the TRMNL design framework.

## 📋 Table of Contents

- [Getting Started](#getting-started)
- [Creating Your First Plugin](#creating-your-first-plugin)
- [Understanding Liquid Templates](#understanding-liquid-templates)
- [Working with Layouts](#working-with-layouts)
- [Data Strategies](#data-strategies)
- [Template Editor Features](#template-editor-features)
- [Testing and Validation](#testing-and-validation)
- [Publishing and Management](#publishing-and-management)
- [Troubleshooting](#troubleshooting)

## 🚀 Getting Started

### Prerequisites
- Stationmaster instance with private plugin system enabled
- User account with plugin creation permissions
- Basic understanding of HTML and CSS (helpful but not required)

### Accessing the Plugin System
1. Log into your Stationmaster dashboard
2. Navigate to **Plugin Management** in the main menu
3. Click the **Private Plugins** tab
4. You'll see your existing private plugins or a welcome message

## 🎨 Creating Your First Plugin

Let's create a simple "Hello World" plugin to get familiar with the system.

### Step 1: Start Plugin Creation
1. Click **"Create Private Plugin"** button
2. The plugin editor dialog will open
3. You'll see tabs for different layout templates

### Step 2: Basic Information
Fill out the plugin details:

```
Name: My First Plugin
Description: A simple hello world plugin
Version: 1.0.0
Data Strategy: Merge (simplest for first plugin)
```

### Step 3: Create Your Template
In the **Shared Markup** tab, add:

```html
<div id="plugin-{{ instance_id }}" class="plugin-container view--full">
  <div class="u--align-center u--space-all">
    <h1 class="text--huge u--space-bottom">Hello, World! 👋</h1>
    <p class="text--large">Welcome to Private Plugins</p>
    <div class="u--space-top">
      <p class="text--medium">Device: {{ device.name }}</p>
      <p class="text--small">Time: {{ timestamp }}</p>
    </div>
  </div>
</div>
```

### Step 4: Test Your Plugin
1. Click **"Validate Templates"** - should show green checkmark
2. Click **"Preview"** to see how it looks
3. Try different layouts in the preview dropdown

### Step 5: Save and Use
1. Click **"Create Plugin"** 
2. Go to **Plugin Instances** tab
3. Create a new instance using your private plugin
4. Add it to a device playlist

Congratulations! You've created your first private plugin.

## 💧 Understanding Liquid Templates

Liquid is a templating language that lets you insert dynamic content into HTML. Here are the key concepts:

### Variables
Display data using double curly braces:
```liquid
{{ user.first_name }}
{{ device.name }}
{{ data.temperature }}
```

### Filters
Transform data using the pipe symbol:
```liquid
{{ user.first_name | upcase }}
{{ timestamp | date: "%B %d, %Y" }}
{{ data.price | currency }}
```

Common filters:
- `upcase` / `downcase` - Change text case
- `capitalize` - Capitalize first letter
- `strip` - Remove whitespace
- `truncate: 50` - Limit text length
- `date: "%H:%M"` - Format dates/times

### Conditionals
Show content based on conditions:
```liquid
{% if data.temperature > 75 %}
  <p class="text--hot">It's hot outside! 🔥</p>
{% elsif data.temperature > 60 %}
  <p class="text--warm">Nice weather! ☀️</p>
{% else %}
  <p class="text--cold">Bundle up! ❄️</p>
{% endif %}
```

### Loops
Repeat content for lists:
```liquid
{% for item in data.news_items %}
  <div class="news-item u--space-bottom">
    <h3>{{ item.title }}</h3>
    <p>{{ item.summary }}</p>
  </div>
{% endfor %}
```

### Comments
Add notes that won't appear in output:
```liquid
{% comment %}
This is a comment - useful for documentation
{% endcomment %}
```

## 📱 Working with Layouts

Private plugins support all TRMNL layout types. Each requires different design considerations:

### Layout Types

#### Full Screen (800×480)
Best for: Dashboards, detailed information, rich content
```html
<div id="plugin-{{ instance_id }}" class="plugin-container view--full">
  <!-- Full canvas available - use generously -->
  <div class="grid-2-columns">
    <div class="column">
      <h1 class="text--huge">Main Content</h1>
      <!-- Rich, detailed content -->
    </div>
    <div class="column">
      <!-- Secondary content -->
    </div>
  </div>
</div>
```

#### Half Vertical (400×480)
Best for: Side panels, narrow widgets, supplementary info
```html
<div id="plugin-{{ instance_id }}" class="plugin-container view--half_vertical">
  <!-- Narrow but tall - stack content vertically -->
  <div class="u--align-center u--space-all">
    <h2 class="text--large">Widget Title</h2>
    <div class="u--space-top">
      <!-- Compact vertical layout -->
    </div>
  </div>
</div>
```

#### Half Horizontal (800×240)
Best for: Status bars, tickers, horizontal information
```html
<div id="plugin-{{ instance_id }}" class="plugin-container view--half_horizontal">
  <!-- Wide but short - use horizontal layout -->
  <div class="flex-horizontal u--space-sides">
    <div class="flex-item">Content 1</div>
    <div class="flex-item">Content 2</div>
    <div class="flex-item">Content 3</div>
  </div>
</div>
```

#### Quadrant (400×240)
Best for: Small widgets, icons, summary information
```html
<div id="plugin-{{ instance_id }}" class="plugin-container view--quadrant">
  <!-- Small space - be concise -->
  <div class="u--align-center u--space-all">
    <div class="icon--large">📊</div>
    <h3 class="text--medium">{{ data.value }}</h3>
    <p class="text--small">{{ data.label }}</p>
  </div>
</div>
```

### Layout-Specific Templates

You can create different templates for each layout:

1. **Shared Markup**: Used by all layouts if specific template not provided
2. **Full Layout**: Custom template for full screen
3. **Half Vertical**: Custom template for half vertical
4. **Half Horizontal**: Custom template for half horizontal  
5. **Quadrant**: Custom template for quadrant

### Best Practices for Layouts

- **Test all sizes**: Preview your plugin in every layout
- **Be responsive**: Use TRMNL's utility classes for spacing
- **Consider readability**: Ensure text is readable at all sizes
- **Use appropriate detail**: More detail for larger layouts

## 📊 Data Strategies

Choose the right data strategy for your use case:

### Merge Strategy (Recommended for Beginners)
Perfect for: Static content, user information, device data

**Available Data:**
```liquid
{{ user.first_name }}          <!-- User's name -->
{{ user.email }}               <!-- User's email -->
{{ device.name }}              <!-- Device name -->
{{ device.width }}             <!-- Screen width -->
{{ device.height }}            <!-- Screen height -->
{{ timestamp }}                <!-- Current timestamp -->
{{ instance_id }}              <!-- Unique plugin instance ID -->
```

**Example Template:**
```html
<div id="plugin-{{ instance_id }}" class="plugin-container">
  <h1>Welcome {{ user.first_name }}!</h1>
  <p>Your {{ device.name }} is online</p>
  <p>Last updated: {{ timestamp | date: "%I:%M %p" }}</p>
</div>
```

### Webhook Strategy
Perfect for: Real-time data, IoT sensors, push notifications

**How it Works:**
1. Create plugin with webhook strategy
2. Get your unique webhook URL
3. Send POST requests with JSON data
4. Data appears in templates as `data.*`

**Setting Up Webhooks:**
1. Select "Webhook" data strategy
2. Save plugin to generate webhook URL
3. Copy webhook URL from plugin list
4. Send HTTP POST requests to URL

**Example Webhook Payload:**
```json
POST /api/webhooks/plugin/your-token-here
Content-Type: application/json

{
  "temperature": "72°F",
  "humidity": "45%",
  "location": "Living Room",
  "status": "comfortable",
  "timestamp": "2024-03-15T10:30:00Z"
}
```

**Template Using Webhook Data:**
```html
<div id="plugin-{{ instance_id }}" class="plugin-container">
  <h2>{{ data.location }} Status</h2>
  <div class="stats-grid">
    <div class="stat">
      <span class="stat-label">Temperature</span>
      <span class="stat-value text--large">{{ data.temperature }}</span>
    </div>
    <div class="stat">
      <span class="stat-label">Humidity</span>
      <span class="stat-value text--large">{{ data.humidity }}</span>
    </div>
  </div>
  <p class="text--small">Status: {{ data.status | upcase }}</p>
</div>
```

### Polling Strategy  
Perfect for: APIs, RSS feeds, scheduled data updates

**Configuration:**
```json
{
  "urls": [
    {
      "url": "https://api.weather.com/current",
      "method": "GET",
      "headers": {
        "Authorization": "Bearer your-api-key",
        "User-Agent": "Stationmaster/1.0"
      },
      "interval": 1800
    }
  ],
  "merge_data": true,
  "timeout": 30
}
```

**Template Using Polled Data:**
```html
<div id="plugin-{{ instance_id }}" class="plugin-container">
  <h2>Current Weather</h2>
  <div class="weather-display u--align-center">
    <div class="temperature text--huge">
      {{ data.temperature }}°
    </div>
    <div class="condition text--large">
      {{ data.condition }}
    </div>
    <div class="details u--space-top">
      <p>Humidity: {{ data.humidity }}%</p>
      <p>Wind: {{ data.wind_speed }} mph</p>
    </div>
  </div>
</div>
```

## ⚙️ Template Editor Features

The Monaco editor provides powerful features for template development:

### Syntax Highlighting
- **Liquid tags**: `{% %}` highlighted in blue
- **Variables**: `{{ }}` highlighted in green  
- **HTML**: Standard HTML syntax highlighting
- **CSS**: Style blocks properly highlighted

### Auto-completion
- **TRMNL CSS classes**: Auto-complete framework classes
- **Liquid syntax**: Built-in liquid tag suggestions
- **HTML tags**: Standard HTML completion

### Error Detection
- **Liquid syntax errors**: Immediately highlighted
- **HTML validation**: Malformed tags detected
- **Missing variables**: Warnings for undefined variables

### Editor Shortcuts
- `Ctrl/Cmd + S`: Save plugin
- `Ctrl/Cmd + /`: Toggle comment
- `Ctrl/Cmd + F`: Find and replace
- `Ctrl/Cmd + Z`: Undo changes
- `Tab` / `Shift+Tab`: Indent/unindent

### Multi-tab Editing
Switch between layout templates:
- **Shared Markup**: Used by all layouts
- **Full Screen**: Full layout override
- **Half Vertical**: Half vertical override
- **Half Horizontal**: Half horizontal override  
- **Quadrant**: Quadrant override

## ✅ Testing and Validation

### Template Validation
Always validate templates before saving:

1. Click **"Validate Templates"** button
2. Review any errors or warnings
3. Fix issues before proceeding

**Common Validation Errors:**
- Missing container div with `plugin-{{ instance_id }}`
- Script tags (not allowed for security)
- External resource references
- Invalid liquid syntax
- Missing required attributes

**Validation Warnings:**
- Inline styles (use TRMNL classes instead)
- Fixed positioning (may not work on all devices)
- Missing TRMNL framework classes

### Live Preview
Test your plugin with sample data:

1. Click **Preview** tab in editor
2. Select layout to test
3. Modify sample data in JSON editor
4. See real-time rendering results

**Sample Data Structure:**
```json
{
  "user": {
    "first_name": "John",
    "email": "john@example.com"
  },
  "device": {
    "name": "My TRMNL",
    "width": 800,
    "height": 480
  },
  "data": {
    "temperature": "72°F",
    "status": "online"
  },
  "timestamp": "2024-03-15T10:30:00Z"
}
```

### Testing Checklist
Before publishing your plugin:

- [ ] Validates without errors
- [ ] Previews correctly in all layouts
- [ ] Data displays properly with sample data
- [ ] Text is readable and properly sized
- [ ] Colors work well on e-ink displays
- [ ] No animations or transitions used
- [ ] Proper containerization with unique ID

## 🚀 Publishing and Management

### Saving Your Plugin
1. Complete template development
2. Validate templates successfully
3. Test with preview feature
4. Click **"Create Plugin"** or **"Update Plugin"**

### Creating Plugin Instances
1. Go to **Plugin Instances** tab
2. Click **"Create Plugin Instance"**
3. Select your private plugin from dropdown
4. Configure refresh rate and device assignment
5. Add to device playlists

### Version Control
- Update version number when making changes
- Keep description current with features
- Test thoroughly after updates
- Monitor plugin performance

### Plugin Management
- **Edit**: Modify templates and configuration
- **Delete**: Remove plugin (removes all instances)
- **Copy**: Duplicate for variations
- **Export/Import**: Share configurations (future feature)

## 🐛 Troubleshooting

### Common Issues and Solutions

#### Plugin Not Displaying
**Symptoms**: Plugin instance created but shows blank/error
**Solutions**:
- Check template validation results
- Verify container div has correct ID format
- Test with simplified template
- Check device logs for errors

#### Data Not Appearing  
**Symptoms**: Template renders but shows no dynamic content
**Solutions**:
- Verify data strategy configuration
- Check webhook endpoint receiving data
- Test polling URLs manually
- Review sample data format

#### Layout Problems
**Symptoms**: Content cut off or poorly positioned
**Solutions**:
- Test all layout sizes in preview
- Use TRMNL utility classes for spacing
- Avoid fixed positioning
- Check container dimensions

#### Validation Errors
**Symptoms**: Templates fail validation
**Solutions**:
- Review error messages carefully
- Remove script tags and external resources
- Add required container div
- Fix liquid syntax errors

#### Performance Issues
**Symptoms**: Plugin renders slowly or fails
**Solutions**:
- Reduce template complexity
- Optimize polling intervals
- Minimize external API calls
- Use appropriate refresh rates

### Getting Help

1. **Check validation messages** for specific error details
2. **Review examples** in documentation
3. **Test with minimal templates** to isolate issues
4. **Check API logs** for webhook/polling problems
5. **Contact support** with specific error messages

### Debug Mode
Enable debug information:
1. Add `?debug=1` to plugin instance URL
2. View detailed error messages
3. Check data availability
4. Review rendering logs

## 🎯 Next Steps

Now that you understand the basics:

1. **Explore Examples**: Check out example plugins for inspiration
2. **Read Template Reference**: Deep dive into TRMNL CSS classes
3. **Experiment with Data**: Try different webhook and polling scenarios  
4. **Share Your Work**: Export successful plugins for others
5. **Advanced Features**: Look into complex liquid logic and animations

## 📚 Additional Resources

- [Template Reference](./TEMPLATE_REFERENCE.md) - Complete liquid and CSS reference
- [API Documentation](./API.md) - Technical API details
- [Examples](./examples/) - Working example plugins
- [TRMNL Framework](https://usetrmnl.com/css/latest/plugins.css) - Official CSS framework

---

*Happy plugin creation! The private plugin system opens up unlimited possibilities for customizing your TRMNL experience.*