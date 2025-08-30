package official

import (
	"strings"
	"testing"
)

func TestTemplateConverter_ConvertERBToLiquid(t *testing.T) {
	converter := NewTemplateConverter("https://example.com")
	
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple ERB output tag",
			input:    "<%= days_left %>",
			expected: "{{ days_left }}",
		},
		{
			name:     "Rails base_url",
			input:    "<%= Rails.application.credentials.base_url %>",
			expected: "{{ base_url }}",
		},
		{
			name:     "ERB if statement",
			input:    "<% if show_days_left %>content<% end %>",
			expected: "{% if show_days_left %}content{% endif %}",
		},
		{
			name:     "Instance name",
			input:    "<%= instance_name %>",
			expected: "{{ trmnl.plugin_instance.name }}",
		},
		{
			name:     "Translation helper",
			input:    "<%= t('renders.days_left_year.days_left') %>",
			expected: "{{ 'Days Left' }}",
		},
		{
			name:     "Times loop",
			input:    "<% max_days.times do |idx| %>content<% end %>",
			expected: "{% for idx in (1..max_days) %}content{% endfor %}",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.ConvertERBToLiquid(tt.input)
			if result != tt.expected {
				t.Errorf("ConvertERBToLiquid() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestTemplateConverter_ConvertControlStructures(t *testing.T) {
	converter := NewTemplateConverter("")
	
	input := `
<% if condition %>
  <div>Content</div>
<% elsif other_condition %>
  <div>Other</div>
<% else %>
  <div>Default</div>
<% end %>
`
	
	expected := `
{% if condition %}
  <div>Content</div>
{% elsif other_condition %}
  <div>Other</div>
{% else %}
  <div>Default</div>
{% endif %}
`
	
	result := converter.convertControlStructures(input)
	if strings.TrimSpace(result) != strings.TrimSpace(expected) {
		t.Errorf("convertControlStructures() failed\nGot:\n%s\nWant:\n%s", result, expected)
	}
}

func TestTemplateConverter_ConvertPartials(t *testing.T) {
	converter := NewTemplateConverter("")
	
	input := `<%= render 'plugins/days_left_until/progress_bar', percent_passed: percent_passed %>`
	expected := `{% render 'progress_bar' percent_passed= percent_passed %}`
	
	result := converter.convertPartials(input)
	if result != expected {
		t.Errorf("convertPartials() = %q, want %q", result, expected)
	}
}