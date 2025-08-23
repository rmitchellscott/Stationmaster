# Multi-Layout Design Example

A comprehensive example showing how to create private plugins that work effectively across all TRMNL screen layouts with adaptive design patterns.

## Overview

This example demonstrates:
- **Responsive Design**: How content adapts to different screen dimensions
- **Layout-Specific Templates**: Custom templates for each layout type
- **Content Prioritization**: Showing more/less detail based on available space
- **Design Patterns**: Common UI patterns that work across all layouts

## Plugin Configuration

### Basic Information
```json
{
  "name": "Multi-Layout News Feed",
  "description": "Adaptive news display that works perfectly on any screen size",
  "version": "1.0.0",
  "data_strategy": "polling"
}
```

### Polling Configuration
```json
{
  "urls": [
    {
      "url": "https://api.example-news.com/headlines",
      "method": "GET",
      "headers": {
        "Authorization": "Bearer your-api-key"
      },
      "interval": 3600,
      "timeout": 30
    }
  ]
}
```

### Sample News Data
```json
{
  "articles": [
    {
      "id": 1,
      "title": "Major Technology Breakthrough Announced",
      "summary": "Scientists at leading university make significant discovery in quantum computing research.",
      "category": "technology",
      "published_at": "2024-03-15T09:30:00Z",
      "priority": "high",
      "read_time": 3
    },
    {
      "id": 2,
      "title": "Global Markets Show Strong Performance", 
      "summary": "Stock markets worldwide continue upward trend amid positive economic indicators.",
      "category": "business",
      "published_at": "2024-03-15T08:15:00Z",
      "priority": "medium",
      "read_time": 2
    },
    {
      "id": 3,
      "title": "Climate Summit Reaches Historic Agreement",
      "summary": "World leaders commit to ambitious new targets for carbon emission reductions.",
      "category": "environment",
      "published_at": "2024-03-15T07:45:00Z", 
      "priority": "high",
      "read_time": 4
    }
  ]
}
```

## Layout-Specific Templates

### Full Screen Layout (800×480)
Maximum space available - show rich, detailed content:

```html
<div id="plugin-{{ instance_id }}" class="plugin-container view--full">
  <div class="news-feed-full u--space-all">
    
    <!-- Rich Header with Branding -->
    <header class="news-header u--space-bottom border-bottom">
      <div class="flex-horizontal">
        <div class="flex-item">
          <h1 class="text--huge">📰 News Feed</h1>
          <p class="text--medium">Latest headlines and updates</p>
        </div>
        <div class="flex-item u--align-right">
          <div class="update-info">
            <div class="update-time text--medium">{{ timestamp | date: "%I:%M %p" }}</div>
            <div class="article-count text--small">{{ data.articles | size }} articles</div>
          </div>
        </div>
      </div>
    </header>

    {% if data.articles and data.articles.size > 0 %}
      <!-- Feature Story (First Article) -->
      {% assign featured = data.articles | first %}
      <section class="featured-story u--space-bottom">
        <article class="feature-article u--pad-all border">
          <header class="article-header u--space-bottom">
            <div class="article-meta flex-horizontal u--space-bottom-small">
              <div class="category-badge flex-item">
                {% case featured.category %}
                  {% when 'technology' %}🔬 Technology
                  {% when 'business' %}💼 Business
                  {% when 'environment' %}🌍 Environment
                  {% when 'politics' %}🏛️ Politics
                  {% when 'sports' %}⚽ Sports
                  {% else %}📰 News
                {% endcase %}
              </div>
              <div class="article-priority flex-item u--align-right">
                {% if featured.priority == 'high' %}
                  <span class="priority-high">🔥 Breaking</span>
                {% endif %}
              </div>
            </div>
            <h2 class="article-title text--large">{{ featured.title }}</h2>
          </header>
          
          <div class="article-content">
            <p class="article-summary text--medium u--space-bottom">{{ featured.summary }}</p>
            
            <div class="article-footer flex-horizontal">
              <div class="publish-info flex-item">
                <span class="publish-time text--small">{{ featured.published_at | date: "%B %d, %I:%M %p" }}</span>
              </div>
              <div class="read-time flex-item u--align-right">
                <span class="text--small">{{ featured.read_time }} min read</span>
              </div>
            </div>
          </div>
        </article>
      </section>

      <!-- Additional Headlines -->
      <section class="headlines-grid">
        <h3 class="section-title text--medium u--space-bottom">More Headlines</h3>
        
        <div class="headlines-list">
          {% for article in data.articles offset: 1 limit: 4 %}
            <article class="headline-item u--space-bottom border-bottom">
              <div class="headline-content flex-horizontal">
                <div class="headline-text flex-item">
                  <h4 class="headline-title text--medium">{{ article.title }}</h4>
                  <p class="headline-summary text--small">{{ article.summary | truncate: 100 }}</p>
                </div>
                <div class="headline-meta flex-item u--align-right">
                  <div class="category-icon text--medium">
                    {% case article.category %}
                      {% when 'technology' %}🔬
                      {% when 'business' %}💼
                      {% when 'environment' %}🌍
                      {% when 'politics' %}🏛️
                      {% when 'sports' %}⚽
                      {% else %}📰
                    {% endcase %}
                  </div>
                  <div class="publish-time text--tiny">{{ article.published_at | date: "%I:%M %p" }}</div>
                </div>
              </div>
            </article>
          {% endfor %}
        </div>
      </section>

    {% else %}
      <!-- No Articles State -->
      <div class="no-content u--align-center u--space-all">
        <div class="no-content-icon text--huge">📰</div>
        <h2 class="text--large u--space-bottom">No Articles Available</h2>
        <p class="text--medium">Check your news feed configuration or try again later.</p>
      </div>
    {% endif %}

    <!-- Footer with Source Attribution -->
    <footer class="news-footer u--space-top u--align-center">
      <p class="text--tiny">News provided by Example News API</p>
    </footer>
    
  </div>
</div>
```

