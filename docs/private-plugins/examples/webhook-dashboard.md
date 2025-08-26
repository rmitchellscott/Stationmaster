# Webhook Dashboard Example

A comprehensive example of a private plugin that displays system metrics and alerts using the webhook data strategy.

## Overview

This plugin demonstrates:
- **Webhook Strategy**: Receives real-time data from monitoring systems
- **Complex Data Structures**: Handles nested objects and arrays
- **Status Indicators**: Visual health monitoring with icons and colors
- **Responsive Design**: Adapts to all TRMNL layout sizes

## Plugin Configuration

### Basic Information
```json
{
  "name": "System Monitor Dashboard",
  "description": "Real-time system health and performance metrics",
  "version": "1.2.0",
  "data_strategy": "webhook"
}
```

### Webhook URL
After creating the plugin, you'll get a unique webhook URL:
```
https://your-stationmaster.com/api/webhooks/plugin/abc123def456ghi789
```

## Sample Webhook Data

Send POST requests with JSON data to display system information:

### Server Monitoring Data
```json
{
  "timestamp": "2024-03-15T10:30:00Z",
  "environment": "production",
  "overview": {
    "status": "healthy",
    "total_servers": 5,
    "online_servers": 5,
    "alerts": 1,
    "uptime_percentage": 99.97
  },
  "servers": [
    {
      "name": "web-01",
      "status": "online",
      "cpu": 45.2,
      "memory": 67.8,
      "disk": 34.1,
      "uptime": "15d 4h 23m",
      "load_avg": 1.24
    },
    {
      "name": "web-02", 
      "status": "online",
      "cpu": 52.1,
      "memory": 71.3,
      "disk": 29.7,
      "uptime": "15d 4h 23m",
      "load_avg": 1.58
    },
    {
      "name": "db-01",
      "status": "warning",
      "cpu": 78.9,
      "memory": 89.2,
      "disk": 82.4,
      "uptime": "8d 12h 15m",
      "load_avg": 2.87,
      "alerts": ["High memory usage", "Disk space low"]
    }
  ],
  "services": {
    "web_server": {"status": "online", "response_time": 245},
    "database": {"status": "warning", "connections": 89, "max_connections": 100},
    "cache": {"status": "online", "hit_rate": 94.2},
    "queue": {"status": "online", "pending_jobs": 3}
  },
  "alerts": [
    {
      "level": "warning",
      "message": "Database connection pool at 89% capacity",
      "server": "db-01",
      "time": "2024-03-15T10:25:00Z"
    }
  ]
}
```

## Template Implementation

