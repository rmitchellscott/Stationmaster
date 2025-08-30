#!/usr/bin/env ruby

require 'bundler/setup'
require 'json'
require 'trmnl/liquid'

# Ruby Liquid Service Wrapper
# Processes TRMNL liquid templates using the trmnl-liquid gem
# Input: JSON with 'template' and 'data' keys via stdin
# Output: Rendered HTML to stdout

class LiquidRenderer
  def initialize
    @environment = TRMNL::Liquid.build_environment
  end
  
  def print_data_structure(data, indent_level = 0, max_depth = 3)
    return if indent_level > max_depth
    indent = "  " * indent_level
    
    case data
    when Hash
      data.each do |key, value|
        case value
        when Hash
          STDERR.puts "#{indent}#{key}: Hash (#{value.keys.size} keys)"
          print_data_structure(value, indent_level + 1, max_depth)
        when Array
          STDERR.puts "#{indent}#{key}: Array (#{value.size} items)"
          if value.size <= 5 && indent_level < 3
            value.each_with_index do |item, idx|
              STDERR.puts "#{indent}  [#{idx}]: #{item.class} #{item.is_a?(String) ? "\"#{item[0..50]}#{item.length > 50 ? '...' : ''}\"" : item.to_s}"
            end
          end
        when String
          display_str = value.length > 100 ? "#{value[0..50]}... (#{value.length} chars)" : "\"#{value}\""
          STDERR.puts "#{indent}#{key}: String #{display_str}"
        else
          STDERR.puts "#{indent}#{key}: #{value.class} #{value}"
        end
      end
    when Array
      data.each_with_index do |item, idx|
        STDERR.puts "#{indent}[#{idx}]: #{item.class} #{item.is_a?(String) ? "\"#{item[0..30]}#{item.length > 30 ? '...' : ''}\"" : item.to_s}"
      end
    else
      STDERR.puts "#{indent}#{data.class}: #{data}"
    end
  end
  
  def deep_stringify_keys(obj)
    case obj
    when Hash
      obj.each_with_object({}) do |(key, value), result|
        result[key.to_s] = deep_stringify_keys(value)
      end
    when Array
      obj.map { |item| deep_stringify_keys(item) }
    else
      obj
    end
  end
  
  def render(template_str, data_hash)
    begin
      STDERR.puts "\n" + "="*80
      STDERR.puts "[DEBUG] RUBY LIQUID RENDERER - Starting template render"
      STDERR.puts "="*80
      STDERR.puts "[DEBUG] Template length: #{template_str.length} chars"
      STDERR.puts "[DEBUG] Data structure:"
      print_data_structure(data_hash, 1)
      
      # Special focus on trmnl structure if it exists
      if data_hash[:trmnl]
        STDERR.puts "[DEBUG] ===== TRMNL OBJECT DETAILED BREAKDOWN ====="
        print_data_structure(data_hash[:trmnl], 1)
      end
      
      # Look for template definitions and render calls
      template_definitions = template_str.scan(/\{%\s*template\s+([^%]+)\s*%\}/).flatten
      render_calls = template_str.scan(/\{%\s*render\s+([^%]+)\s*%\}/).flatten
      
      STDERR.puts "[DEBUG] Template definitions found: #{template_definitions.inspect}"
      STDERR.puts "[DEBUG] Render calls found: #{render_calls.inspect}"
      
      STDERR.puts "[DEBUG] Template content sections:"
      if template_str.include?("{% template")
        STDERR.puts "  - Contains template definitions: YES"
      end
      if template_str.include?("{% render")
        STDERR.puts "  - Contains render calls: YES"  
      end
      if template_str.include?("{% liquid")
        STDERR.puts "  - Contains liquid blocks: YES"
      end
      
      STDERR.puts "[DEBUG] Template preview (first 300 chars):"
      STDERR.puts template_str[0..300] + (template_str.length > 300 ? "..." : "")
      STDERR.puts "[DEBUG] Template preview (last 200 chars):"
      STDERR.puts (template_str.length > 200 ? "..." + template_str[-200..-1] : template_str)
      
      # Parse template using TRMNL environment
      STDERR.puts "[DEBUG] Parsing template with TRMNL environment"
      template = Liquid::Template.parse(template_str, environment: @environment)
      
      # Check for parsing errors
      if template.errors.any?
        STDERR.puts "[DEBUG] Template parsing errors: #{template.errors.inspect}"
      end
      
      # Render with provided data
      STDERR.puts "[DEBUG] Starting template render with data"
      # Convert symbol keys to string keys recursively for Liquid template access
      string_data = deep_stringify_keys(data_hash)
      STDERR.puts "[DEBUG] Converted data keys: #{string_data.keys.inspect}"
      rendered = template.render(string_data)
      
      # Check for render errors
      if template.errors.any?
        STDERR.puts "[DEBUG] Template rendering errors: #{template.errors.inspect}"
        return {
          success: false,
          result: nil,
          error: "Template errors during render: #{template.errors.join(', ')}"
        }
      end
      
      STDERR.puts "[DEBUG] Template rendered successfully, length: #{rendered.length}"
      STDERR.puts "[DEBUG] Rendered preview: #{rendered[0..200]}..."
      
      return {
        success: true,
        result: rendered,
        error: nil
      }
      
    rescue Liquid::SyntaxError => e
      error_msg = "Template syntax error: #{e.message}"
      STDERR.puts "[ERROR] #{error_msg}"
      STDERR.puts "[ERROR] Error location: line #{e.line_number rescue 'unknown'}" if e.respond_to?(:line_number)
      return {
        success: false,
        result: nil,
        error: error_msg
      }
      
    rescue Liquid::Error => e
      error_msg = "Liquid rendering error: #{e.message}"
      STDERR.puts "[ERROR] #{error_msg}"
      STDERR.puts "[ERROR] Backtrace: #{e.backtrace[0..5].join("\n")}"
      return {
        success: false,
        result: nil,
        error: error_msg
      }
      
    rescue StandardError => e
      error_msg = "Ruby error: #{e.message}"
      STDERR.puts "[ERROR] #{error_msg}"
      STDERR.puts "[ERROR] Error class: #{e.class}"
      STDERR.puts "[ERROR] Backtrace: #{e.backtrace[0..10].join("\n")}"
      return {
        success: false,
        result: nil,
        error: error_msg
      }
    end
  end
  
  def process_stdin
    input = STDIN.read
    
    begin
      request = JSON.parse(input, symbolize_names: true)
      
      unless request[:template]
        puts JSON.generate({
          success: false,
          result: nil,
          error: "Missing 'template' key in input JSON"
        })
        return
      end
      
      template = request[:template]
      data = request[:data] || {}
      
      # Render template
      result = render(template, data)
      
      # Output result as JSON
      puts JSON.generate(result)
      
    rescue JSON::ParserError => e
      puts JSON.generate({
        success: false,
        result: nil,
        error: "Invalid JSON input: #{e.message}"
      })
    end
  end
end

# Main execution
if __FILE__ == $0
  renderer = LiquidRenderer.new
  renderer.process_stdin
end