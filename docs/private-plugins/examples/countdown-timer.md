# Countdown Timer Example

A simple yet effective example of a private plugin that displays countdown timers for important events using the merge data strategy and JavaScript date calculations.

## Overview

This plugin demonstrates:
- **Merge Strategy**: Uses only system and user data, no external APIs
- **Date Calculations**: Client-side date math with Liquid filters
- **Event Management**: Multiple countdown events with different priorities
- **Visual Design**: Clean, readable countdown display for e-ink screens

## Plugin Configuration

### Basic Information
```json
{
  "name": "Event Countdown Timer",
  "description": "Display countdown to important dates and events",
  "version": "1.0.0", 
  "data_strategy": "merge"
}
```

### Configuration Data
Since this uses the merge strategy, event data would be configured in the plugin settings:

```json
{
  "events": [
    {
      "name": "Project Launch",
      "date": "2024-04-15T09:00:00Z",
      "priority": "high",
      "icon": "🚀",
      "color": "primary"
    },
    {
      "name": "Team Meeting",
      "date": "2024-03-18T14:00:00Z", 
      "priority": "medium",
      "icon": "📅",
      "color": "secondary"
    },
    {
      "name": "Vacation Starts",
      "date": "2024-05-20T17:00:00Z",
      "priority": "high",
      "icon": "🏖️",
      "color": "success"
    }
  ],
  "timezone": "America/New_York",
  "show_past_events": false,
  "max_events": 3
}
```

## Template Implementation