### Full Screen Layout (800×480)
```html
<div id="plugin-{{ instance_id }}" class="plugin-container view--full">
  <div class="dashboard-full u--space-all">
    {% if data.overview %}
      <!-- Dashboard Header -->
      <header class="dashboard-header u--space-bottom border-bottom">
        <div class="flex-horizontal">
          <div class="flex-item">
            <h1 class="text--large">System Monitor</h1>
            <p class="text--small">{{ data.environment | upcase }} Environment</p>
          </div>
          <div class="flex-item u--align-right">
            <div class="status-indicator">
              {% case data.overview.status %}
                {% when 'healthy' %}
                  <span class="status-icon text--large">🟢</span>
                  <span class="text--medium">All Systems Operational</span>
                {% when 'warning' %}
                  <span class="status-icon text--large">🟡</span>
                  <span class="text--medium">Minor Issues Detected</span>
                {% when 'critical' %}
                  <span class="status-icon text--large">🔴</span>
                  <span class="text--medium">Critical Issues</span>
                {% else %}
                  <span class="status-icon text--large">⚪</span>
                  <span class="text--medium">Status Unknown</span>
              {% endcase %}
            </div>
            <p class="text--small">Updated: {{ timestamp | date: "%I:%M %p" }}</p>
          </div>
        </div>
      </header>

      <!-- Key Metrics -->
      <section class="key-metrics u--space-bottom">
        <div class="metrics-grid grid-4-columns">
          <div class="metric-card column u--align-center u--pad-all border">
            <div class="metric-value text--huge">{{ data.overview.total_servers }}</div>
            <div class="metric-label text--small">Total Servers</div>
            <div class="metric-detail text--tiny">{{ data.overview.online_servers }} online</div>
          </div>

          <div class="metric-card column u--align-center u--pad-all border">
            <div class="metric-value text--huge">{{ data.overview.uptime_percentage }}%</div>
            <div class="metric-label text--small">Uptime</div>
            <div class="metric-detail text--tiny">Last 30 days</div>
          </div>

          <div class="metric-card column u--align-center u--pad-all border">
            <div class="metric-value text--huge">
              {% if data.overview.alerts > 0 %}
                <span class="text--warning">{{ data.overview.alerts }}</span>
              {% else %}
                0
              {% endif %}
            </div>
            <div class="metric-label text--small">Active Alerts</div>
            <div class="metric-detail text--tiny">
              {% if data.overview.alerts > 0 %}Needs attention{% else %}All clear{% endif %}
            </div>
          </div>

          <div class="metric-card column u--align-center u--pad-all border">
            <div class="metric-value text--huge">
              {{ data.services | size }}
            </div>
            <div class="metric-label text--small">Services</div>
            <div class="metric-detail text--tiny">Monitored</div>
          </div>
        </div>
      </section>

      <!-- Server Status Grid -->
      <section class="server-status u--space-bottom">
        <h2 class="text--medium u--space-bottom">Server Status</h2>
        
        {% if data.servers %}
          <div class="servers-grid">
            {% for server in data.servers limit: 6 %}
              <div class="server-card u--pad-all border u--space-bottom">
                <div class="server-header flex-horizontal u--space-bottom-small">
                  <div class="server-name flex-item">
                    <h3 class="text--medium">{{ server.name }}</h3>
                  </div>
                  <div class="server-status flex-item u--align-right">
                    {% case server.status %}
                      {% when 'online' %}
                        <span class="status-badge">🟢 Online</span>
                      {% when 'warning' %}
                        <span class="status-badge">🟡 Warning</span>
                      {% when 'critical' %}
                        <span class="status-badge">🔴 Critical</span>
                      {% when 'offline' %}
                        <span class="status-badge">⚫ Offline</span>
                    {% endcase %}
                  </div>
                </div>

                <div class="server-metrics grid-3-columns">
                  <div class="metric column">
                    <div class="metric-name text--small">CPU</div>
                    <div class="metric-value text--medium">{{ server.cpu | round: 1 }}%</div>
                    <div class="metric-bar">
                      <div class="bar-fill" style="width: {{ server.cpu }}%"></div>
                    </div>
                  </div>

                  <div class="metric column">
                    <div class="metric-name text--small">Memory</div>
                    <div class="metric-value text--medium">{{ server.memory | round: 1 }}%</div>
                    <div class="metric-bar">
                      <div class="bar-fill" style="width: {{ server.memory }}%"></div>
                    </div>
                  </div>

                  <div class="metric column">
                    <div class="metric-name text--small">Disk</div>
                    <div class="metric-value text--medium">{{ server.disk | round: 1 }}%</div>
                    <div class="metric-bar">
                      <div class="bar-fill" style="width: {{ server.disk }}%"></div>
                    </div>
                  </div>
                </div>

                {% if server.alerts %}
                  <div class="server-alerts u--space-top-small">
                    {% for alert in server.alerts limit: 2 %}
                      <div class="alert-item text--tiny">⚠️ {{ alert }}</div>
                    {% endfor %}
                  </div>
                {% endif %}
              </div>
            {% endfor %}
          </div>
        {% endif %}
      </section>

      <!-- Active Alerts -->
      {% if data.alerts and data.alerts.size > 0 %}
        <section class="active-alerts">
          <h2 class="text--medium u--space-bottom">Recent Alerts</h2>
          <div class="alerts-list">
            {% for alert in data.alerts limit: 3 %}
              <div class="alert-item u--pad-all border-bottom">
                <div class="alert-header flex-horizontal">
                  <div class="alert-level flex-item">
                    {% case alert.level %}
                      {% when 'critical' %}🔴
                      {% when 'warning' %}🟡
                      {% when 'info' %}🔵
                      {% else %}⚪
                    {% endcase %}
                    <span class="text--medium">{{ alert.level | upcase }}</span>
                  </div>
                  <div class="alert-time flex-item u--align-right">
                    <span class="text--small">{{ alert.time | date: "%I:%M %p" }}</span>
                  </div>
                </div>
                <div class="alert-message text--medium u--space-top-small">
                  {{ alert.message }}
                </div>
                {% if alert.server %}
                  <div class="alert-source text--small u--space-top-small">
                    Source: {{ alert.server }}
                  </div>
                {% endif %}
              </div>
            {% endfor %}
          </div>
        </section>
      {% endif %}

    {% else %}
      <!-- No Data State -->
      <div class="no-data u--align-center u--space-all">
        <div class="no-data-icon text--huge">📊</div>
        <h2 class="text--large u--space-bottom">Waiting for Data</h2>
        <p class="text--medium">Send system metrics to your webhook URL to begin monitoring.</p>
        <div class="webhook-info u--space-top">
          <p class="text--small">Webhook endpoint ready</p>
        </div>
      </div>
    {% endif %}
  </div>
</div>
```