### Half Vertical Layout (400×480)
Narrower but tall - stack content vertically:

```html
<div id="plugin-{{ instance_id }}" class="plugin-container view--half_vertical">
  <div class="news-feed-vertical u--space-all">
    
    <!-- Compact Header -->
    <header class="news-header-compact u--space-bottom u--align-center">
      <h2 class="text--large">📰 News</h2>
      <p class="text--small">{{ timestamp | date: "%I:%M %p" }}</p>
    </header>

    {% if data.articles and data.articles.size > 0 %}
      <!-- Top Story -->
      {% assign top_story = data.articles | first %}
      <section class="top-story u--space-bottom">
        <article class="story-compact u--pad-all border">
          <div class="story-category u--space-bottom-small">
            <span class="category-tag text--small">
              {% case top_story.category %}
                {% when 'technology' %}🔬 TECH
                {% when 'business' %}💼 BIZ
                {% when 'environment' %}🌍 ENV
                {% when 'politics' %}🏛️ POLITICS
                {% when 'sports' %}⚽ SPORTS
                {% else %}📰 NEWS
              {% endcase %}
            </span>
            {% if top_story.priority == 'high' %}
              <span class="breaking-badge text--small">🔥</span>
            {% endif %}
          </div>
          
          <h3 class="story-title text--medium u--space-bottom">
            {{ top_story.title | truncate: 50 }}
          </h3>
          
          <p class="story-summary text--small u--space-bottom">
            {{ top_story.summary | truncate: 80 }}
          </p>
          
          <div class="story-time text--tiny">
            {{ top_story.published_at | date: "%I:%M %p" }}
          </div>
        </article>
      </section>

      <!-- Headlines List -->
      <section class="headlines-vertical">
        <h4 class="section-title text--medium u--space-bottom">Headlines</h4>
        
        {% for article in data.articles offset: 1 limit: 3 %}
          <div class="headline-vertical u--space-bottom">
            <div class="headline-row">
              <div class="headline-icon">
                {% case article.category %}
                  {% when 'technology' %}🔬
                  {% when 'business' %}💼
                  {% when 'environment' %}🌍
                  {% when 'politics' %}🏛️
                  {% when 'sports' %}⚽
                  {% else %}📰
                {% endcase %}
              </div>
              <div class="headline-content">
                <h5 class="headline-title text--small">{{ article.title | truncate: 35 }}</h5>
                <div class="headline-time text--tiny">{{ article.published_at | date: "%I:%M %p" }}</div>
              </div>
            </div>
          </div>
        {% endfor %}
      </section>

    {% else %}
      <!-- No Content - Vertical -->
      <div class="no-content-vertical u--align-center u--space-all">
        <div class="text--large">📰</div>
        <h3 class="text--medium u--space-bottom">No News</h3>
        <p class="text--small">Feed unavailable</p>
      </div>
    {% endif %}

    <!-- Compact Footer -->
    <footer class="footer-vertical u--space-top">
      <p class="text--tiny u--align-center">{{ data.articles | size }} articles</p>
    </footer>
    
  </div>
</div>
```

