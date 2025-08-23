# Template Reference

Complete reference for creating private plugin templates using Liquid syntax and the TRMNL CSS framework.

## 📋 Table of Contents

- [Liquid Template Syntax](#liquid-template-syntax)
- [Available Variables](#available-variables)
- [TRMNL CSS Framework](#trmnl-css-framework)
- [Layout Guidelines](#layout-guidelines)
- [Security Restrictions](#security-restrictions)
- [Best Practices](#best-practices)
- [Common Patterns](#common-patterns)

## 💧 Liquid Template Syntax

Liquid is a templating language that allows you to insert dynamic content into HTML templates.

### Variables and Output

Display variables using double curly braces:

```liquid
{{ variable_name }}
{{ object.property }}
{{ array[0] }}
```

**Examples:**
```liquid
Hello {{ user.first_name }}!
Temperature: {{ data.temperature }}
Device: {{ device.name }}
```

### Filters

Transform data using filters with the pipe `|` symbol:

```liquid
{{ variable | filter }}
{{ variable | filter: parameter }}
{{ variable | filter1 | filter2 }}
```

#### String Filters
```liquid
{{ "hello world" | upcase }}           <!-- HELLO WORLD -->
{{ "HELLO WORLD" | downcase }}         <!-- hello world -->
{{ "hello world" | capitalize }}       <!-- Hello world -->
{{ "  hello  " | strip }}              <!-- hello -->
{{ "hello world" | truncate: 8 }}      <!-- hello... -->
{{ "hello world" | replace: "world", "universe" }}  <!-- hello universe -->
{{ text | escape }}                    <!-- HTML escaped -->
```

#### Number Filters
```liquid
{{ 42.7 | round }}                     <!-- 43 -->
{{ 42.7 | round: 1 }}                  <!-- 42.7 -->
{{ 42 | plus: 8 }}                     <!-- 50 -->
{{ 50 | minus: 8 }}                    <!-- 42 -->
{{ 6 | times: 7 }}                     <!-- 42 -->
{{ 84 | divided_by: 2 }}               <!-- 42 -->
```

#### Date Filters
```liquid
{{ timestamp | date: "%B %d, %Y" }}    <!-- March 15, 2024 -->
{{ timestamp | date: "%I:%M %p" }}     <!-- 2:30 PM -->
{{ timestamp | date: "%A" }}           <!-- Friday -->
```

Common date format codes:
- `%Y` - 4-digit year (2024)
- `%y` - 2-digit year (24)
- `%B` - Full month name (March)
- `%b` - Abbreviated month (Mar)
- `%m` - Month number (03)
- `%d` - Day of month (15)
- `%A` - Full weekday (Friday)
- `%a` - Abbreviated weekday (Fri)
- `%I` - Hour 12-hour (02)
- `%H` - Hour 24-hour (14)
- `%M` - Minutes (30)
- `%p` - AM/PM

#### Array Filters
```liquid
{{ array | size }}                     <!-- Array length -->
{{ array | first }}                    <!-- First item -->
{{ array | last }}                     <!-- Last item -->
{{ array | join: ", " }}               <!-- Join with separator -->
{{ array | sort }}                     <!-- Sort items -->
{{ array | reverse }}                  <!-- Reverse order -->
```

### Control Flow

#### Conditionals
```liquid
{% if condition %}
  Content when true
{% elsif other_condition %}
  Content when other condition true
{% else %}
  Content when false
{% endif %}
```

**Comparison Operators:**
- `==` - equals
- `!=` - not equals
- `>` - greater than
- `<` - less than
- `>=` - greater than or equal
- `<=` - less than or equal
- `contains` - string/array contains

**Logical Operators:**
- `and` - logical AND
- `or` - logical OR

**Examples:**
```liquid
{% if user.first_name %}
  <h1>Hello {{ user.first_name }}!</h1>
{% else %}
  <h1>Hello there!</h1>
{% endif %}

{% if data.temperature > 75 %}
  <p class="text--hot">It's hot! 🔥</p>
{% elsif data.temperature > 60 %}
  <p class="text--warm">Nice weather ☀️</p>
{% else %}
  <p class="text--cold">Bundle up ❄️</p>
{% endif %}

{% if user.first_name and user.first_name != "" %}
  <p>Welcome back, {{ user.first_name }}!</p>
{% endif %}
```

#### Loops
```liquid
{% for item in array %}
  {{ item }}
{% endfor %}
```

**Loop Variables:**
- `forloop.index` - Current iteration (1-indexed)
- `forloop.index0` - Current iteration (0-indexed)  
- `forloop.first` - True on first iteration
- `forloop.last` - True on last iteration
- `forloop.length` - Total iterations

**Examples:**
```liquid
{% for news in data.news_items %}
  <div class="news-item">
    <h3>{{ news.title }}</h3>
    <p>{{ news.content | truncate: 100 }}</p>
    {% if forloop.last %}
      <hr class="divider">
    {% endif %}
  </div>
{% endfor %}

{% for item in data.list limit: 5 %}
  <li>{{ forloop.index }}. {{ item.name }}</li>
{% endfor %}
```

#### Case/When
```liquid
{% case variable %}
  {% when 'value1' %}
    Content for value1
  {% when 'value2' %}
    Content for value2  
  {% else %}
    Default content
{% endcase %}
```

**Example:**
```liquid
{% case data.weather %}
  {% when 'sunny' %}
    <div class="weather sunny">☀️ Sunny</div>
  {% when 'rainy' %}
    <div class="weather rainy">🌧️ Rainy</div>
  {% when 'cloudy' %}
    <div class="weather cloudy">☁️ Cloudy</div>
  {% else %}
    <div class="weather unknown">🤷 Unknown</div>
{% endcase %}
```

### Variable Assignment
```liquid
{% assign variable_name = value %}
{% assign full_name = user.first_name | append: " " | append: user.last_name %}
```

### Comments
```liquid
{% comment %}
This is a comment that won't appear in the output.
Useful for documentation and notes.
{% endcomment %}
```

## 📊 Available Variables

### User Context
Information about the current user:

```liquid
{{ user.first_name }}        <!-- User's first name -->
{{ user.email }}             <!-- User's email address -->
{{ user.id }}                <!-- User's unique ID -->
```

### Device Context  
Information about the TRMNL device:

```liquid
{{ device.name }}            <!-- Device display name -->
{{ device.id }}              <!-- Device unique ID -->
{{ device.width }}           <!-- Screen width in pixels -->
{{ device.height }}          <!-- Screen height in pixels -->
{{ device.model }}           <!-- Device model name -->
```

### Plugin Context
Information about the plugin instance:

```liquid
{{ instance_id }}            <!-- Unique plugin instance ID -->
{{ plugin_name }}            <!-- Name of the plugin -->
{{ refresh_rate }}           <!-- Configured refresh rate -->
```

### Timestamp
Current timestamp in various formats:

```liquid
{{ timestamp }}              <!-- Full ISO timestamp -->
{{ timestamp | date: "%I:%M %p" }}  <!-- 2:30 PM -->
{{ timestamp | date: "%B %d" }}     <!-- March 15 -->
```

### Data Variables
Content varies by data strategy:

#### Webhook Data
All webhook payload data is available under `data.*`:

```liquid
{{ data.temperature }}       <!-- From webhook: {"temperature": "72°F"} -->
{{ data.status }}            <!-- From webhook: {"status": "online"} -->
{{ data.sensors.living_room }}  <!-- Nested objects supported -->
```

#### Polling Data  
Polled API responses available under `data.*`:

```liquid
{{ data.weather.current }}   <!-- From polled weather API -->
{{ data.news[0].title }}     <!-- First news item title -->
```

#### Merge Data
Access to user, device, and system information (no external data).

## 🎨 TRMNL CSS Framework

The TRMNL CSS framework provides responsive, e-ink optimized styles automatically included in all plugins.

### Layout Classes

#### View Container Classes
Required container classes for proper layout:

```html
<!-- Full screen layout -->
<div class="view--full">...</div>

<!-- Half vertical layout -->  
<div class="view--half_vertical">...</div>

<!-- Half horizontal layout -->
<div class="view--half_horizontal">...</div>

<!-- Quadrant layout -->
<div class="view--quadrant">...</div>
```

### Typography

#### Text Size Classes
```html
<h1 class="text--huge">Huge Text</h1>
<h2 class="text--large">Large Text</h2>  
<h3 class="text--medium">Medium Text</h3>
<p class="text--normal">Normal Text</p>
<p class="text--small">Small Text</p>
<p class="text--tiny">Tiny Text</p>
```

#### Text Style Classes  
```html
<p class="text--bold">Bold Text</p>
<p class="text--italic">Italic Text</p>
<p class="text--underline">Underlined Text</p>
<p class="text--caps">UPPERCASE TEXT</p>
```

### Utility Classes

#### Spacing Classes
```html
<!-- Margins -->
<div class="u--space-all">All sides margin</div>
<div class="u--space-top">Top margin only</div>  
<div class="u--space-bottom">Bottom margin only</div>
<div class="u--space-sides">Left & right margin</div>

<!-- Padding -->
<div class="u--pad-all">All sides padding</div>
<div class="u--pad-top">Top padding only</div>
<div class="u--pad-bottom">Bottom padding only</div>
<div class="u--pad-sides">Left & right padding</div>

<!-- Size variants -->
<div class="u--space-all-small">Small margin</div>
<div class="u--space-all-large">Large margin</div>
```

#### Alignment Classes
```html
<div class="u--align-left">Left aligned</div>
<div class="u--align-center">Center aligned</div>  
<div class="u--align-right">Right aligned</div>
<div class="u--align-justify">Justified</div>

<!-- Vertical alignment -->
<div class="u--valign-top">Top aligned</div>
<div class="u--valign-middle">Middle aligned</div>
<div class="u--valign-bottom">Bottom aligned</div>
```

#### Display Classes
```html
<div class="u--hidden">Hidden element</div>
<div class="u--visible">Visible element</div>
<div class="u--block">Block display</div>
<div class="u--inline">Inline display</div>
<div class="u--inline-block">Inline-block display</div>
```

### Grid and Layout

#### Flex Layout
```html
<div class="flex-horizontal">
  <div class="flex-item">Item 1</div>
  <div class="flex-item">Item 2</div>  
  <div class="flex-item">Item 3</div>
</div>

<div class="flex-vertical">
  <div class="flex-item">Item 1</div>
  <div class="flex-item">Item 2</div>
</div>

<!-- Flex item sizing -->
<div class="flex-horizontal">
  <div class="flex-item flex-grow">Grows to fill</div>
  <div class="flex-item flex-shrink">Shrinks as needed</div>
</div>
```

#### Grid Layout
```html
<div class="grid-2-columns">
  <div class="column">Column 1</div>
  <div class="column">Column 2</div>
</div>

<div class="grid-3-columns">
  <div class="column">Column 1</div>
  <div class="column">Column 2</div>  
  <div class="column">Column 3</div>
</div>

<div class="grid-4-columns">
  <div class="column">Column 1</div>
  <div class="column">Column 2</div>
  <div class="column">Column 3</div>
  <div class="column">Column 4</div>
</div>
```

### Components

#### Cards and Panels
```html
<div class="card">
  <div class="card-header">
    <h3 class="card-title">Card Title</h3>
  </div>
  <div class="card-body">
    <p>Card content goes here.</p>
  </div>
</div>

<div class="panel">
  <div class="panel-content">
    <p>Panel content</p>
  </div>
</div>
```

#### Lists  
```html
<ul class="list--clean">
  <li class="list-item">Clean list item</li>
  <li class="list-item">No bullets</li>
</ul>

<ul class="list--spaced">
  <li class="list-item">Spaced list item</li>
  <li class="list-item">More spacing</li>
</ul>
```

#### Buttons (Visual Only)
```html
<div class="button">Button Style</div>
<div class="button button--primary">Primary Button</div>
<div class="button button--secondary">Secondary Button</div>
<div class="button button--small">Small Button</div>
<div class="button button--large">Large Button</div>
```

### Icons and Graphics

#### Icon Classes
```html
<div class="icon--small">📊</div>
<div class="icon--medium">📈</div>  
<div class="icon--large">📋</div>
<div class="icon--huge">🎯</div>

<!-- Icon positioning -->
<div class="icon-with-text">
  <span class="icon">📊</span>
  <span class="text">Statistics</span>
</div>
```

#### Borders and Dividers
```html
<div class="border">Bordered content</div>
<div class="border-top">Top border only</div>
<div class="border-bottom">Bottom border only</div>

<hr class="divider">
<hr class="divider--thick">
<hr class="divider--dashed">
```

## 📐 Layout Guidelines

### Required Container Structure

Every plugin template must include a containerized structure:

```html
<div id="plugin-{{ instance_id }}" class="plugin-container view--LAYOUT">
  <!-- Your plugin content here -->
</div>
```

**Key Requirements:**
- **Unique ID**: `plugin-{{ instance_id }}` ensures CSS isolation
- **Plugin Container**: `.plugin-container` class for base styling
- **View Class**: Appropriate view class for layout type

### Layout-Specific Best Practices

#### Full Screen (800×480)
```html
<div id="plugin-{{ instance_id }}" class="plugin-container view--full">
  <div class="u--space-all">
    <!-- Rich content with multiple sections -->
    <header class="u--space-bottom">
      <h1 class="text--huge">Dashboard Title</h1>
    </header>
    
    <main class="grid-2-columns">
      <section class="column">
        <!-- Primary content -->
      </section>
      <aside class="column">
        <!-- Secondary content -->  
      </aside>
    </main>
  </div>
</div>
```

#### Half Vertical (400×480)
```html
<div id="plugin-{{ instance_id }}" class="plugin-container view--half_vertical">
  <div class="u--space-all u--align-center">
    <!-- Vertical stack of content -->
    <div class="u--space-bottom">
      <h2 class="text--large">Widget Title</h2>
    </div>
    
    <div class="u--space-bottom">
      <div class="text--huge">{{ data.value }}</div>
    </div>
    
    <div class="text--small">
      {{ data.description }}
    </div>
  </div>
</div>
```

#### Half Horizontal (800×240)  
```html
<div id="plugin-{{ instance_id }}" class="plugin-container view--half_horizontal">
  <div class="flex-horizontal u--space-sides u--valign-middle">
    <!-- Horizontal layout -->
    <div class="flex-item">
      <h3 class="text--medium">Status</h3>
    </div>
    <div class="flex-item flex-grow u--align-center">
      <span class="text--large">{{ data.status }}</span>
    </div>
    <div class="flex-item">
      <span class="text--small">{{ timestamp | date: "%I:%M %p" }}</span>
    </div>
  </div>
</div>
```

#### Quadrant (400×240)
```html
<div id="plugin-{{ instance_id }}" class="plugin-container view--quadrant">
  <div class="u--space-all u--align-center">
    <!-- Compact, centered content -->
    <div class="u--space-bottom">
      <div class="icon--large">{{ data.icon | default: "📊" }}</div>
    </div>
    <div class="text--medium">{{ data.value }}</div>
    <div class="text--small">{{ data.label }}</div>
  </div>
</div>
```

## 🔒 Security Restrictions

### Blocked Elements
The following HTML elements are not allowed for security:

```html
<!-- NOT ALLOWED -->
<script>...</script>                 <!-- JavaScript execution -->
<iframe src="...">                   <!-- External content embedding -->
<object>...</object>                 <!-- Plugin objects -->
<embed>...</embed>                   <!-- Embedded content -->
<link rel="stylesheet" href="...">   <!-- External stylesheets -->
```

### Blocked Attributes
Certain attributes are restricted:

```html
<!-- NOT ALLOWED -->
<div onclick="...">                  <!-- JavaScript event handlers -->
<a href="javascript:...">            <!-- JavaScript URLs -->
<div style="background: url(http://...)"> <!-- External URLs in CSS -->
```

### Allowed External Resources
Only these external resources are permitted:

- **Data URLs**: `data:image/png;base64,...`
- **Fragment URLs**: `#section-id`
- **TRMNL Framework**: Automatically included CSS/JS

### Content Security Policy
Templates are subject to strict CSP:

- **No inline scripts**: JavaScript execution blocked
- **No external scripts**: Only TRMNL framework scripts allowed
- **No external styles**: Only TRMNL framework CSS allowed
- **Limited external images**: Only data URLs permitted

## ✨ Best Practices

### Template Organization
```html
<div id="plugin-{{ instance_id }}" class="plugin-container view--full">
  <!-- Header Section -->
  <header class="plugin-header u--space-bottom">
    <h1 class="text--large">{{ plugin_name }}</h1>
  </header>
  
  <!-- Main Content -->
  <main class="plugin-content">
    {% if data %}
      <!-- Content when data available -->
      <div class="data-display">
        {{ data.content }}
      </div>
    {% else %}
      <!-- Fallback when no data -->
      <div class="u--align-center">
        <p class="text--medium">No data available</p>
      </div>
    {% endif %}
  </main>
  
  <!-- Footer Section -->
  <footer class="plugin-footer u--space-top">
    <p class="text--small">Updated: {{ timestamp | date: "%I:%M %p" }}</p>
  </footer>
</div>
```

### Error Handling
```liquid
{% if data.error %}
  <div class="error-message u--align-center">
    <div class="icon--large">⚠️</div>
    <p class="text--medium">{{ data.error }}</p>
  </div>
{% elsif data %}
  <!-- Normal content -->
{% else %}
  <div class="loading-message u--align-center">
    <p class="text--medium">Loading...</p>
  </div>
{% endif %}
```

### Responsive Design
```html
<!-- Use utility classes for responsive behavior -->
<div class="u--space-all">
  <!-- Content adapts to container -->
  <div class="flex-horizontal">
    <div class="flex-item">Item 1</div>
    <div class="flex-item">Item 2</div>
  </div>
</div>

<!-- Conditional content based on layout -->
{% if device.width >= 800 %}
  <!-- Full or half-horizontal layout content -->
  <div class="grid-3-columns">...</div>
{% else %}
  <!-- Half-vertical or quadrant content -->
  <div class="u--align-center">...</div>
{% endif %}
```

### Performance Optimization
```liquid
<!-- Limit loops to prevent performance issues -->
{% for item in data.items limit: 10 %}
  <div class="item">{{ item.name }}</div>
{% endfor %}

<!-- Cache expensive operations -->
{% assign formatted_date = timestamp | date: "%B %d, %Y" %}
<p>Date: {{ formatted_date }}</p>

<!-- Minimize nested conditions -->
{% assign status_class = "status-unknown" %}
{% if data.status == "online" %}
  {% assign status_class = "status-online" %}
{% elsif data.status == "offline" %}
  {% assign status_class = "status-offline" %}
{% endif %}
<div class="{{ status_class }}">{{ data.status | upcase }}</div>
```

## 🎯 Common Patterns

### Weather Display
```html
<div id="plugin-{{ instance_id }}" class="plugin-container view--full">
  <div class="weather-display u--space-all u--align-center">
    <div class="current-temp text--huge u--space-bottom">
      {{ data.temperature }}°
    </div>
    
    <div class="weather-condition text--large u--space-bottom">
      {% case data.condition %}
        {% when "sunny" %}☀️ Sunny
        {% when "cloudy" %}☁️ Cloudy  
        {% when "rainy" %}🌧️ Rainy
        {% else %}{{ data.condition }}
      {% endcase %}
    </div>
    
    <div class="weather-details grid-2-columns">
      <div class="column u--align-center">
        <div class="text--small">Humidity</div>
        <div class="text--medium">{{ data.humidity }}%</div>
      </div>
      <div class="column u--align-center">
        <div class="text--small">Wind</div>  
        <div class="text--medium">{{ data.wind }} mph</div>
      </div>
    </div>
  </div>
</div>
```

### Status Dashboard
```html
<div id="plugin-{{ instance_id }}" class="plugin-container view--full">
  <div class="dashboard u--space-all">
    <header class="dashboard-header u--space-bottom">
      <h1 class="text--large">System Status</h1>
      <p class="text--small">Last updated: {{ timestamp | date: "%I:%M %p" }}</p>
    </header>
    
    <main class="status-grid grid-2-columns">
      {% for service in data.services %}
        <div class="status-item column u--pad-all border">
          <div class="service-name text--medium u--space-bottom">
            {{ service.name }}
          </div>
          
          <div class="service-status">
            {% if service.status == "online" %}
              <span class="status-indicator">🟢</span>
              <span class="text--small">Online</span>
            {% else %}
              <span class="status-indicator">🔴</span>
              <span class="text--small">Offline</span>
            {% endif %}
          </div>
          
          {% if service.last_check %}
            <div class="last-check text--tiny u--space-top">
              Checked: {{ service.last_check | date: "%I:%M %p" }}
            </div>
          {% endif %}
        </div>
      {% endfor %}
    </main>
  </div>
</div>
```

### News Feed
```html
<div id="plugin-{{ instance_id }}" class="plugin-container view--half_vertical">
  <div class="news-feed u--space-all">
    <header class="feed-header u--space-bottom u--align-center">
      <h2 class="text--large">Latest News</h2>
    </header>
    
    <main class="news-items">
      {% for article in data.articles limit: 5 %}
        <article class="news-item u--space-bottom border-bottom">
          <h3 class="article-title text--medium u--space-bottom-small">
            {{ article.title | truncate: 60 }}
          </h3>
          
          <p class="article-summary text--small u--space-bottom-small">
            {{ article.summary | truncate: 120 }}
          </p>
          
          <div class="article-meta text--tiny">
            <span class="source">{{ article.source }}</span>
            <span class="time">{{ article.published_at | date: "%I:%M %p" }}</span>
          </div>
        </article>
      {% endfor %}
    </main>
  </div>
</div>
```

### Metric Widgets
```html
<div id="plugin-{{ instance_id }}" class="plugin-container view--quadrant">
  <div class="metric-widget u--space-all u--align-center">
    <div class="metric-icon u--space-bottom">
      {{ data.icon | default: "📊" }}
    </div>
    
    <div class="metric-value text--huge u--space-bottom">
      {{ data.value }}
      {% if data.unit %}<span class="text--small">{{ data.unit }}</span>{% endif %}
    </div>
    
    <div class="metric-label text--medium u--space-bottom">
      {{ data.label }}
    </div>
    
    {% if data.change %}
      <div class="metric-change text--small">
        {% if data.change > 0 %}
          <span class="change-positive">↗ +{{ data.change }}%</span>
        {% else %}
          <span class="change-negative">↘ {{ data.change }}%</span>
        {% endif %}
      </div>
    {% endif %}
  </div>
</div>
```

### Calendar Events
```html
<div id="plugin-{{ instance_id }}" class="plugin-container view--half_horizontal">
  <div class="calendar-events flex-horizontal u--space-sides u--valign-middle">
    <div class="event-header flex-item">
      <h3 class="text--medium">Today's Events</h3>
    </div>
    
    <div class="event-list flex-item flex-grow">
      {% if data.events.size > 0 %}
        {% for event in data.events limit: 3 %}
          <div class="event-item u--space-sides">
            <span class="event-time text--small">{{ event.time }}</span>
            <span class="event-title text--medium">{{ event.title | truncate: 30 }}</span>
          </div>
        {% endfor %}
      {% else %}
        <div class="no-events u--align-center">
          <span class="text--medium">No events today</span>
        </div>
      {% endif %}
    </div>
    
    <div class="current-time flex-item">
      <div class="time-display text--large">
        {{ timestamp | date: "%I:%M" }}
      </div>
      <div class="time-period text--small">
        {{ timestamp | date: "%p" }}
      </div>
    </div>
  </div>
</div>
```

---

*This reference covers the complete template system. Refer to [examples](./examples/) for more complex implementations and [API documentation](./API.md) for technical integration details.*