### Half Vertical Layout (400×480)
```html
<div id="plugin-{{ instance_id }}" class="plugin-container view--half_vertical">
  <div class="dashboard-compact u--space-all">
    {% if data.overview %}
      <!-- Compact Header -->
      <header class="compact-header u--space-bottom u--align-center">
        <h2 class="text--large">System Health</h2>
        <div class="overall-status u--space-top-small">
          {% case data.overview.status %}
            {% when 'healthy' %}🟢 Healthy
            {% when 'warning' %}🟡 Warning
            {% when 'critical' %}🔴 Critical
            {% else %}⚪ Unknown
          {% endcase %}
        </div>
      </header>

      <!-- Key Stats -->
      <section class="compact-stats u--space-bottom">
        <div class="stat-row u--space-bottom-small">
          <span class="stat-label text--small">Servers:</span>
          <span class="stat-value text--medium">{{ data.overview.online_servers }}/{{ data.overview.total_servers }}</span>
        </div>
        <div class="stat-row u--space-bottom-small">
          <span class="stat-label text--small">Uptime:</span>
          <span class="stat-value text--medium">{{ data.overview.uptime_percentage }}%</span>
        </div>
        <div class="stat-row u--space-bottom-small">
          <span class="stat-label text--small">Alerts:</span>
          <span class="stat-value text--medium">
            {% if data.overview.alerts > 0 %}⚠️ {{ data.overview.alerts }}{% else %}✅ 0{% endif %}
          </span>
        </div>
      </section>

      <!-- Service Status -->
      <section class="service-status u--space-bottom">
        <h3 class="text--medium u--space-bottom">Services</h3>
        {% for service in data.services %}
          <div class="service-item u--space-bottom-small">
            <div class="service-info flex-horizontal">
              <div class="service-name flex-item text--small">{{ service[0] | replace: '_', ' ' | capitalize }}</div>
              <div class="service-status flex-item u--align-right">
                {% case service[1].status %}
                  {% when 'online' %}🟢
                  {% when 'warning' %}🟡
                  {% when 'critical' %}🔴
                  {% when 'offline' %}⚫
                {% endcase %}
              </div>
            </div>
          </div>
        {% endfor %}
      </section>

      <!-- Recent Alerts -->
      {% if data.alerts and data.alerts.size > 0 %}
        <section class="recent-alerts">
          <h3 class="text--medium u--space-bottom">Latest Alert</h3>
          {% assign latest_alert = data.alerts | first %}
          <div class="alert-compact u--pad-all border">
            <div class="alert-level text--small">
              {% case latest_alert.level %}
                {% when 'critical' %}🔴 CRITICAL
                {% when 'warning' %}🟡 WARNING
                {% else %}🔵 INFO
              {% endcase %}
            </div>
            <div class="alert-message text--small u--space-top-small">
              {{ latest_alert.message | truncate: 80 }}
            </div>
            <div class="alert-time text--tiny u--space-top-small">
              {{ latest_alert.time | date: "%I:%M %p" }}
            </div>
          </div>
        </section>
      {% endif %}

      <footer class="compact-footer u--space-top">
        <p class="text--tiny u--align-center">
          Updated: {{ timestamp | date: "%I:%M %p" }}
        </p>
      </footer>

    {% else %}
      <!-- Compact No Data State -->
      <div class="compact-no-data u--align-center u--space-all">
        <div class="text--large">📊</div>
        <h3 class="text--medium u--space-bottom">System Monitor</h3>
        <p class="text--small">Waiting for webhook data...</p>
      </div>
    {% endif %}
  </div>
</div>
```

