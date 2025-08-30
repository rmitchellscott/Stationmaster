#!/usr/bin/env ruby

require 'bundler/setup'
require 'json'
require 'trmnl/liquid'

# Test different data access patterns
class DataAccessTest
  def initialize
    @environment = TRMNL::Liquid.build_environment
  end
  
  def test_bracket_notation
    puts "=== Testing Bracket Notation Access ==="
    
    template = <<~LIQUID
      Test 1: {{ [current_date].imageUrl }}
      Test 2: {{ ['current_date'].imageUrl }}
      Test 3: {{ current_date.imageUrl }}
      Test 4: {{ data['current_date'].imageUrl }}
    LIQUID
    
    data = {
      "current_date" => {
        "imageUrl" => "https://example.com/test.jpg"
      },
      "data" => {
        "current_date" => {
          "imageUrl" => "https://example.com/from-data.jpg"
        }
      }
    }
    
    puts "Template: #{template}"
    puts "Data structure: #{data.inspect}"
    
    begin
      liquid_template = Liquid::Template.parse(template, environment: @environment)
      result = liquid_template.render(data)
      puts "Result:"
      puts result
      
      if liquid_template.errors.any?
        puts "Errors: #{liquid_template.errors.inspect}"
      end
      
    rescue => e
      puts "ERROR: #{e.message}"
    end
  end
  
  def test_art_of_day_pattern
    puts "\n=== Testing Art of the Day Pattern ==="
    
    # Test the exact pattern from Art of the Day
    template = <<~LIQUID
      <img src="{{ [current_date].imageUrl }}" alt="{{ [current_date].title }}">
      <h2>{{ [current_date].title }}</h2>
      <p>by {{ [current_date].artist }}</p>
    LIQUID
    
    # What the data might look like based on polling results
    data = {
      "current_date" => "2025-08-30",  # This might be the issue!
      "2025-08-30" => {  # The actual data might be keyed by date
        "imageUrl" => "https://example.com/art.jpg",
        "title" => "Starry Night",
        "artist" => "Vincent van Gogh"
      }
    }
    
    puts "Template: #{template}"
    puts "Data structure: #{data.inspect}"
    
    begin
      liquid_template = Liquid::Template.parse(template, environment: @environment)
      result = liquid_template.render(data)
      puts "Result:"
      puts result
      
      if liquid_template.errors.any?
        puts "Errors: #{liquid_template.errors.inspect}"
      end
      
    rescue => e
      puts "ERROR: #{e.message}"
    end
  end
  
  def test_correct_pattern
    puts "\n=== Testing Correct Data Access Pattern ==="
    
    template = <<~LIQUID
      <!-- Using variable as key -->
      <img src="{{ current_data.imageUrl }}" alt="{{ current_data.title }}">
      <h2>{{ current_data.title }}</h2>
      <p>by {{ current_data.artist }}</p>
    LIQUID
    
    data = {
      "current_data" => {
        "imageUrl" => "https://example.com/art.jpg", 
        "title" => "Starry Night",
        "artist" => "Vincent van Gogh"
      }
    }
    
    puts "Template: #{template}"
    puts "Data structure: #{data.inspect}"
    
    begin
      liquid_template = Liquid::Template.parse(template, environment: @environment)
      result = liquid_template.render(data)
      puts "Result:"
      puts result
      
    rescue => e
      puts "ERROR: #{e.message}"
    end
  end
end

if __FILE__ == $0
  tester = DataAccessTest.new
  tester.test_bracket_notation
  tester.test_art_of_day_pattern  
  tester.test_correct_pattern
end