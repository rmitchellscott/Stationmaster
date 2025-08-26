import React, { useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import {
  HelpCircle,
  ChevronDown,
  Copy,
  CheckCircle2,
  Code2,
  Palette,
  Database,
  Zap,
  Globe,
  Webhook,
  Shield,
  Layout,
  Book,
} from "lucide-react";

interface PrivatePluginHelpProps {
  isOpen: boolean;
  onClose: () => void;
}

export function PrivatePluginHelp({ isOpen, onClose }: PrivatePluginHelpProps) {
  const [copiedSnippet, setCopiedSnippet] = useState<string | null>(null);

  const copyToClipboard = async (text: string, snippetId: string) => {
    try {
      await navigator.clipboard.writeText(text);
      setCopiedSnippet(snippetId);
      setTimeout(() => setCopiedSnippet(null), 2000);
    } catch (error) {
      console.error('Failed to copy:', error);
    }
  };

  const CodeSnippet: React.FC<{ code: string; language?: string; snippetId: string; title?: string }> = ({ 
    code, 
    language = 'liquid', 
    snippetId,
    title 
  }) => (
    <div className="relative">
      {title && <h4 className="text-sm font-medium mb-2">{title}</h4>}
      <div className="bg-muted rounded-md p-4 relative group">
        <Button
          size="sm"
          variant="ghost"
          className="absolute top-2 right-2 opacity-0 group-hover:opacity-100 transition-opacity"
          onClick={() => copyToClipboard(code, snippetId)}
        >
          {copiedSnippet === snippetId ? (
            <CheckCircle2 className="h-4 w-4 text-green-600" />
          ) : (
            <Copy className="h-4 w-4" />
          )}
        </Button>
        <pre className="text-sm overflow-x-auto">
          <code className={`language-${language}`}>{code}</code>
        </pre>
      </div>
    </div>
  );

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="max-w-4xl max-h-[90vh] overflow-hidden">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <HelpCircle className="h-5 w-5" />
            Private Plugin Help & Reference
          </DialogTitle>
        </DialogHeader>

        <Tabs defaultValue="quick-start" className="w-full h-full overflow-hidden">
          <TabsList className="grid w-full grid-cols-6">
            <TabsTrigger value="quick-start">Quick Start</TabsTrigger>
            <TabsTrigger value="liquid">Liquid Syntax</TabsTrigger>
            <TabsTrigger value="css">CSS Classes</TabsTrigger>
            <TabsTrigger value="data">Data Strategies</TabsTrigger>
            <TabsTrigger value="layouts">Layouts</TabsTrigger>
            <TabsTrigger value="examples">Examples</TabsTrigger>
          </TabsList>

          {/* Quick Start Tab */}
          <TabsContent value="quick-start" className="overflow-y-auto max-h-[70vh]">
            <div className="space-y-6">
              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <Zap className="h-4 w-4" />
                    Getting Started
                  </CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div>
                    <h4 className="font-medium mb-2">1. Choose Your Data Strategy</h4>
                    <div className="grid grid-cols-3 gap-4 text-sm">
                      <div className="p-3 border rounded-md">
                        <div className="flex items-center gap-2 font-medium mb-1">
                          <Webhook className="h-4 w-4" />
                          Webhook
                        </div>
                        <p className="text-muted-foreground">Real-time data from external systems</p>
                      </div>
                      <div className="p-3 border rounded-md">
                        <div className="flex items-center gap-2 font-medium mb-1">
                          <Globe className="h-4 w-4" />
                          Polling
                        </div>
                        <p className="text-muted-foreground">Pull data from APIs periodically</p>
                      </div>
                      <div className="p-3 border rounded-md">
                        <div className="flex items-center gap-2 font-medium mb-1">
                          <Database className="h-4 w-4" />
                          Merge
                        </div>
                        <p className="text-muted-foreground">Use system and user data only</p>
                      </div>
                    </div>
                  </div>

                  <div>
                    <h4 className="font-medium mb-2">2. Basic Template Structure</h4>
                    <CodeSnippet
                      snippetId="basic-template"
                      code={`<div id="plugin-{{ instance_id }}" class="plugin-container view--full">
  <div class="u--space-all u--align-center">
    <h1 class="text--huge">Hello {{ user.first_name }}!</h1>
    <p class="text--medium">Welcome to Private Plugins</p>
    <div class="u--space-top">
      <p class="text--small">Updated: {{ timestamp | date: "%I:%M %p" }}</p>
    </div>
  </div>
</div>`}
                    />
                  </div>

                  <div>
                    <h4 className="font-medium mb-2">3. Test Your Plugin</h4>
                    <ol className="list-decimal list-inside space-y-2 text-sm">
                      <li>Click "Validate Templates" to check for errors</li>
                      <li>Use "Preview" to see how it renders</li>
                      <li>Test different layouts and sample data</li>
                      <li>Save and create plugin instances</li>
                    </ol>
                  </div>
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle>Essential Requirements</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="space-y-3">
                    <div className="flex items-start gap-3">
                      <Shield className="h-4 w-4 text-green-600 mt-1" />
                      <div>
                        <div className="font-medium">Container Div Required</div>
                        <div className="text-sm text-muted-foreground">
                          Must include: <code className="bg-muted px-1 rounded">id="plugin-{`{{ instance_id }}`}"</code>
                        </div>
                      </div>
                    </div>
                    <div className="flex items-start gap-3">
                      <Shield className="h-4 w-4 text-green-600 mt-1" />
                      <div>
                        <div className="font-medium">No JavaScript Allowed</div>
                        <div className="text-sm text-muted-foreground">
                          Script tags and event handlers are blocked for security
                        </div>
                      </div>
                    </div>
                    <div className="flex items-start gap-3">
                      <Layout className="h-4 w-4 text-blue-600 mt-1" />
                      <div>
                        <div className="font-medium">Use TRMNL Framework</div>
                        <div className="text-sm text-muted-foreground">
                          CSS classes are automatically included and optimized for e-ink displays
                        </div>
                      </div>
                    </div>
                  </div>
                </CardContent>
              </Card>
            </div>
          </TabsContent>

          {/* Liquid Syntax Tab */}
          <TabsContent value="liquid" className="overflow-y-auto max-h-[70vh]">
            <div className="space-y-4">
              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <Code2 className="h-4 w-4" />
                    Liquid Template Basics
                  </CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div>
                    <h4 className="font-medium mb-2">Variables</h4>
                    <CodeSnippet
                      snippetId="variables"
                      code={`{{ user.first_name }}        <!-- User's name -->
{{ device.name }}           <!-- Device name -->
{{ data.temperature }}      <!-- Webhook/polling data -->
{{ timestamp }}             <!-- Current time -->`}
                    />
                  </div>

                  <div>
                    <h4 className="font-medium mb-2">Filters</h4>
                    <CodeSnippet
                      snippetId="filters"
                      code={`{{ "hello world" | upcase }}           <!-- HELLO WORLD -->
{{ user.first_name | capitalize }}     <!-- John -->
{{ timestamp | date: "%I:%M %p" }}     <!-- 2:30 PM -->
{{ data.temperature | round: 1 }}      <!-- 72.5 -->
{{ article.title | truncate: 50 }}     <!-- Long title... -->`}
                    />
                  </div>

                  <div>
                    <h4 className="font-medium mb-2">Conditionals</h4>
                    <CodeSnippet
                      snippetId="conditionals"
                      code={`{% if data.temperature > 75 %}
  <p>It's hot! 🔥</p>
{% elsif data.temperature > 60 %}
  <p>Nice weather ☀️</p>
{% else %}
  <p>Bundle up ❄️</p>
{% endif %}`}
                    />
                  </div>

                  <div>
                    <h4 className="font-medium mb-2">Loops</h4>
                    <CodeSnippet
                      snippetId="loops"
                      code={`{% for article in data.news limit: 5 %}
  <div class="news-item">
    <h3>{{ article.title }}</h3>
    <p>{{ article.summary }}</p>
    {% if forloop.last %}<hr>{% endif %}
  </div>
{% endfor %}`}
                    />
                  </div>
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle>Available Data</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <h4 className="font-medium mb-2">System Variables</h4>
                      <ul className="space-y-1 text-sm font-mono">
                        <li>user.first_name</li>
                        <li>user.email</li>
                        <li>device.name</li>
                        <li>device.width</li>
                        <li>device.height</li>
                        <li>instance_id</li>
                        <li>timestamp</li>
                      </ul>
                    </div>
                    <div>
                      <h4 className="font-medium mb-2">Your Data</h4>
                      <ul className="space-y-1 text-sm font-mono">
                        <li>data.* (webhook/polling)</li>
                        <li>form_fields.* (settings)</li>
                      </ul>
                    </div>
                  </div>
                </CardContent>
              </Card>
            </div>
          </TabsContent>

          {/* CSS Classes Tab */}
          <TabsContent value="css" className="overflow-y-auto max-h-[70vh]">
            <div className="space-y-4">
              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <Palette className="h-4 w-4" />
                    TRMNL CSS Framework
                  </CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <Collapsible>
                    <CollapsibleTrigger className="flex items-center gap-2 w-full text-left font-medium">
                      <ChevronDown className="h-4 w-4" />
                      Typography Classes
                    </CollapsibleTrigger>
                    <CollapsibleContent className="mt-2">
                      <div className="grid grid-cols-2 gap-4 text-sm">
                        <div>
                          <div className="font-medium mb-1">Size Classes</div>
                          <ul className="space-y-1 font-mono">
                            <li>text--huge</li>
                            <li>text--large</li>
                            <li>text--medium</li>
                            <li>text--normal</li>
                            <li>text--small</li>
                            <li>text--tiny</li>
                          </ul>
                        </div>
                        <div>
                          <div className="font-medium mb-1">Style Classes</div>
                          <ul className="space-y-1 font-mono">
                            <li>text--bold</li>
                            <li>text--italic</li>
                            <li>text--underline</li>
                            <li>text--caps</li>
                          </ul>
                        </div>
                      </div>
                    </CollapsibleContent>
                  </Collapsible>

                  <Collapsible>
                    <CollapsibleTrigger className="flex items-center gap-2 w-full text-left font-medium">
                      <ChevronDown className="h-4 w-4" />
                      Spacing & Alignment
                    </CollapsibleTrigger>
                    <CollapsibleContent className="mt-2">
                      <div className="grid grid-cols-2 gap-4 text-sm">
                        <div>
                          <div className="font-medium mb-1">Spacing</div>
                          <ul className="space-y-1 font-mono">
                            <li>u--space-all</li>
                            <li>u--space-top</li>
                            <li>u--space-bottom</li>
                            <li>u--space-sides</li>
                            <li>u--pad-all</li>
                          </ul>
                        </div>
                        <div>
                          <div className="font-medium mb-1">Alignment</div>
                          <ul className="space-y-1 font-mono">
                            <li>u--align-center</li>
                            <li>u--align-left</li>
                            <li>u--align-right</li>
                            <li>u--valign-middle</li>
                          </ul>
                        </div>
                      </div>
                    </CollapsibleContent>
                  </Collapsible>

                  <Collapsible>
                    <CollapsibleTrigger className="flex items-center gap-2 w-full text-left font-medium">
                      <ChevronDown className="h-4 w-4" />
                      Layout Systems
                    </CollapsibleTrigger>
                    <CollapsibleContent className="mt-2">
                      <CodeSnippet
                        snippetId="layout-systems"
                        title="Grid Layout"
                        code={`<div class="grid-2-columns">
  <div class="column">Column 1</div>
  <div class="column">Column 2</div>
</div>

<div class="flex-horizontal">
  <div class="flex-item">Item 1</div>
  <div class="flex-item flex-grow">Item 2</div>
</div>`}
                      />
                    </CollapsibleContent>
                  </Collapsible>
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle>Common Patterns</CardTitle>
                </CardHeader>
                <CardContent>
                  <CodeSnippet
                    snippetId="common-patterns"
                    title="Card Layout Pattern"
                    code={`<div class="card u--space-bottom">
  <div class="card-header">
    <h3 class="card-title text--medium">Card Title</h3>
  </div>
  <div class="card-body u--pad-all">
    <p class="text--small">Card content goes here.</p>
  </div>
</div>`}
                  />
                </CardContent>
              </Card>
            </div>
          </TabsContent>

          {/* Data Strategies Tab */}
          <TabsContent value="data" className="overflow-y-auto max-h-[70vh]">
            <div className="space-y-4">
              <div className="grid grid-cols-3 gap-4">
                <Card>
                  <CardHeader>
                    <CardTitle className="flex items-center gap-2">
                      <Webhook className="h-4 w-4" />
                      Webhook
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-2">
                    <p className="text-sm text-muted-foreground">
                      Receive real-time data via HTTP POST requests
                    </p>
                    <div className="text-sm">
                      <div className="font-medium">Best for:</div>
                      <ul className="list-disc list-inside space-y-1 mt-1 text-muted-foreground">
                        <li>IoT sensors</li>
                        <li>Real-time alerts</li>
                        <li>Event notifications</li>
                      </ul>
                    </div>
                    <div className="text-sm">
                      <div className="font-medium">Limit:</div>
                      <div className="text-muted-foreground">2KB per request</div>
                    </div>
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader>
                    <CardTitle className="flex items-center gap-2">
                      <Globe className="h-4 w-4" />
                      Polling
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-2">
                    <p className="text-sm text-muted-foreground">
                      Fetch data from APIs at regular intervals
                    </p>
                    <div className="text-sm">
                      <div className="font-medium">Best for:</div>
                      <ul className="list-disc list-inside space-y-1 mt-1 text-muted-foreground">
                        <li>Weather APIs</li>
                        <li>News feeds</li>
                        <li>Stock prices</li>
                      </ul>
                    </div>
                    <div className="text-sm">
                      <div className="font-medium">Min interval:</div>
                      <div className="text-muted-foreground">5 minutes</div>
                    </div>
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader>
                    <CardTitle className="flex items-center gap-2">
                      <Database className="h-4 w-4" />
                      Merge
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-2">
                    <p className="text-sm text-muted-foreground">
                      Use only system data (user, device, time)
                    </p>
                    <div className="text-sm">
                      <div className="font-medium">Best for:</div>
                      <ul className="list-disc list-inside space-y-1 mt-1 text-muted-foreground">
                        <li>Welcome messages</li>
                        <li>Clocks & timers</li>
                        <li>Static content</li>
                      </ul>
                    </div>
                    <div className="text-sm">
                      <div className="font-medium">No limits</div>
                    </div>
                  </CardContent>
                </Card>
              </div>

              <Card>
                <CardHeader>
                  <CardTitle>Webhook Example</CardTitle>
                </CardHeader>
                <CardContent>
                  <CodeSnippet
                    snippetId="webhook-example"
                    language="bash"
                    title="Send data to webhook"
                    code={`curl -X POST "https://your-domain.com/api/webhooks/instance/your-instance-id" \\
  -H "Content-Type: application/json" \\
  -d '{
    "merge_variables": {
      "temperature": "72°F",
      "humidity": "45%",
      "status": "comfortable"
    }
  }'`}
                  />
                  <CodeSnippet
                    snippetId="webhook-template"
                    title="Use in template"
                    code={`<div class="weather-display">
  <div class="temp">{{ data.temperature }}</div>
  <div class="humidity">Humidity: {{ data.humidity }}</div>
  <div class="status">{{ data.status | capitalize }}</div>
</div>`}
                  />
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <Globe className="h-4 w-4" />
                    Polling Configuration
                  </CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div>
                    <h4 className="font-medium mb-2">Basic GET Request</h4>
                    <CodeSnippet
                      snippetId="polling-basic"
                      title="Simple polling setup"
                      code={`URL: https://api.weather.com/current?q=New York
Method: GET
Headers: authorization=bearer {{ api_key }}`}
                    />
                    <CodeSnippet
                      snippetId="polling-basic-template"
                      title="Use in template (single URL - direct access)"
                      code={`<div class="weather">
  <h2>{{ location }}</h2>
  <div class="temp">{{ temperature }}°</div>
  <div class="condition">{{ condition }}</div>
</div>`}
                    />
                  </div>

                  <div>
                    <h4 className="font-medium mb-2">POST Request with Body</h4>
                    <CodeSnippet
                      snippetId="polling-post"
                      title="POST request with JSON body"
                      code={`URL: https://api.example.com/search
Method: POST
Headers: content-type=application/json&authorization=bearer {{ api_key }}
Body: {"query": "{{ search_term }}", "limit": 5}`}
                    />
                  </div>

                  <div>
                    <h4 className="font-medium mb-2">Multiple URLs</h4>
                    <CodeSnippet
                      snippetId="polling-multiple"
                      title="Fetch from multiple sources"
                      code={`URL 1: https://api.weather.com/current

URL 2: https://api.news.com/headlines

Template access (multiple URLs - indexed access):
{{ IDX_0.temperature }}
{{ IDX_1[0].title }}`}
                    />
                  </div>

                  <div>
                    <h4 className="font-medium mb-2">Using Form Field Variables</h4>
                    <p className="text-sm text-muted-foreground mb-2">
                      Form field values are automatically available as merge variables in polling URLs, headers, and body.
                    </p>
                    <CodeSnippet
                      snippetId="polling-form-vars"
                      title="Form fields in polling config"
                      code={`URL: https://api.weather.com/current?q={{ location }}
Headers: authorization=bearer {{ api_key }}&units={{ temperature_unit }}
Body: {"city": "{{ location }}", "lang": "{{ language }}"}`}
                    />
                  </div>
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <Shield className="h-4 w-4" />
                    Form Fields (Plugin Settings)
                  </CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div>
                    <h4 className="font-medium mb-2">YAML Form Field Definition</h4>
                    <CodeSnippet
                      snippetId="form-fields-yaml"
                      language="yaml"
                      title="Example form fields configuration"
                      code={`- keyname: api_key
  field_type: password
  name: API Key
  description: Your weather API key
  optional: false
  help_text: Get your API key from weather.com

- keyname: location
  field_type: string
  name: Location
  description: City name
  default: New York
  placeholder: Enter city name

- keyname: temperature_unit
  field_type: select
  name: Temperature Unit
  options:
    - label: Celsius
      value: metric
    - label: Fahrenheit
      value: imperial`}
                    />
                  </div>

                  <div>
                    <h4 className="font-medium mb-2">Supported Field Types</h4>
                    <div className="grid grid-cols-2 gap-4 text-sm">
                      <div>
                        <div className="font-medium mb-2">Basic Types</div>
                        <ul className="space-y-1 font-mono text-xs">
                          <li>string - Text input</li>
                          <li>text - Multi-line textarea</li>
                          <li>number - Numeric input</li>
                          <li>password - Password field</li>
                          <li>url - URL input</li>
                          <li>code - Code editor</li>
                        </ul>
                      </div>
                      <div>
                        <div className="font-medium mb-2">Advanced Types</div>
                        <ul className="space-y-1 font-mono text-xs">
                          <li>date - Date picker</li>
                          <li>time - Time picker</li>
                          <li>time_zone - Timezone selector</li>
                          <li>select - Dropdown selection</li>
                          <li>copyable - Read-only copyable</li>
                          <li>author_bio - Author info</li>
                        </ul>
                      </div>
                    </div>
                  </div>

                  <div>
                    <h4 className="font-medium mb-2">Accessing Form Fields in Templates</h4>
                    <CodeSnippet
                      snippetId="form-fields-template"
                      title="Using form field values"
                      code={`<div class="weather-widget">
  <h2>Weather for {{ location }}</h2>
  <div class="temp">{{ weather.temperature }}°{{ temperature_unit }}</div>
  <div class="updated">API Key: {{ api_key | truncate: 8 }}...</div>
</div>`}
                    />
                  </div>
                </CardContent>
              </Card>
            </div>
          </TabsContent>

          {/* Layouts Tab */}
          <TabsContent value="layouts" className="overflow-y-auto max-h-[70vh]">
            <div className="space-y-4">
              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <Layout className="h-4 w-4" />
                    TRMNL Layout Types
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="grid grid-cols-2 gap-4">
                    <div className="space-y-4">
                      <div className="border p-4 rounded-md">
                        <div className="flex items-center gap-2 mb-2">
                          <div className="w-8 h-5 border-2 border-black bg-white"></div>
                          <span className="font-medium">Full Screen</span>
                        </div>
                        <div className="text-sm text-muted-foreground mb-2">800×480 pixels</div>
                        <div className="text-sm">Perfect for dashboards, detailed information, and rich content layouts.</div>
                      </div>

                      <div className="border p-4 rounded-md">
                        <div className="flex items-center gap-2 mb-2">
                          <div className="w-4 h-5 border-2 border-black bg-white"></div>
                          <span className="font-medium">Half Vertical</span>
                        </div>
                        <div className="text-sm text-muted-foreground mb-2">400×480 pixels</div>
                        <div className="text-sm">Ideal for side panels, narrow widgets, and supplementary information.</div>
                      </div>
                    </div>

                    <div className="space-y-4">
                      <div className="border p-4 rounded-md">
                        <div className="flex items-center gap-2 mb-2">
                          <div className="w-8 h-3 border-2 border-black bg-white"></div>
                          <span className="font-medium">Half Horizontal</span>
                        </div>
                        <div className="text-sm text-muted-foreground mb-2">800×240 pixels</div>
                        <div className="text-sm">Great for status bars, tickers, and horizontal information displays.</div>
                      </div>

                      <div className="border p-4 rounded-md">
                        <div className="flex items-center gap-2 mb-2">
                          <div className="w-4 h-3 border-2 border-black bg-white"></div>
                          <span className="font-medium">Quadrant</span>
                        </div>
                        <div className="text-sm text-muted-foreground mb-2">400×240 pixels</div>
                        <div className="text-sm">Best for small widgets, icons, and summary information.</div>
                      </div>
                    </div>
                  </div>
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle>Layout Best Practices</CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div>
                    <h4 className="font-medium mb-2">Content Strategy</h4>
                    <ul className="list-disc list-inside space-y-1 text-sm text-muted-foreground">
                      <li><strong>Full:</strong> Show all details, multiple sections, rich formatting</li>
                      <li><strong>Half Vertical:</strong> Stack content vertically, medium detail level</li>
                      <li><strong>Half Horizontal:</strong> Use horizontal space, minimal vertical content</li>
                      <li><strong>Quadrant:</strong> Essential information only, large fonts</li>
                    </ul>
                  </div>

                  <div>
                    <h4 className="font-medium mb-2">Responsive Template Pattern</h4>
                    <CodeSnippet
                      snippetId="responsive-template"
                      code={`<!-- Full layout: Detailed view -->
{% if device.width >= 800 and device.height >= 480 %}
  <div class="detailed-dashboard">
    <h1 class="text--huge">{{ title }}</h1>
    <p class="text--large">{{ full_description }}</p>
    <div class="stats-grid">...</div>
  </div>

<!-- Compact layouts: Essential info only -->  
{% else %}
  <div class="compact-widget">
    <h2 class="text--medium">{{ title | truncate: 20 }}</h2>
    <p class="text--small">{{ summary }}</p>
  </div>
{% endif %}`}
                    />
                  </div>
                </CardContent>
              </Card>
            </div>
          </TabsContent>

          {/* Examples Tab */}
          <TabsContent value="examples" className="overflow-y-auto max-h-[70vh]">
            <div className="space-y-4">
              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <Book className="h-4 w-4" />
                    Ready-to-Use Examples
                  </CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <Collapsible>
                    <CollapsibleTrigger className="flex items-center gap-2 w-full text-left font-medium">
                      <ChevronDown className="h-4 w-4" />
                      Welcome Message (Merge Strategy)
                    </CollapsibleTrigger>
                    <CollapsibleContent className="mt-2">
                      <CodeSnippet
                        snippetId="welcome-example"
                        code={`<div id="plugin-{{ instance_id }}" class="plugin-container view--full">
  <div class="welcome-message u--space-all u--align-center">
    <h1 class="text--huge u--space-bottom">
      Welcome {{ user.first_name }}! 👋
    </h1>
    <p class="text--large u--space-bottom">
      Your {{ device.name }} is ready
    </p>
    <div class="status-info u--space-top">
      <p class="text--medium">Last updated: {{ timestamp | date: "%I:%M %p" }}</p>
      <p class="text--small">{{ timestamp | date: "%A, %B %d, %Y" }}</p>
    </div>
  </div>
</div>`}
                      />
                    </CollapsibleContent>
                  </Collapsible>

                  <Collapsible>
                    <CollapsibleTrigger className="flex items-center gap-2 w-full text-left font-medium">
                      <ChevronDown className="h-4 w-4" />
                      Status Monitor (Webhook Strategy)
                    </CollapsibleTrigger>
                    <CollapsibleContent className="mt-2">
                      <CodeSnippet
                        snippetId="status-example"
                        code={`<div id="plugin-{{ instance_id }}" class="plugin-container view--full">
  <div class="status-monitor u--space-all">
    <header class="u--space-bottom u--align-center">
      <h2 class="text--large">System Status</h2>
    </header>
    
    {% if data.status %}
      <main class="status-display">
        <div class="overall-status u--space-bottom u--align-center">
          {% case data.status %}
            {% when 'online' %}
              <div class="status-icon text--huge">🟢</div>
              <h3 class="text--medium">All Systems Online</h3>
            {% when 'warning' %}
              <div class="status-icon text--huge">🟡</div>
              <h3 class="text--medium">Minor Issues Detected</h3>
            {% when 'offline' %}
              <div class="status-icon text--huge">🔴</div>
              <h3 class="text--medium">System Offline</h3>
          {% endcase %}
        </div>
        
        {% if data.services %}
          <div class="services-list">
            {% for service in data.services %}
              <div class="service-item flex-horizontal u--space-bottom">
                <span class="service-name flex-item">{{ service.name }}</span>
                <span class="service-status flex-item u--align-right">
                  {{ service.status | upcase }}
                </span>
              </div>
            {% endfor %}
          </div>
        {% endif %}
      </main>
    {% else %}
      <div class="no-data u--align-center">
        <p class="text--medium">Waiting for status data...</p>
      </div>
    {% endif %}
  </div>
</div>`}
                      />
                    </CollapsibleContent>
                  </Collapsible>

                  <Collapsible>
                    <CollapsibleTrigger className="flex items-center gap-2 w-full text-left font-medium">
                      <ChevronDown className="h-4 w-4" />
                      News Feed (Polling Strategy)
                    </CollapsibleTrigger>
                    <CollapsibleContent className="mt-2">
                      <CodeSnippet
                        snippetId="news-example"
                        code={`<div id="plugin-{{ instance_id }}" class="plugin-container view--full">
  <div class="news-feed u--space-all">
    <header class="u--space-bottom">
      <h2 class="text--large">📰 Latest News</h2>
    </header>
    
    {% if data.articles %}
      <main class="articles-list">
        {% for article in data.articles limit: 5 %}
          <article class="news-item u--space-bottom border-bottom">
            <h3 class="article-title text--medium u--space-bottom-small">
              {{ article.title | truncate: 80 }}
            </h3>
            <p class="article-summary text--small u--space-bottom-small">
              {{ article.summary | truncate: 150 }}
            </p>
            <div class="article-meta text--tiny">
              <span>{{ article.source }}</span>
              <span>{{ article.published_at | date: "%I:%M %p" }}</span>
            </div>
          </article>
        {% endfor %}
      </main>
    {% else %}
      <div class="no-articles u--align-center">
        <p class="text--medium">No articles available</p>
      </div>
    {% endif %}
    
    <footer class="u--space-top u--align-center">
      <p class="text--tiny">Updated: {{ timestamp | date: "%I:%M %p" }}</p>
    </footer>
  </div>
</div>`}
                      />
                    </CollapsibleContent>
                  </Collapsible>
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle>More Examples</CardTitle>
                </CardHeader>
                <CardContent>
                  <p className="text-sm text-muted-foreground mb-4">
                    Find more comprehensive examples in the documentation:
                  </p>
                  <ul className="space-y-2 text-sm">
                    <li>• Weather Widget with API integration</li>
                    <li>• System Dashboard with real-time metrics</li>
                    <li>• Countdown Timer for events</li>
                    <li>• Multi-layout responsive designs</li>
                  </ul>
                </CardContent>
              </Card>
            </div>
          </TabsContent>
        </Tabs>
      </DialogContent>
    </Dialog>
  );
}