### Half Horizontal Layout (800×240)
```html
<div id="plugin-{{ instance_id }}" class="plugin-container view--half_horizontal">
  <div class="dashboard-horizontal flex-horizontal u--space-sides u--valign-middle">
    {% if data.overview %}
      <!-- Status Overview -->
      <div class="status-overview flex-item">
        <div class="status-summary">
          <h3 class="text--medium">{{ data.environment | upcase }}</h3>
          <div class="status-indicator">
            {% case data.overview.status %}
              {% when 'healthy' %}🟢 All Systems Go
              {% when 'warning' %}🟡 {{ data.overview.alerts }} Alert(s)
              {% when 'critical' %}🔴 Critical Issues
              {% else %}⚪ Status Unknown
            {% endcase %}
          </div>
        </div>
      </div>

      <!-- Quick Metrics -->
      <div class="quick-metrics flex-item flex-grow">
        <div class="metrics-horizontal flex-horizontal">
          <div class="metric-item flex-item u--align-center">
            <div class="metric-value text--medium">{{ data.overview.online_servers }}</div>
            <div class="metric-label text--small">Servers Online</div>
          </div>
          
          <div class="metric-item flex-item u--align-center">
            <div class="metric-value text--medium">{{ data.overview.uptime_percentage }}%</div>
            <div class="metric-label text--small">Uptime</div>
          </div>

          <div class="metric-item flex-item u--align-center">
            <div class="metric-value text--medium">
              {% assign online_services = 0 %}
              {% for service in data.services %}
                {% if service[1].status == 'online' %}
                  {% assign online_services = online_services | plus: 1 %}
                {% endif %}
              {% endfor %}
              {{ online_services }}/{{ data.services | size }}
            </div>
            <div class="metric-label text--small">Services Up</div>
          </div>

          {% if data.alerts and data.alerts.size > 0 %}
            <div class="metric-item flex-item u--align-center">
              <div class="metric-value text--medium">⚠️{{ data.alerts | size }}</div>
              <div class="metric-label text--small">Alerts</div>
            </div>
          {% endif %}
        </div>
      </div>

      <!-- Timestamp -->
      <div class="timestamp-section flex-item u--align-right">
        <div class="update-time">
          <div class="time-value text--medium">{{ timestamp | date: "%I:%M" }}</div>
          <div class="time-label text--small">{{ timestamp | date: "%p" }}</div>
        </div>
      </div>

    {% else %}
      <!-- Horizontal No Data -->
      <div class="horizontal-no-data flex-horizontal u--valign-middle">
        <div class="no-data-icon flex-item text--large">📊</div>
        <div class="no-data-text flex-item">
          <span class="text--medium">System Monitor - Awaiting Data</span>
        </div>
      </div>
    {% endif %}
  </div>
</div>
```

### Quadrant Layout (400×240)
```html
<div id="plugin-{{ instance_id }}" class="plugin-container view--quadrant">
  <div class="dashboard-mini u--space-all u--align-center">
    {% if data.overview %}
      <div class="mini-status u--space-bottom">
        <div class="status-icon text--large">
          {% case data.overview.status %}
            {% when 'healthy' %}🟢
            {% when 'warning' %}🟡
            {% when 'critical' %}🔴
            {% else %}⚪
          {% endcase %}
        </div>
        <div class="status-text text--small">{{ data.overview.status | upcase }}</div>
      </div>

      <div class="mini-metrics">
        <div class="mini-stat u--space-bottom-small">
          <span class="text--medium">{{ data.overview.online_servers }}/{{ data.overview.total_servers }}</span>
          <span class="text--small"> servers</span>
        </div>
        
        {% if data.overview.alerts > 0 %}
          <div class="mini-alerts">
            <span class="text--small">⚠️ {{ data.overview.alerts }} alert(s)</span>
          </div>
        {% endif %}
      </div>

      <div class="mini-time u--space-top">
        <span class="text--tiny">{{ timestamp | date: "%I:%M %p" }}</span>
      </div>

    {% else %}
      <!-- Mini No Data -->
      <div class="mini-no-data u--align-center">
        <div class="text--medium">📊</div>
        <div class="text--small">Monitor</div>
        <div class="text--tiny">No data</div>
      </div>
    {% endif %}
  </div>
</div>
```

## Integration Examples

