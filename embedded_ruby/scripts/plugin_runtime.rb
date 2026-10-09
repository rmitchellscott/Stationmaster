require_relative 'plugin_rails_shim'
require 'action_view'
require_relative 'plugin_discovery'

# Rails did this through Bundler.require, so plugin source could name any gem in the
# Gemfile as a top-level constant. Without it Ruby reports the miss against the
# anonymous module the plugin was evaluated into, which reads like a scoping fault
# rather than a missing require.
%w[
  httparty
  icalendar
  icalendar/recurrence
  google/apis/calendar_v3
  google/apis/analyticsdata_v1beta
  google/apis/youtube_analytics_v2
].each do |lib|
  require lib
rescue StandardError, LoadError => e
  # never fatal: a plugin that needs the gem fails on its own, and the rest keep working
  Rails.logger.warn "plugin gem unavailable: #{lib} (#{e.class}: #{e.message})"
end

# Replaces PluginDiscoveryService and PluginExecutorService from the Rails service.
module PluginRuntime
  PLUGIN_ROOT = ENV.fetch('PLUGIN_ROOT', '/app/plugins')
  LAYOUTS = %w[full half_vertical half_horizontal quadrant].freeze

  # These lived in an anonymous module the Rails controller mixed into ActionView::Base.
  module Helpers
    def git_commit_grayscale(count)
      case count.to_i
      when 0     then 'bg--white'
      when 1..3  then 'bg--gray-5'
      when 4..7  then 'bg--gray-4'
      when 8..10 then 'bg--gray-3'
      when 11..20 then 'bg--gray-2'
      else 'bg--black'
      end
    end

    def format_number(number)
      return '0' if number.nil?

      num = number.is_a?(String) ? number.to_f : number
      if num == num.to_i
        num.to_i.to_s.gsub(/(\d)(?=(\d{3})+(?!\d))/, '\1,')
      else
        format('%.2f', num).gsub(/(\d)(?=(\d{3})+\.)/, '\1,')
      end
    end

    def formulate_and_group_events_by_day(events, today_in_tz, days_to_show)
      today = case today_in_tz
              when String then Date.parse(today_in_tz)
              when Date then today_in_tz
              else today_in_tz.to_date
              end

      (0...days_to_show).each_with_object({}) do |offset, grouped|
        date = today + offset.days
        grouped[date.strftime('%A, %B %-d')] = events.select do |event|
          event_date = coerce_to_date(event[:date_time])
          event_date == date unless event_date.nil?
        end
      end
    end

    private

    def coerce_to_date(value)
      case value
      when Date then value
      when String then (Date.parse(value) rescue nil)
      when Time, DateTime then value.to_date
      else value.respond_to?(:to_date) ? value.to_date : nil
      end
    end
  end

  class << self
    # `render "plugins/calendars/common"` looks for plugins/calendars/_common.html.erb,
    # one level above where the file lives. Symlinks bridge that, and the Rails executor
    # rebuilt them every run rather than trust them to survive a copy or a checkout.
    def ensure_partial_links
      Dir.children(PLUGIN_ROOT).each do |plugin|
        views = File.join(PLUGIN_ROOT, plugin, 'views')
        next unless Dir.exist?(views)

        Dir.glob(File.join(views, '_*.html.erb')).each do |partial|
          link = File.join(PLUGIN_ROOT, plugin, File.basename(partial))
          next if File.symlink?(link) || File.exist?(link)

          begin
            File.symlink(File.join('views', File.basename(partial)), link)
          rescue SystemCallError => e
            Rails.logger.warn "could not link #{link}: #{e.message}"
          end
        end
      end
    end

    def discover
      PluginDiscoveryService.new(PLUGIN_ROOT).discover_all
    end

    # The token arrives already refreshed, which is why nothing here refreshes it.
    def dynamic_options(plugin_name, field_name, access_token)
      klass = load_plugin_class(plugin_name)
      raise "plugin not found: #{plugin_name}" if klass.nil?

      method_name = field_name.to_sym
      raise "plugin does not support fetching #{field_name}" unless klass.respond_to?(method_name)

      argument = if plugin_name == 'google_calendar'
                   { 'google_calendar' => { 'access_token' => access_token } }
                 else
                   access_token
                 end

      to_options(klass.public_send(method_name, argument))
    end

    def execute(plugin_name, layout, settings, trmnl)
      @linked ||= (ensure_partial_links; true)
      template = LAYOUTS.include?(layout) ? layout : 'full'
      template_file = File.join(PLUGIN_ROOT, plugin_name, 'views', "#{template}.html.erb")
      unless File.exist?(template_file)
        raise "no template for layout #{layout} in #{plugin_name}"
      end

      locals = run_plugin(plugin_name, settings, trmnl)
      locals['trmnl'] = trmnl
      locals['plugin_name'] = plugin_name
      locals['instance_name'] = trmnl.dig('plugin_settings', 'instance_name') || 'Plugin Instance'

      render("plugins/#{plugin_name}/views/#{template}", locals)
    end

    private

    # Each plugin is evaluated into its own module so two plugins defining the same
    # class name cannot collide.
    def load_plugin_class(plugin_name)
      plugin_file = File.join(PLUGIN_ROOT, plugin_name, "#{plugin_name}.rb")
      return nil unless File.exist?(plugin_file)

      base_path = File.join(PLUGIN_ROOT, 'base.rb')
      load_once(base_path) if File.exist?(base_path)

      Dir.glob(File.join(PLUGIN_ROOT, plugin_name, 'helpers', '*.rb')).sort.each { |f| load_once(f) }

      sandbox = Module.new
      sandbox.module_eval(File.read(plugin_file))
      find_plugin_class(sandbox)
    end

    # The frontend binds dropdowns to [{label:, value:}].
    def to_options(result)
      return [] unless result.is_a?(Array)

      result.flat_map do |item|
        if item.is_a?(Hash)
          item.map { |label, value| { label: label.to_s, value: value.to_s } }
        else
          { label: item.to_s, value: item.to_s }
        end
      end
    end

    def run_plugin(plugin_name, settings, trmnl)
      klass = load_plugin_class(plugin_name)
      raise "no plugin class found in #{plugin_name}" if klass.nil?

      instance = klass.new(settings, trmnl)
      result = if instance.respond_to?(:locals) then instance.locals
               elsif instance.respond_to?(:execute) then instance.execute(settings)
               else instance.call(settings)
               end
      stringify(result || {})
    end

    def load_once(path)
      @loaded ||= {}
      return if @loaded[path]

      Object.class_eval(File.read(path))
      @loaded[path] = true
    end

    def find_plugin_class(sandbox)
      candidates = sandbox.constants.flat_map do |const_name|
        const = sandbox.const_get(const_name)
        case const
        when Class then [const]
        when Module then const.constants.map { |c| const.const_get(c) }.grep(Class)
        else []
        end
      end
      candidates.first || (defined?(Plugins) ? Plugins.constants.map { |c| Plugins.const_get(c) }.grep(Class).first : nil)
    end

    def stringify(hash) = hash.each_with_object({}) { |(k, v), out| out[k.to_s] = v }

    def view
      @view ||= begin
        paths = ActionView::PathSet.new([ActionView::FileSystemResolver.new(File.dirname(PLUGIN_ROOT))])
        base = ActionView::Base.with_empty_template_cache.new(ActionView::LookupContext.new(paths), {}, nil)
        base.extend(Helpers)
        base
      end
    end

    # Locals must be symbol-keyed for ERB to bind them as local variables.
    def render(template, locals)
      view.render(template: template, locals: locals.transform_keys(&:to_sym))
    end
  end
end
