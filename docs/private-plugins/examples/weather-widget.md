# Weather Widget Example

A complete example of a private plugin that displays weather information using the polling data strategy.

## Overview

This plugin demonstrates:
- **Polling Strategy**: Fetches data from a weather API every 30 minutes
- **Multi-layout Support**: Different templates for each screen size
- **Error Handling**: Graceful fallbacks when data is unavailable
- **TRMNL Framework**: Uses framework CSS classes for consistent styling

## Plugin Configuration

### Basic Information
```json
{
  "name": "Weather Widget",
  "description": "Display current weather conditions and forecast",
  "version": "1.0.0",
  "data_strategy": "polling"
}
```

### Polling Configuration
```json
{
  "urls": [
    {
      "url": "https://api.openweathermap.org/data/2.5/weather",
      "method": "GET",
      "headers": {
        "User-Agent": "Stationmaster Weather Widget/1.0"
      },
      "query_params": {
        "q": "New York,NY,US",
        "appid": "your-api-key-here",
        "units": "imperial"
      },
      "interval": 1800,
      "timeout": 30,
      "retry_count": 3
    }
  ],
  "cache_duration": 300
}
```

## Template Implementation

### Full Screen Layout (800×480)
```html
<div id="plugin-{{ instance_id }}" class="plugin-container view--full">
  <div class="weather-widget u--space-all">
    {% if data.cod == 200 %}
      <!-- Weather data available -->
      <header class="weather-header u--space-bottom">
        <h1 class="text--huge u--align-center">
          {{ data.name }}, {{ data.sys.country }}
        </h1>
        <p class="text--medium u--align-center">
          {{ timestamp | date: "%A, %B %d at %I:%M %p" }}
        </p>
      </header>

      <main class="weather-content grid-2-columns">
        <!-- Current Weather -->
        <section class="current-weather column">
          <div class="temperature-display u--align-center u--space-bottom">
            <div class="current-temp text--huge">
              {{ data.main.temp | round }}°F
            </div>
            <div class="feels-like text--medium">
              Feels like {{ data.main.feels_like | round }}°F
            </div>
          </div>

          <div class="weather-condition u--align-center u--space-bottom">
            <div class="condition-icon">
              {% case data.weather[0].main %}
                {% when 'Clear' %}☀️
                {% when 'Clouds' %}☁️
                {% when 'Rain' %}🌧️
                {% when 'Snow' %}❄️
                {% when 'Thunderstorm' %}⛈️
                {% when 'Drizzle' %}🌦️
                {% when 'Mist' or 'Fog' %}🌫️
                {% else %}🌤️
              {% endcase %}
            </div>
            <div class="condition-text text--large">
              {{ data.weather[0].description | capitalize }}
            </div>
          </div>
        </section>

        <!-- Weather Details -->
        <section class="weather-details column">
          <div class="details-grid">
            <div class="detail-item u--space-bottom border-bottom">
              <div class="detail-label text--small">Humidity</div>
              <div class="detail-value text--large">{{ data.main.humidity }}%</div>
            </div>

            <div class="detail-item u--space-bottom border-bottom">
              <div class="detail-label text--small">Wind</div>
              <div class="detail-value text--large">
                {{ data.wind.speed | round }} mph
                {% if data.wind.deg %}
                  {{ data.wind.deg | wind_direction }}
                {% endif %}
              </div>
            </div>

            <div class="detail-item u--space-bottom border-bottom">
              <div class="detail-label text--small">Pressure</div>
              <div class="detail-value text--large">
                {{ data.main.pressure | pressure_to_inches }} in
              </div>
            </div>

            <div class="detail-item u--space-bottom">
              <div class="detail-label text--small">Visibility</div>
              <div class="detail-value text--large">
                {% if data.visibility %}
                  {{ data.visibility | meters_to_miles }} mi
                {% else %}
                  N/A
                {% endif %}
              </div>
            </div>
          </div>
        </section>
      </main>

      <footer class="weather-footer u--space-top u--align-center">
        <p class="text--small">
          Sunrise: {{ data.sys.sunrise | date: "%I:%M %p" }} | 
          Sunset: {{ data.sys.sunset | date: "%I:%M %p" }}
        </p>
      </footer>

    {% elsif data.cod %}
      <!-- API Error Response -->
      <div class="error-message u--align-center u--space-all">
        <div class="error-icon text--huge">⚠️</div>
        <h2 class="text--large u--space-bottom">Weather Unavailable</h2>
        <p class="text--medium">{{ data.message | default: "Unable to fetch weather data" }}</p>
        <p class="text--small u--space-top">Error Code: {{ data.cod }}</p>
      </div>
    {% else %}
      <!-- No Data Available -->
      <div class="loading-message u--align-center u--space-all">
        <div class="loading-icon text--huge">🌤️</div>
        <h2 class="text--large u--space-bottom">Loading Weather...</h2>
        <p class="text--medium">Fetching latest weather conditions</p>
      </div>
    {% endif %}
  </div>
</div>
```

