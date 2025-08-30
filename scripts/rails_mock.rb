#!/usr/bin/env ruby

# Rails Mock for TRMNL Plugins
# Provides minimal Rails interface needed by TRMNL plugins without requiring full Rails

require 'ostruct'
require 'date'
require 'json'

# Mock Rails module with credentials support
module Rails
  class Application
    class Credentials
      def self.base_url
        ENV['BASE_URL'] || 'https://app.usetrmnl.com'
      end
      
      def self.plugins
        OpenStruct.new({
          google: {
            client_id: ENV['GOOGLE_CLIENT_ID'],
            client_secret: ENV['GOOGLE_CLIENT_SECRET']
          },
          github_commit_graph_token: ENV['GITHUB_TOKEN']
        })
      end
      
      def self.[](key)
        case key
        when :base_url
          base_url
        when :plugins
          plugins
        else
          ENV[key.to_s.upcase]
        end
      end
    end
    
    def self.credentials
      Credentials
    end
  end
  
  def self.application
    Application
  end
end

# Mock for translation helper
def t(key, options = {})
  # Simple translation mock - returns the last part of the key
  # In production, you could load actual translations from YAML
  parts = key.to_s.split('.')
  parts.last.gsub('_', ' ').capitalize
end

# Mock for localization helper
def l(date_or_time, format: :default)
  return '' if date_or_time.nil?
  
  case format
  when :short
    date_or_time.strftime('%b %d')
  when :long
    date_or_time.strftime('%B %d, %Y')
  else
    date_or_time.to_s
  end
end

# Base class for TRMNL plugins
module Plugins
  class Base
    attr_reader :plugin_settings, :user, :settings
    
    def initialize(plugin_settings = nil)
      @plugin_settings = plugin_settings || OpenStruct.new
      
      # Handle settings - could be a hash or OpenStruct
      if plugin_settings.respond_to?(:settings)
        @settings = plugin_settings.settings || {}
      else
        @settings = plugin_settings || {}
      end
      
      # Mock user object with timezone support
      @user = OpenStruct.new({
        datetime_now: Time.now,
        timezone: ENV['TZ'] || 'UTC'
      })
    end
    
    # Default locals method - should be overridden by specific plugins
    def locals
      {}
    end
    
    # Helper method for date handling
    def today
      @user.datetime_now.to_date
    end
    
    # Helper for instance name (used in templates)
    def instance_name
      @plugin_settings.respond_to?(:name) ? @plugin_settings.name : 'Plugin Instance'
    end
  end
end

# Mock PluginSetting for compatibility
class PluginSetting
  attr_accessor :settings, :created_at, :name, :user
  
  def initialize(attrs = {})
    @settings = attrs[:settings] || {}
    @created_at = attrs[:created_at] || Time.now
    @name = attrs[:name] || 'Plugin Instance'
    @user = attrs[:user] || OpenStruct.new(datetime_now: Time.now, timezone: 'UTC')
  end
end