### Half Horizontal Layout (800×240)
Wide but short - use horizontal space efficiently:

```html
<div id="plugin-{{ instance_id }}" class="plugin-container view--half_horizontal">
  <div class="news-feed-horizontal flex-horizontal u--space-sides u--valign-middle">
    
    {% if data.articles and data.articles.size > 0 %}
      <!-- Main Story -->
      {% assign main_story = data.articles | first %}
      <section class="main-story-horizontal flex-item">
        <article class="story-horizontal">
          <div class="story-meta u--space-bottom-small">
            <span class="category-compact">
              {% case main_story.category %}
                {% when 'technology' %}🔬
                {% when 'business' %}💼
                {% when 'environment' %}🌍
                {% when 'politics' %}🏛️
                {% when 'sports' %}⚽
                {% else %}📰
              {% endcase %}
            </span>
            <span class="story-category text--small">{{ main_story.category | upcase }}</span>
            {% if main_story.priority == 'high' %}
              <span class="breaking-compact">🔥</span>
            {% endif %}
          </div>
          
          <h3 class="story-title-horizontal text--medium">
            {{ main_story.title | truncate: 45 }}
          </h3>
          
          <p class="story-time-horizontal text--small">
            {{ main_story.published_at | date: "%I:%M %p" }}
          </p>
        </article>
      </section>

      <!-- Ticker Headlines -->
      <section class="ticker-section flex-item flex-grow">
        <div class="ticker-header u--align-center u--space-bottom-small">
          <h4 class="text--medium">Headlines</h4>
        </div>
        
        <div class="ticker-content">
          {% assign ticker_articles = data.articles | offset: 1 %}
          {% for article in ticker_articles limit: 3 %}
            <div class="ticker-item">
              <span class="ticker-icon">
                {% case article.category %}
                  {% when 'technology' %}🔬
                  {% when 'business' %}💼
                  {% when 'environment' %}🌍
                  {% when 'politics' %}🏛️
                  {% when 'sports' %}⚽
                  {% else %}📰
                {% endcase %}
              </span>
              <span class="ticker-title text--small">{{ article.title | truncate: 40 }}</span>
              <span class="ticker-time text--tiny">{{ article.published_at | date: "%I:%M" }}</span>
            </div>
          {% endfor %}
        </div>
      </section>

      <!-- Status Panel -->
      <section class="status-horizontal flex-item u--align-right">
        <div class="status-panel u--align-center">
          <div class="status-time text--medium">{{ timestamp | date: "%I:%M" }}</div>
          <div class="status-period text--small">{{ timestamp | date: "%p" }}</div>
          <div class="status-count text--tiny">{{ data.articles | size }} stories</div>
        </div>
      </section>

    {% else %}
      <!-- No Content - Horizontal -->
      <div class="no-content-horizontal flex-horizontal u--valign-middle">
        <div class="no-content-icon flex-item text--large">📰</div>
        <div class="no-content-text flex-item">
          <span class="text--medium">News feed unavailable</span>
        </div>
        <div class="no-content-time flex-item u--align-right">
          <span class="text--small">{{ timestamp | date: "%I:%M %p" }}</span>
        </div>
      </div>
    {% endif %}
    
  </div>
</div>
```

### Quadrant Layout (400×240)
Smallest space - show only essential information:

