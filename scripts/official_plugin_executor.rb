#!/usr/bin/env ruby

# Official Plugin Executor
# Loads and executes TRMNL plugins, returning their locals data as JSON

require 'json'
require 'pathname'
require_relative 'rails_mock'

class PluginExecutor
  def initialize(plugin_dir = nil)
    if plugin_dir
      @plugin_dir = plugin_dir
    else
      # Try to find trmnl-plugins directory relative to script location
      script_dir = File.dirname(__FILE__)
      possible_dirs = [
        File.join(script_dir, '..', 'trmnl-plugins', 'lib'),
        File.join('/app', 'trmnl-plugins', 'lib'),
        File.join('.', 'trmnl-plugins', 'lib')
      ]
      
      @plugin_dir = possible_dirs.find { |dir| File.directory?(dir) } || possible_dirs.first
    end
  end
  
  def execute_plugin(plugin_name, settings = {})
    begin
      # Validate plugin exists
      plugin_path = File.join(@plugin_dir, plugin_name)
      plugin_file = File.join(plugin_path, "#{plugin_name}.rb")
      
      # Debug output
      STDERR.puts "[DEBUG] Plugin dir: #{@plugin_dir}"
      STDERR.puts "[DEBUG] Plugin path: #{plugin_path}"
      STDERR.puts "[DEBUG] Plugin file: #{plugin_file}"
      STDERR.puts "[DEBUG] File exists: #{File.exist?(plugin_file)}"
      
      unless File.exist?(plugin_file)
        return error_response("Plugin file not found: #{plugin_file}")
      end
      
      # Load the plugin Ruby file
      # Use absolute path to avoid relative path issues
      require plugin_file
      
      # Determine plugin class name (convert snake_case to CamelCase)
      class_name = plugin_name.split('_').map(&:capitalize).join
      
      # Find the plugin class
      plugin_class = Plugins.const_get(class_name)
      
      # Create mock plugin settings
      plugin_settings = PluginSetting.new(
        settings: settings,
        name: settings['instance_name'] || plugin_name.gsub('_', ' ').capitalize,
        created_at: Time.now
      )
      
      # Instantiate and execute plugin
      plugin = plugin_class.new(plugin_settings)
      locals = plugin.locals
      
      # Return success response with locals data
      {
        success: true,
        plugin: plugin_name,
        locals: locals,
        error: nil
      }
      
    rescue NameError => e
      error_response("Plugin class not found: Plugins::#{class_name} - #{e.message}")
    rescue StandardError => e
      error_response("Plugin execution error: #{e.message}\n#{e.backtrace.first(5).join("\n")}")
    end
  end
  
  def list_plugins
    Dir.glob(File.join(@plugin_dir, '*')).select { |f| File.directory?(f) }.map { |d| File.basename(d) }
  end
  
  private
  
  def error_response(message)
    {
      success: false,
      plugin: nil,
      locals: nil,
      error: message
    }
  end
end

# Main execution when run as script
if __FILE__ == $0
  # Read input from stdin
  input = STDIN.read
  
  begin
    request = JSON.parse(input, symbolize_names: true)
    
    unless request[:plugin]
      puts JSON.generate({
        success: false,
        error: "Missing 'plugin' key in input"
      })
      exit 1
    end
    
    plugin_name = request[:plugin]
    settings = request[:settings] || {}
    
    # Execute plugin
    executor = PluginExecutor.new
    result = executor.execute_plugin(plugin_name, settings)
    
    # Output result as JSON
    puts JSON.generate(result)
    
  rescue JSON::ParserError => e
    puts JSON.generate({
      success: false,
      error: "Invalid JSON input: #{e.message}"
    })
    exit 1
  end
end