### Half Vertical Layout (400×480)
```html
<div id="plugin-{{ instance_id }}" class="plugin-container view--half_vertical">
  <div class="weather-compact u--space-all u--align-center">
    {% if data.cod == 200 %}
      <header class="weather-location u--space-bottom">
        <h2 class="text--large">{{ data.name }}</h2>
        <p class="text--small">{{ data.sys.country }}</p>
      </header>

      <main class="current-weather u--space-bottom">
        <div class="temp-display u--space-bottom">
          <div class="temperature text--huge">{{ data.main.temp | round }}°</div>
          <div class="condition text--medium">
            {% case data.weather[0].main %}
              {% when 'Clear' %}☀️ {{ data.weather[0].main }}
              {% when 'Clouds' %}☁️ {{ data.weather[0].main }}
              {% when 'Rain' %}🌧️ {{ data.weather[0].main }}
              {% when 'Snow' %}❄️ {{ data.weather[0].main }}
              {% else %}🌤️ {{ data.weather[0].main }}
            {% endcase %}
          </div>
        </div>

        <div class="weather-stats">
          <div class="stat-row u--space-bottom-small">
            <span class="stat-label text--small">Feels like:</span>
            <span class="stat-value text--medium">{{ data.main.feels_like | round }}°F</span>
          </div>
          <div class="stat-row u--space-bottom-small">
            <span class="stat-label text--small">Humidity:</span>
            <span class="stat-value text--medium">{{ data.main.humidity }}%</span>
          </div>
          <div class="stat-row">
            <span class="stat-label text--small">Wind:</span>
            <span class="stat-value text--medium">{{ data.wind.speed | round }} mph</span>
          </div>
        </div>
      </main>

      <footer class="update-time">
        <p class="text--tiny">{{ timestamp | date: "%I:%M %p" }}</p>
      </footer>

    {% else %}
      <div class="error-compact u--align-center">
        <div class="text--large">⚠️</div>
        <p class="text--medium">Weather unavailable</p>
        {% if data.message %}
          <p class="text--small">{{ data.message }}</p>
        {% endif %}
      </div>
    {% endif %}
  </div>
</div>
```

### Half Horizontal Layout (800×240)
```html
<div id="plugin-{{ instance_id }}" class="plugin-container view--half_horizontal">
  <div class="weather-horizontal flex-horizontal u--space-sides u--valign-middle">
    {% if data.cod == 200 %}
      <!-- Location and Temperature -->
      <div class="weather-primary flex-item">
        <div class="location-temp">
          <h3 class="text--medium">{{ data.name }}, {{ data.sys.country }}</h3>
          <div class="current-temp text--large">{{ data.main.temp | round }}°F</div>
        </div>
      </div>

      <!-- Weather Icon and Condition -->
      <div class="weather-condition flex-item u--align-center">
        <div class="condition-display">
          <div class="weather-icon text--large">
            {% case data.weather[0].main %}
              {% when 'Clear' %}☀️
              {% when 'Clouds' %}☁️
              {% when 'Rain' %}🌧️
              {% when 'Snow' %}❄️
              {% else %}🌤️
            {% endcase %}
          </div>
          <div class="condition-text text--medium">{{ data.weather[0].main }}</div>
        </div>
      </div>

      <!-- Quick Stats -->
      <div class="weather-stats flex-item">
        <div class="stats-compact">
          <div class="stat-item">
            <span class="text--small">Humidity: {{ data.main.humidity }}%</span>
          </div>
          <div class="stat-item">
            <span class="text--small">Wind: {{ data.wind.speed | round }} mph</span>
          </div>
          <div class="stat-item">
            <span class="text--tiny">Updated: {{ timestamp | date: "%I:%M %p" }}</span>
          </div>
        </div>
      </div>

    {% else %}
      <div class="error-horizontal flex-horizontal u--valign-middle">
        <div class="error-icon flex-item text--large">⚠️</div>
        <div class="error-text flex-item">
          <span class="text--medium">Weather data unavailable</span>
        </div>
      </div>
    {% endif %}
  </div>
</div>
```