```html
<div id="plugin-{{ instance_id }}" class="plugin-container view--quadrant">
  <div class="news-feed-mini u--space-all u--align-center">
    
    {% if data.articles and data.articles.size > 0 %}
      {% assign featured = data.articles | first %}
      
      <!-- Mini Header -->
      <div class="mini-header u--space-bottom">
        <div class="mini-title text--medium">📰 News</div>
        <div class="mini-count text--small">{{ data.articles | size }} stories</div>
      </div>

      <!-- Featured Story -->
      <div class="featured-mini u--space-bottom">
        <div class="story-category-mini u--space-bottom-small">
          <span class="category-icon-mini">
            {% case featured.category %}
              {% when 'technology' %}🔬
              {% when 'business' %}💼
              {% when 'environment' %}🌍
              {% when 'politics' %}🏛️
              {% when 'sports' %}⚽
              {% else %}📰
            {% endcase %}
          </span>
          {% if featured.priority == 'high' %}
            <span class="breaking-mini">🔥</span>
          {% endif %}
        </div>
        
        <h4 class="story-title-mini text--medium u--space-bottom-small">
          {{ featured.title | truncate: 30 }}
        </h4>
        
        <div class="story-time-mini text--small">
          {{ featured.published_at | date: "%I:%M %p" }}
        </div>
      </div>

      <!-- Quick Headlines -->
      {% if data.articles.size > 1 %}
        <div class="headlines-mini">
          <div class="headlines-count text--tiny u--space-bottom-small">
            +{{ data.articles.size | minus: 1 }} more headlines
          </div>
          
          {% assign next_story = data.articles[1] %}
          <div class="next-headline text--small">
            {% case next_story.category %}
              {% when 'technology' %}🔬
              {% when 'business' %}💼
              {% when 'environment' %}🌍
              {% when 'politics' %}🏛️
              {% when 'sports' %}⚽
              {% else %}📰
            {% endcase %}
            {{ next_story.title | truncate: 25 }}
          </div>
        </div>
      {% endif %}

    {% else %}
      <!-- No Content - Mini -->
      <div class="no-content-mini u--align-center">
        <div class="text--medium">📰</div>
        <div class="text--small u--space-top-small">News</div>
        <div class="text--tiny">Unavailable</div>
      </div>
    {% endif %}
    
  </div>
</div>
```

## Design Patterns

### Content Prioritization Strategy
```html
<!-- Show different levels of detail based on available space -->

<!-- Full Layout: Rich detail -->
<div class="full-detail">
  <h1>{{ article.title }}</h1>
  <p>{{ article.summary }}</p>
  <div class="metadata">
    <span>{{ article.author }}</span>
    <span>{{ article.published_at | date: "%B %d, %Y at %I:%M %p" }}</span>
    <span>{{ article.read_time }} min read</span>
    <span>{{ article.source }}</span>
  </div>
</div>

<!-- Half Layouts: Medium detail -->
<div class="medium-detail">
  <h2>{{ article.title | truncate: 50 }}</h2>
  <p>{{ article.summary | truncate: 100 }}</p>
  <span>{{ article.published_at | date: "%I:%M %p" }}</span>
</div>

<!-- Quadrant Layout: Essential only -->
<div class="minimal-detail">
  <h3>{{ article.title | truncate: 30 }}</h3>
  <span>{{ article.published_at | date: "%I:%M" }}</span>
</div>
```

### Responsive Icon Strategy
```html
<!-- Icons that adapt to layout constraints -->

<!-- Full size: Icon + Label -->
<div class="category-full">
  <span class="icon">🔬</span>
  <span class="label">Technology</span>
</div>

<!-- Medium: Icon + Abbreviated -->
<div class="category-medium">
  <span class="icon">🔬</span>
  <span class="label">TECH</span>
</div>

<!-- Small: Icon only -->
<div class="category-small">
  <span class="icon">🔬</span>
</div>
```

