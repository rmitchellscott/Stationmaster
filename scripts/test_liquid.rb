#!/usr/bin/env ruby

# Test script for verifying trmnl-liquid functionality
# This script tests basic template rendering and TRMNL-specific features

require 'json'
require 'trmnl/liquid'

def test_basic_template
  puts "Testing basic template rendering..."
  
  template_str = "Hello {{ name }}! You have {{ count }} messages."
  data = { "name" => "Mitchell", "count" => 5 }
  
  environment = TRMNL::Liquid.build_environment
  template = Liquid::Template.parse(template_str, environment: environment)
  result = template.render(data)
  
  expected = "Hello Mitchell! You have 5 messages."
  if result == expected
    puts "✅ Basic template test passed"
  else
    puts "❌ Basic template test failed"
    puts "  Expected: #{expected}"
    puts "  Got: #{result}"
  end
end

def test_trmnl_filters
  puts "Testing TRMNL-specific filters..."
  
  # Test number_with_delimiter filter
  template_str = "{{ count | number_with_delimiter }}"
  data = { "count" => 1337 }
  
  environment = TRMNL::Liquid.build_environment
  template = Liquid::Template.parse(template_str, environment: environment)
  result = template.render(data)
  
  expected = "1,337"
  if result == expected
    puts "✅ number_with_delimiter filter test passed"
  else
    puts "❌ number_with_delimiter filter test failed"
    puts "  Expected: #{expected}"
    puts "  Got: #{result}"
  end
end

def test_complex_template
  puts "Testing complex template with nested data..."
  
  template_str = <<~TEMPLATE
    <div class="view">
      <h1>{{ user.name }}</h1>
      <p>Temperature: {{ weather.temperature }}°F</p>
      <p>Total: {{ total | number_with_delimiter }}</p>
    </div>
  TEMPLATE
  
  data = {
    "user" => { "name" => "Test User" },
    "weather" => { "temperature" => 72 },
    "total" => 12345
  }
  
  environment = TRMNL::Liquid.build_environment
  template = Liquid::Template.parse(template_str, environment: environment)
  result = template.render(data)
  
  if result.include?("Test User") && result.include?("72°F") && result.include?("12,345")
    puts "✅ Complex template test passed"
  else
    puts "❌ Complex template test failed"
    puts "Result: #{result}"
  end
end

def test_error_handling
  puts "Testing error handling..."
  
  # Test invalid template syntax
  template_str = "{{ unclosed_tag"
  
  begin
    environment = TRMNL::Liquid.build_environment
    template = Liquid::Template.parse(template_str, environment: environment)
    puts "❌ Error handling test failed - should have raised syntax error"
  rescue Liquid::SyntaxError => e
    puts "✅ Error handling test passed - caught syntax error: #{e.message}"
  end
end

if __FILE__ == $0
  puts "Testing TRMNL Liquid Implementation"
  puts "=" * 40
  
  test_basic_template
  test_trmnl_filters
  test_complex_template
  test_error_handling
  
  puts "=" * 40
  puts "Tests completed!"
end