#!/usr/bin/env ruby

require 'bundler/setup'
require 'json'
require 'trmnl/liquid'

# Test script to debug failing templates with real data
class FailingTemplateTest
  def initialize
    @environment = TRMNL::Liquid.build_environment
    puts "[TEST] TRMNL Liquid environment loaded"
  end
  
  def test_template(name, template_str, data_hash)
    puts "\n" + "="*60
    puts "[TEST] Testing template: #{name}"
    puts "[TEST] Template length: #{template_str.length}"
    puts "[TEST] Data keys: #{data_hash.keys.inspect}"
    puts "="*60
    
    begin
      # Parse template
      template = Liquid::Template.parse(template_str, environment: @environment)
      
      if template.errors.any?
        puts "[PARSE ERRORS] #{template.errors.inspect}"
        return false
      end
      
      # Render template
      rendered = template.render(data_hash)
      
      if template.errors.any?
        puts "[RENDER ERRORS] #{template.errors.inspect}"
        return false
      end
      
      puts "[SUCCESS] Template rendered (#{rendered.length} chars)"
      puts "[PREVIEW] #{rendered[0..300]}..."
      return true
      
    rescue => e
      puts "[ERROR] #{e.class}: #{e.message}"
      puts "[BACKTRACE] #{e.backtrace[0..3].join("\n")}"
      return false
    end
  end
  
  def test_simple_liquid_block
    puts "\n=== Testing Simple {% liquid %} Block ==="
    
    template = <<~LIQUID
      {% liquid
        assign test_var = 'Hello World'
        assign number = 42
      %}
      Result: {{ test_var }} - {{ number }}
    LIQUID
    
    data = {}
    test_template("Simple Liquid Block", template, data)
  end
  
  def test_date_operations
    puts "\n=== Testing Date Operations ==="
    
    template = <<~LIQUID
      {% liquid
        assign current_time = 'today' | date: '%s' | plus: 3600
        assign formatted_time = current_time | date: '%Y-%m-%d %H:%M:%S'
      %}
      Current time: {{ formatted_time }}
    LIQUID
    
    data = {}
    test_template("Date Operations", template, data)
  end
  
  def test_template_render_tags
    puts "\n=== Testing Template/Render Tags ==="
    
    template = <<~LIQUID
      {% template test_partial %}
      <div class="test">Hello {{ name }}!</div>
      {% endtemplate %}
      
      {% render 'test_partial', name: 'World' %}
    LIQUID
    
    data = {}
    test_template("Template/Render Tags", template, data)
  end
  
  def test_literature_clock_logic
    puts "\n=== Testing Literature Clock Logic ==="
    
    # Simplified version of the literature clock logic
    template = <<~LIQUID
      {% liquid
        assign granularity = 5
        assign granularity_sec = granularity | times: 60
        assign current_time = 'today' | date: '%s' | plus: -21600
        assign snapped_time = current_time | times: 1.0 | divided_by: granularity_sec | ceil | times: granularity_sec | date: '%Y-%m-%dT%H:%M:%S'
        assign key = snapped_time | date: '%H:%M'
      %}
      Time: {{ snapped_time }}
      Key: {{ key }}
    LIQUID
    
    data = {
      "trmnl" => {
        "user" => {
          "utc_offset" => -21600  # -6 hours in seconds
        }
      }
    }
    test_template("Literature Clock Logic", template, data)
  end
  
  def test_complex_data_access
    puts "\n=== Testing Complex Data Access ==="
    
    template = <<~LIQUID
      Image URL: {{ [current_date].imageUrl }}
      Title: {{ data.title }}
      Count: {{ items | size }}
    LIQUID
    
    data = {
      "current_date" => {
        "imageUrl" => "https://example.com/image.jpg"
      },
      "data" => {
        "title" => "Test Title"
      },
      "items" => ["a", "b", "c"]
    }
    test_template("Complex Data Access", template, data)
  end
  
  def run_all_tests
    puts "Starting Template Debug Tests"
    puts "Ruby version: #{RUBY_VERSION}"
    puts "Liquid version: #{Liquid::VERSION}"
    
    test_simple_liquid_block
    test_date_operations
    test_template_render_tags
    test_literature_clock_logic  
    test_complex_data_access
    
    puts "\n" + "="*60
    puts "All tests completed"
  end
end

if __FILE__ == $0
  tester = FailingTemplateTest.new
  tester.run_all_tests
end