### Flexible Grid System
```html
<!-- Responsive grid that adapts to content -->

<!-- Full: 2-3 columns -->
<div class="grid-adaptive grid-3-columns">
  <div class="grid-item">Content 1</div>
  <div class="grid-item">Content 2</div>
  <div class="grid-item">Content 3</div>
</div>

<!-- Half Vertical: 1 column -->
<div class="grid-adaptive grid-1-column">
  <div class="grid-item">Content 1</div>
  <div class="grid-item">Content 2</div>
</div>

<!-- Half Horizontal: Horizontal flow -->
<div class="grid-adaptive grid-horizontal">
  <div class="grid-item">Content 1</div>
  <div class="grid-item">Content 2</div>
  <div class="grid-item">Content 3</div>
</div>

<!-- Quadrant: Minimal grid -->
<div class="grid-adaptive grid-minimal">
  <div class="grid-item">Content</div>
</div>
```

## CSS Framework Utilization

### Layout-Aware Styling
```css
/* Use TRMNL framework classes that respond to layout */

/* Typography scales */
.view--full .title { font-size: var(--text-huge); }
.view--half_vertical .title { font-size: var(--text-large); }
.view--half_horizontal .title { font-size: var(--text-medium); }
.view--quadrant .title { font-size: var(--text-medium); }

/* Spacing adjustments */
.view--full .content { padding: var(--space-large); }
.view--half_vertical .content { padding: var(--space-medium); }
.view--half_horizontal .content { padding: var(--space-small); }
.view--quadrant .content { padding: var(--space-small); }

/* Content hiding for space constraints */
.view--quadrant .secondary-content { display: none; }
.view--half_horizontal .detailed-meta { display: none; }
```

### Utility Class Combinations
```html
<!-- Combine TRMNL classes for responsive behavior -->

<!-- Full layout: Rich spacing and typography -->
<div class="u--space-all u--pad-all">
  <h1 class="text--huge u--space-bottom">Title</h1>
  <p class="text--medium u--space-bottom">Content</p>
  <div class="text--small">Metadata</div>
</div>

<!-- Compact layouts: Reduced spacing -->
<div class="u--space-sides u--pad-sides">
  <h2 class="text--large u--space-bottom-small">Title</h2>
  <p class="text--small">Content</p>
</div>

<!-- Mini layout: Minimal spacing -->
<div class="u--space-all-small">
  <h3 class="text--medium">Title</h3>
  <p class="text--small">Content</p>
</div>
```

## Best Practices

### Content Strategy
1. **Progressive Enhancement**: Start with minimal content, add detail for larger layouts
2. **Information Hierarchy**: Most important content first, supporting details later
3. **Graceful Degradation**: Ensure core message is clear even in smallest layout

### Performance Optimization
1. **Conditional Content**: Only render what fits the layout
2. **Efficient Loops**: Limit iterations based on available space
3. **Smart Truncation**: Cut content intelligently, not arbitrarily

### User Experience
1. **Consistent Navigation**: Similar interaction patterns across layouts
2. **Clear Visual Hierarchy**: Maintain importance levels across sizes
3. **Readable Typography**: Ensure text is legible at all sizes

### E-ink Considerations
1. **High Contrast**: Black text on white background
2. **Bold Typography**: Use font weights that render well on e-ink
3. **Static Content**: Avoid animations or dynamic effects
4. **Clear Borders**: Use borders and spacing to separate content areas

## Testing Strategy

### Layout Testing Checklist
- [ ] Content displays correctly in all four layouts
- [ ] Text remains readable at all sizes
- [ ] Important information is never cut off
- [ ] Visual hierarchy is maintained
- [ ] No content overlap or collision
- [ ] Performance is acceptable with sample data
- [ ] Error states display properly

### Test Data Scenarios
```json
{
  "test_scenarios": [
    {
      "name": "empty_feed",
      "articles": []
    },
    {
      "name": "single_article", 
      "articles": [{"title": "Short Title", "summary": "Brief summary"}]
    },
    {
      "name": "long_titles",
      "articles": [
        {
          "title": "This Is An Extremely Long Article Title That May Cause Layout Issues",
          "summary": "Very long summary text that could potentially break the layout if not handled properly with appropriate truncation and responsive design patterns."
        }
      ]
    },
    {
      "name": "many_articles",
      "articles": [/* 20+ articles for testing limits */]
    }
  ]
}
```

This multi-layout example demonstrates how to create truly responsive private plugins that provide an excellent user experience regardless of screen size or layout configuration.