### Quadrant Layout (400×240)
```html
<div id="plugin-{{ instance_id }}" class="plugin-container view--quadrant">
  <div class="weather-mini u--space-all u--align-center">
    {% if data.cod == 200 %}
      <div class="mini-weather">
        <div class="location text--medium u--space-bottom-small">
          {{ data.name | truncate: 12 }}
        </div>
        
        <div class="temp-condition u--space-bottom-small">
          <div class="temperature text--large">{{ data.main.temp | round }}°</div>
          <div class="condition-icon text--medium">
            {% case data.weather[0].main %}
              {% when 'Clear' %}☀️
              {% when 'Clouds' %}☁️
              {% when 'Rain' %}🌧️
              {% when 'Snow' %}❄️
              {% else %}🌤️
            {% endcase %}
          </div>
        </div>

        <div class="mini-stats">
          <div class="text--small">{{ data.main.humidity }}% • {{ data.wind.speed | round }}mph</div>
        </div>
      </div>

    {% else %}
      <div class="mini-error u--align-center">
        <div class="text--medium">⚠️</div>
        <div class="text--small">Weather unavailable</div>
      </div>
    {% endif %}
  </div>
</div>
```

## Custom Liquid Filters

To support the weather widget, you might want to add custom filters:

```liquid
<!-- Wind direction from degrees -->
{{ 180 | wind_direction }}  <!-- S -->

<!-- Pressure conversion -->
{{ 1013.25 | pressure_to_inches }}  <!-- 29.92 -->

<!-- Distance conversion -->
{{ 10000 | meters_to_miles }}  <!-- 6.2 -->
```

These filters would need to be implemented in the liquid renderer.

## API Integration Notes

### OpenWeatherMap API
- **Base URL**: `https://api.openweathermap.org/data/2.5/weather`
- **Rate Limit**: 1000 calls/day (free tier)
- **Response Format**: JSON with weather data structure
- **Error Codes**: Standard HTTP codes plus API-specific error responses

### Query Parameters
- `q`: City name and country code
- `appid`: Your API key
- `units`: Temperature units (imperial, metric, kelvin)
- `lang`: Language for weather descriptions

### Sample API Response
```json
{
  "cod": 200,
  "name": "New York",
  "sys": {
    "country": "US",
    "sunrise": 1710515400,
    "sunset": 1710558600
  },
  "main": {
    "temp": 72.5,
    "feels_like": 75.2,
    "humidity": 45,
    "pressure": 1013.25
  },
  "weather": [
    {
      "main": "Clear",
      "description": "clear sky",
      "icon": "01d"
    }
  ],
  "wind": {
    "speed": 8.5,
    "deg": 180
  },
  "visibility": 10000
}
```

## Styling Notes

### CSS Classes Used
- Layout: `grid-2-columns`, `flex-horizontal`, `flex-item`
- Typography: `text--huge`, `text--large`, `text--medium`, `text--small`, `text--tiny`
- Spacing: `u--space-all`, `u--space-bottom`, `u--space-top`
- Alignment: `u--align-center`, `u--valign-middle`
- Visual: `border-bottom`

### E-ink Optimizations
- **High contrast**: Black text on white background
- **No animations**: Static content only
- **Large fonts**: Readable on e-ink displays  
- **Clear icons**: Emoji weather icons work well on e-ink
- **Minimal colors**: Focus on typography and layout

## Deployment Steps

1. **Get API Key**: Sign up for OpenWeatherMap API
2. **Create Plugin**: Use the template code above
3. **Configure Polling**: Set your API key and location
4. **Test Layouts**: Preview in all layout sizes
5. **Validate**: Check for security and best practice compliance
6. **Create Instance**: Add to device playlists
7. **Monitor**: Check API usage and error rates

## Customization Ideas

### Location-based Variants
- Use device location for weather data
- Multiple cities in different layouts
- Weather alerts and warnings

### Enhanced Data Display
- 5-day forecast in full layout
- Weather history graphs
- Sunrise/sunset times with visual indicators

### Interactive Features
- Toggle between Fahrenheit/Celsius
- Multiple weather providers
- Weather-based device brightness adjustment

### Integration Options
- Combine with calendar for weather-aware scheduling
- Smart home integration for temperature control
- Weather-based notifications

This weather widget example demonstrates the full power of the private plugin system, showcasing proper error handling, responsive design, and effective use of the TRMNL framework.