### Python Monitoring Script
```python
import requests
import psutil
import time
from datetime import datetime

WEBHOOK_URL = "https://your-stationmaster.com/api/webhooks/plugin/your-token"

def get_system_metrics():
    """Collect system metrics"""
    cpu_percent = psutil.cpu_percent(interval=1)
    memory = psutil.virtual_memory()
    disk = psutil.disk_usage('/')
    
    return {
        "cpu": cpu_percent,
        "memory": memory.percent,
        "disk": disk.percent,
        "uptime": time.time() - psutil.boot_time()
    }

def send_dashboard_update():
    """Send metrics to webhook"""
    metrics = get_system_metrics()
    
    # Determine overall status
    status = "healthy"
    if metrics["cpu"] > 80 or metrics["memory"] > 85 or metrics["disk"] > 90:
        status = "critical"
    elif metrics["cpu"] > 60 or metrics["memory"] > 75 or metrics["disk"] > 80:
        status = "warning"
    
    # Build webhook payload
    payload = {
        "timestamp": datetime.utcnow().isoformat() + "Z",
        "environment": "production",
        "overview": {
            "status": status,
            "total_servers": 3,
            "online_servers": 3,
            "alerts": 1 if status != "healthy" else 0,
            "uptime_percentage": 99.97
        },
        "servers": [
            {
                "name": "app-01",
                "status": status,
                "cpu": metrics["cpu"],
                "memory": metrics["memory"],
                "disk": metrics["disk"],
                "uptime": f"{int(metrics['uptime'] // 86400)}d {int((metrics['uptime'] % 86400) // 3600)}h"
            }
        ],
        "services": {
            "web_server": {"status": "online", "response_time": 245},
            "database": {"status": "online", "connections": 45},
            "cache": {"status": "online", "hit_rate": 94.2}
        },
        "alerts": [
            {
                "level": "warning",
                "message": f"High CPU usage: {metrics['cpu']:.1f}%",
                "server": "app-01",
                "time": datetime.utcnow().isoformat() + "Z"
            }
        ] if status != "healthy" else []
    }
    
    try:
        response = requests.post(WEBHOOK_URL, json=payload, timeout=10)
        response.raise_for_status()
        print(f"✅ Dashboard updated successfully - Status: {status}")
    except requests.RequestException as e:
        print(f"❌ Failed to update dashboard: {e}")

if __name__ == "__main__":
    # Send update every 5 minutes
    while True:
        send_dashboard_update()
        time.sleep(300)
```

### Docker Compose Health Check
```bash
#!/bin/bash
# docker-health-webhook.sh

WEBHOOK_URL="https://your-stationmaster.com/api/webhooks/plugin/your-token"
COMPOSE_FILE="docker-compose.yml"

# Get service status
services=$(docker-compose -f $COMPOSE_FILE ps --services)
online_count=0
total_count=0
service_status="{}"

for service in $services; do
    status=$(docker-compose -f $COMPOSE_FILE ps -q $service | xargs docker inspect -f '{{.State.Status}}' 2>/dev/null || echo "stopped")
    total_count=$((total_count + 1))
    
    if [ "$status" = "running" ]; then
        online_count=$((online_count + 1))
        service_status=$(echo $service_status | jq ". + {\"$service\": {\"status\": \"online\"}}")
    else
        service_status=$(echo $service_status | jq ". + {\"$service\": {\"status\": \"offline\"}}")
    fi
done

# Determine overall health
if [ $online_count -eq $total_count ]; then
    overall_status="healthy"
elif [ $online_count -gt 0 ]; then
    overall_status="warning"
else
    overall_status="critical"
fi

# Send webhook
curl -X POST "$WEBHOOK_URL" \
  -H "Content-Type: application/json" \
  -d "{
    \"timestamp\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\",
    \"environment\": \"docker\",
    \"overview\": {
      \"status\": \"$overall_status\",
      \"total_servers\": $total_count,
      \"online_servers\": $online_count,
      \"uptime_percentage\": 99.5
    },
    \"services\": $service_status
  }"
```

## Customization Ideas

### Multi-Environment Support
- Separate dashboards for dev/staging/production
- Environment-specific color schemes
- Cross-environment status comparison

### Advanced Metrics
- Network traffic graphs
- Application-specific metrics
- Performance trends and predictions

### Alert Management
- Alert escalation rules
- Notification integration
- Alert acknowledgment tracking

### Interactive Features
- Drill-down into specific servers
- Historical data views
- Maintenance mode indicators

This webhook dashboard example shows how to create a comprehensive monitoring solution using real-time data delivery and responsive design principles.