### Shared Markup (Used by All Layouts)
```html
<div id="plugin-{{ instance_id }}" class="plugin-container">
  <div class="countdown-timer u--space-all">
    
    <!-- Helper: Calculate time differences -->
    {% assign now_timestamp = timestamp | date: '%s' %}
    {% assign upcoming_events = '' | split: ',' %}
    
    <!-- Filter and sort events -->
    {% for event in form_fields.events %}
      {% assign event_timestamp = event.date | date: '%s' %}
      {% assign time_diff = event_timestamp | minus: now_timestamp %}
      
      {% if time_diff > 0 or form_fields.show_past_events %}
        {% assign upcoming_events = upcoming_events | push: event %}
      {% endif %}
    {% endfor %}
    
    {% if upcoming_events.size > 0 %}
      <!-- Events Found -->
      <div class="events-container">
        {% for event in upcoming_events limit: form_fields.max_events %}
          {% assign event_timestamp = event.date | date: '%s' %}
          {% assign time_diff = event_timestamp | minus: now_timestamp %}
          {% assign is_past = time_diff <= 0 %}
          
          <div class="event-item {% if forloop.last %}u--space-bottom{% else %}u--space-bottom border-bottom{% endif %}">
            
            <!-- Event Header -->
            <div class="event-header flex-horizontal u--space-bottom-small">
              <div class="event-title flex-item">
                <div class="event-icon-name">
                  {% if event.icon %}{{ event.icon }}{% endif %}
                  <span class="event-name text--medium">{{ event.name }}</span>
                </div>
              </div>
              <div class="event-priority flex-item u--align-right">
                {% case event.priority %}
                  {% when 'high' %}
                    <span class="priority-badge priority--high text--small">HIGH</span>
                  {% when 'medium' %}
                    <span class="priority-badge priority--medium text--small">MED</span>
                  {% when 'low' %}
                    <span class="priority-badge priority--low text--small">LOW</span>
                {% endcase %}
              </div>
            </div>
            
            <!-- Countdown Display -->
            <div class="countdown-display u--align-center">
              {% if is_past %}
                <!-- Past Event -->
                <div class="past-event">
                  <div class="past-indicator text--large">✅</div>
                  <div class="past-text text--medium">Completed</div>
                  <div class="past-date text--small">{{ event.date | date: "%B %d, %Y at %I:%M %p" }}</div>
                </div>
              {% else %}
                <!-- Future Event - Calculate Time Components -->
                {% assign days = time_diff | divided_by: 86400 %}
                {% assign hours = time_diff | modulo: 86400 | divided_by: 3600 %}
                {% assign minutes = time_diff | modulo: 3600 | divided_by: 60 %}
                {% assign seconds = time_diff | modulo: 60 %}
                
                <div class="countdown-components">
                  {% if days > 0 %}
                    <!-- Days and Hours -->
                    <div class="countdown-row">
                      <div class="countdown-unit">
                        <div class="unit-value text--huge">{{ days }}</div>
                        <div class="unit-label text--small">day{% if days != 1 %}s{% endif %}</div>
                      </div>
                      {% if days < 7 %}
                        <div class="countdown-separator text--large">:</div>
                        <div class="countdown-unit">
                          <div class="unit-value text--huge">{{ hours }}</div>
                          <div class="unit-label text--small">hour{% if hours != 1 %}s{% endif %}</div>
                        </div>
                      {% endif %}
                    </div>
                  {% elsif hours > 0 %}
                    <!-- Hours and Minutes -->
                    <div class="countdown-row">
                      <div class="countdown-unit">
                        <div class="unit-value text--huge">{{ hours }}</div>
                        <div class="unit-label text--small">hour{% if hours != 1 %}s{% endif %}</div>
                      </div>
                      <div class="countdown-separator text--large">:</div>
                      <div class="countdown-unit">
                        <div class="unit-value text--huge">{{ minutes }}</div>
                        <div class="unit-label text--small">min{% if minutes != 1 %}s{% endif %}</div>
                      </div>
                    </div>
                  {% else %}
                    <!-- Minutes Only (or "Soon" for very close events) -->
                    <div class="countdown-row">
                      {% if minutes > 0 %}
                        <div class="countdown-unit">
                          <div class="unit-value text--huge">{{ minutes }}</div>
                          <div class="unit-label text--small">minute{% if minutes != 1 %}s{% endif %}</div>
                        </div>
                      {% else %}
                        <div class="countdown-urgent">
                          <div class="urgent-text text--huge">SOON!</div>
                          <div class="urgent-label text--small">Any moment now</div>
                        </div>
                      {% endif %}
                    </div>
                  {% endif %}
                </div>
                
                <!-- Event Date -->
                <div class="event-date u--space-top">
                  <div class="date-display text--small">
                    {{ event.date | date: "%A, %B %d" }}
                  </div>
                  <div class="time-display text--small">
                    {{ event.date | date: "%I:%M %p" }}
                    {% if form_fields.timezone %}
                      <span class="timezone">({{ form_fields.timezone | split: '/' | last | replace: '_', ' ' }})</span>
                    {% endif %}
                  </div>
                </div>
              {% endif %}
            </div>
          </div>
        {% endfor %}
      </div>
      
      <!-- Footer -->
      <div class="timer-footer u--space-top u--align-center">
        <div class="last-updated text--tiny">
          Updated: {{ timestamp | date: "%I:%M:%S %p" }}
        </div>
      </div>
      
    {% else %}
      <!-- No Events -->
      <div class="no-events u--align-center u--space-all">
        <div class="no-events-icon text--huge">📅</div>
        <h3 class="text--large u--space-bottom">No Upcoming Events</h3>
        <p class="text--medium">Add events to your configuration to see countdowns.</p>
      </div>
    {% endif %}
  </div>
</div>
```

### Layout-Specific Styling

Since this plugin uses shared markup, we add CSS classes that adapt to different layouts:

```css
/* Responsive adjustments for different layouts */

/* Full Screen - Show all details */
.view--full .countdown-components {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 1rem;
}

.view--full .countdown-unit {
  text-align: center;
  min-width: 100px;
}

/* Half Vertical - Stack vertically */
.view--half_vertical .countdown-row {
  flex-direction: column;
  gap: 0.5rem;
}

.view--half_vertical .countdown-separator {
  display: none;
}

.view--half_vertical .event-name {
  font-size: 0.9em;
  max-width: 200px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* Half Horizontal - Compact horizontal */
.view--half_horizontal .countdown-components {
  flex-direction: row;
  justify-content: space-around;
}

.view--half_horizontal .unit-value {
  font-size: 1.5em;
}

.view--half_horizontal .event-date {
  display: none; /* Hide date details to save space */
}

/* Quadrant - Minimal display */
.view--quadrant .event-item {
  margin-bottom: 0.5rem;
}

.view--quadrant .event-header {
  flex-direction: column;
  text-align: center;
}

.view--quadrant .priority-badge {
  display: none;
}

.view--quadrant .countdown-unit .unit-label {
  font-size: 0.7em;
}

.view--quadrant .event-date {
  display: none;
}
```

## Advanced Features

### Multiple Event Types
```html
<!-- Event categories with different styling -->
{% case event.category %}
  {% when 'work' %}
    <div class="event-category work-event">
      <span class="category-icon">💼</span>
      <span class="category-label">Work</span>
    </div>
  {% when 'personal' %}
    <div class="event-category personal-event">
      <span class="category-icon">👤</span>
      <span class="category-label">Personal</span>
    </div>
  {% when 'holiday' %}
    <div class="event-category holiday-event">
      <span class="category-icon">🎉</span>
      <span class="category-label">Holiday</span>
    </div>
{% endcase %}
```

### Progress Indicators
```html
<!-- Progress bar for events with start/end dates -->
{% if event.start_date and event.end_date %}
  {% assign start_timestamp = event.start_date | date: '%s' %}
  {% assign end_timestamp = event.end_date | date: '%s' %}
  {% assign total_duration = end_timestamp | minus: start_timestamp %}
  {% assign elapsed_time = now_timestamp | minus: start_timestamp %}
  {% assign progress_percent = elapsed_time | times: 100 | divided_by: total_duration %}
  
  <div class="event-progress u--space-top">
    <div class="progress-bar">
      <div class="progress-fill" style="width: {{ progress_percent | at_most: 100 }}%"></div>
    </div>
    <div class="progress-text text--tiny">
      {{ progress_percent | at_most: 100 | at_least: 0 | round }}% complete
    </div>
  </div>
{% endif %}
```

### Urgency Indicators
```html
<!-- Visual urgency based on time remaining -->
{% assign urgency_class = 'normal' %}
{% if days == 0 and hours < 2 %}
  {% assign urgency_class = 'urgent' %}
{% elsif days == 0 and hours < 24 %}
  {% assign urgency_class = 'soon' %}
{% elsif days <= 3 %}
  {% assign urgency_class = 'approaching' %}
{% endif %}

<div class="countdown-display urgency--{{ urgency_class }}">
  <!-- Countdown content with urgency styling -->
</div>
```

## Configuration Examples

### Plugin Settings JSON
```json
{
  "events": [
    {
      "name": "Product Launch",
      "date": "2024-04-15T09:00:00Z",
      "priority": "high",
      "category": "work",
      "icon": "🚀",
      "color": "#FF6B6B",
      "description": "Official product launch event"
    },
    {
      "name": "Birthday Party",
      "date": "2024-03-25T19:00:00Z",
      "priority": "high", 
      "category": "personal",
      "icon": "🎂",
      "color": "#4ECDC4",
      "description": "Sarah's birthday celebration"
    },
    {
      "name": "Quarterly Review",
      "date": "2024-03-31T14:00:00Z",
      "priority": "medium",
      "category": "work", 
      "icon": "📊",
      "color": "#45B7D1",
      "description": "Q1 performance review meeting"
    }
  ],
  "timezone": "America/New_York",
  "show_past_events": true,
  "max_events": 5,
  "date_format": "%B %d, %Y",
  "time_format": "%I:%M %p",
  "refresh_interval": 60
}
```

### Dynamic Event Management
```javascript
// Example of updating events via API call
const updateCountdownEvents = async () => {
  const events = await fetchFromAPI('/api/calendar/upcoming');
  
  const formattedEvents = events.map(event => ({
    name: event.title,
    date: event.start_time,
    priority: event.importance || 'medium',
    category: event.category || 'general',
    icon: getIconForCategory(event.category),
    description: event.description
  }));

  // Update plugin configuration
  await updatePluginConfig(pluginId, {
    events: formattedEvents,
    last_sync: new Date().toISOString()
  });
};
```

## Use Cases

### Project Management
- Sprint deadlines
- Milestone deliveries  
- Release dates
- Review meetings

### Personal Events
- Birthdays and anniversaries
- Vacation starts
- Bill due dates
- Appointment reminders

### Business Operations
- Campaign launches
- Sales targets
- Conference dates
- Contract renewals

### Academic Calendar
- Assignment due dates
- Exam schedules
- Semester breaks
- Graduation events

## Styling Guidelines

### E-ink Optimized Design
```css
/* High contrast for e-ink displays */
.countdown-display {
  color: #000;
  background: #fff;
  font-weight: bold;
}

/* Large, readable fonts */
.unit-value {
  font-size: 2.5rem;
  font-weight: 900;
  line-height: 1;
}

/* Clear visual hierarchy */
.event-name {
  font-size: 1.2rem;
  font-weight: 600;
  margin-bottom: 0.5rem;
}

/* Subtle borders and dividers */
.event-item {
  border-bottom: 2px solid #ccc;
  padding-bottom: 1rem;
  margin-bottom: 1rem;
}

/* Priority indicators */
.priority--high { 
  background: #000; 
  color: #fff; 
  padding: 2px 8px;
  font-size: 0.8rem;
}

.priority--medium { 
  background: #666; 
  color: #fff; 
  padding: 2px 8px;
  font-size: 0.8rem;
}
```

### Responsive Typography
```css
/* Adjust font sizes for different layouts */
.view--full .unit-value { font-size: 3rem; }
.view--half_vertical .unit-value { font-size: 2rem; }
.view--half_horizontal .unit-value { font-size: 1.5rem; }
.view--quadrant .unit-value { font-size: 1.2rem; }

/* Maintain readability */
@media (max-width: 400px) {
  .unit-value { font-size: 1.5rem !important; }
  .event-name { font-size: 1rem !important; }
}
```

## Troubleshooting

### Common Issues

#### Dates Not Calculating Correctly
- **Issue**: Countdown shows wrong time
- **Solution**: Ensure dates are in ISO format with timezone
- **Example**: `2024-03-15T14:30:00Z` (UTC) or `2024-03-15T14:30:00-04:00` (EST)

#### Events Not Appearing
- **Issue**: No events show despite configuration
- **Solution**: Check JSON syntax and date formats
- **Debug**: Add `{{ form_fields.events | json }}` to see parsed data

#### Layout Issues
- **Issue**: Content cut off or poorly formatted
- **Solution**: Test in all layout sizes and adjust CSS
- **Tip**: Use `text--small` for constrained layouts

### Testing Scenarios

#### Test with Different Time Ranges
```json
{
  "test_events": [
    {
      "name": "In 30 seconds",
      "date": "2024-03-15T10:30:30Z",
      "priority": "high"
    },
    {
      "name": "In 5 minutes", 
      "date": "2024-03-15T10:35:00Z",
      "priority": "medium"
    },
    {
      "name": "In 2 hours",
      "date": "2024-03-15T12:30:00Z", 
      "priority": "low"
    },
    {
      "name": "In 5 days",
      "date": "2024-03-20T10:30:00Z",
      "priority": "high"
    }
  ]
}
```

This countdown timer example demonstrates how to create engaging, time-sensitive displays using only the merge data strategy and careful date manipulation with Liquid templates.