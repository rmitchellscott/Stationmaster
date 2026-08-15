class PluginDiscoveryService
  # Ported verbatim from the Rails service apart from this path: the regex-based
  # extraction of form fields, OAuth config and credential requirements is 600 lines
  # of behaviour that is cheaper to keep than to re-derive.
  def initialize(plugins_path = ENV.fetch('PLUGIN_ROOT', '/app/plugins'))
    @plugins_path = Pathname.new(plugins_path)
  end

  def discover_all
    plugins = {}
    
    return plugins unless Dir.exist?(@plugins_path)
    
    # Scan for plugin directories
    Dir.glob(@plugins_path.join('*')).each do |plugin_dir|
      next unless File.directory?(plugin_dir)
      
      plugin_name = File.basename(plugin_dir)
      plugin_metadata = discover_plugin(plugin_name, plugin_dir)
      
      if plugin_metadata
        plugins[plugin_name] = plugin_metadata
      end
    end
    
    plugins
  rescue => e
    Rails.logger.error "Plugin discovery failed: #{e.message}"
    raise "Failed to discover plugins: #{e.message}"
  end

  private

  def discover_plugin(name, plugin_dir)
    # Look for the main plugin file
    plugin_file = File.join(plugin_dir, "#{name}.rb")

    unless File.exist?(plugin_file)
      Rails.logger.warn "Plugin file not found: #{plugin_file}"
      return nil
    end

    # Read and analyze the plugin file to extract metadata
    plugin_content = File.read(plugin_file)
    
    # Check if plugin requires credentials that aren't available
    if requires_unavailable_credentials?(plugin_content, name)
      Rails.logger.warn "Skipping #{name} plugin: required credentials not configured"
      return nil
    end
    
    template_files = discover_templates(plugin_dir)
    
    metadata = {
      name: name.humanize.titleize,
      description: extract_description(plugin_content),
      author: "TRMNL",
      version: "1.0.0",
      templates: template_files,
      form_fields: extract_form_fields_with_yaml_fallback(plugin_dir, plugin_content),
      enabled: true
    }
    
    # Add OAuth configuration if plugin supports it
    oauth_config = extract_oauth_config(plugin_content, name)
    metadata[:oauth_config] = oauth_config if oauth_config
    
    metadata
  end

  def extract_description(content)
    # Look for class comment or description
    if match = content.match(/# Description: (.+)/i)
      match[1].strip
    elsif match = content.match(/class\s+\w+.*\n\s*#\s*(.+)/m)
      match[1].strip
    else
      "Plugin description not available"
    end
  end

  def extract_form_fields_with_yaml_fallback(plugin_dir, content)
    # Check for existing form_fields.yaml file first
    form_fields_file = File.join(plugin_dir, 'form_fields.yaml')
    
    if File.exist?(form_fields_file)
      begin
        yaml_content = File.read(form_fields_file)
        return { 'yaml' => yaml_content }
      rescue => e
        Rails.logger.warn "Failed to read form_fields.yaml for #{File.basename(plugin_dir)}: #{e.message}"
        # Fall back to code analysis if YAML file is invalid
      end
    end
    
    # Fall back to code analysis
    extract_form_fields(content)
  end

  def extract_form_fields(content)
    # Extract settings['key'] patterns from Ruby code to infer form fields
    settings_keys = content.scan(/settings\['([^']+)'\]/).flatten.uniq
    
    return {} if settings_keys.empty?
    
    # Build YAML array structure for form fields based on detected settings
    form_fields = settings_keys.map do |key|
      field_config = {
        'keyname' => key,
        'name' => key.humanize,
        'field_type' => infer_field_type(key, content),
        'description' => "Configuration for #{key.humanize.downcase}"
      }
      
      # Check if field should be optional based on default value patterns
      field_config['optional'] = is_field_optional?(key, content)
      
      # Add field-specific configurations
      case field_config['field_type']
      when 'password'
        field_config['placeholder'] = "Enter #{key.humanize.downcase}"
      when 'date'
        field_config['placeholder'] = 'YYYY-MM-DD'
        field_config['help_text'] = "Enter date in YYYY-MM-DD format"
      when 'select'
        options = extract_select_options(key, content)
        if options.any?
          field_config['options'] = options
        else
          # Fallback to text if no options found
          field_config['field_type'] = 'text'
          field_config['placeholder'] = "Enter #{key.humanize.downcase}"
        end
      when 'checkbox'
        field_config['help_text'] = "Check to enable"
      else
        field_config['placeholder'] = "Enter #{key.humanize.downcase}"
      end
      
      field_config
    end
    
    # Convert to YAML array format (like private plugins)
    yaml_content = form_fields.to_yaml.lines.drop(1).join # Remove "---\n" header
    { 'yaml' => yaml_content }
  end

  def discover_templates(plugin_dir)
    templates = {}
    
    # Look for ERB template files in the views directory
    views_dir = File.join(plugin_dir, 'views')
    return templates unless Dir.exist?(views_dir)
    
    template_files = Dir.glob(File.join(views_dir, "*.html.erb")).reject { |f| File.basename(f).start_with?('_') }
    
    template_files.each do |template_file|
      template_name = File.basename(template_file, '.html.erb')
      
      # Map template names to expected layout names
      layout_name = case template_name
                   when 'full'
                     'full'
                   when /half.*vertical/
                     'half_vert'  
                   when /half.*horizontal/
                     'half_horiz'
                   when 'quadrant'
                     'quadrant'
                   else
                     'full' # default to full layout
                   end
      
      # Store placeholder content to indicate layout support
      # This will make database queries show has_full = true, etc.
      templates[layout_name] = 'external'
    end
    
    templates
  end

  def infer_field_type(key, content)
    # Infer field type based on key name and usage patterns
    case key.downcase
    when /token|key|secret|password/
      'password'
    when /date/
      'date'
    when /show_|display_|enable_|disable_/
      # Check if it's a boolean-like field (comparing to 'yes', true, etc.)
      if content.match(/settings\['#{Regexp.escape(key)}'\]\s*==\s*['"]yes['"]/) ||
         content.match(/settings\['#{Regexp.escape(key)}'\]\s*==\s*true/)
        'checkbox'
      else
        'text'
      end
    when /type|mode|format/
      # Check if there are case statements or conditionals that suggest options
      if content.include?("when '") && content.match(/#{Regexp.escape(key)}/)
        'select'
      else
        'text'
      end
    when /url|endpoint/
      'url'
    when /email/
      'email'
    when /count|limit|size|number/
      'number'
    else
      'text'
    end
  end

  def extract_select_options(key, content)
    # Try to extract options from case statements or conditional logic
    options = []
    
    # Look for case statements that use the key
    case_match = content.match(/case\s+#{Regexp.escape(key)}.*?when\s+'([^']+)'.*?when\s+'([^']+)'/m)
    if case_match
      options = case_match.captures
    end
    
    # Fallback to common patterns
    if options.empty?
      case key.downcase
      when /item_type/
        options = ['budgets', 'accounts']
      when /type/
        options = ['default', 'custom']
      end
    end
    
    options
  end

  def is_field_optional?(key, content)
    # Check for various patterns that indicate a field has default values or is optional
    
    # Pattern 1: Methods that provide defaults when setting is not present
    # e.g., "return default_value unless settings['key'].present?"
    if content.match(/return\s+.+\s+unless\s+settings\[['"]#{Regexp.escape(key)}['"]\]\.present\?/)
      return true
    end
    
    # Pattern 2: Conditional assignment with defaults
    # e.g., "settings['key'] || default_value"
    if content.match(/settings\[['"]#{Regexp.escape(key)}['"]\]\s*\|\|\s*/)
      return true
    end
    
    # Pattern 3: Methods that handle missing settings gracefully
    # e.g., def method_name = settings['key'] || default
    if content.match(/def\s+\w*#{Regexp.escape(key.gsub('_', '.*'))}\w*\s*=.*settings\[['"]#{Regexp.escape(key)}['"]\].*\|\|/)
      return true
    end
    
    # Pattern 4: Ternary operators with presence checks
    # e.g., "settings['key'].present? ? settings['key'] : default"
    if content.match(/settings\[['"]#{Regexp.escape(key)}['"]\]\.present\?\s*\?\s*settings\[['"]#{Regexp.escape(key)}['"]\]\s*:\s*/)
      return true
    end
    
    # Pattern 5: Checkbox fields that have explicit true/false checks should be optional by default
    if key.match?(/^(show_|display_|enable_|disable_)/) && 
       content.match(/settings\[['"]#{Regexp.escape(key)}['"]\]\s*==\s*['"]yes['"]/)
      return true
    end
    
    # Pattern 6: Check if there's a method definition that provides a default for this key
    method_name = key.gsub('_', '.*')
    if content.match(/def\s+#{method_name}.*?return\s+.*unless.*settings\[['"]#{Regexp.escape(key)}['"]\]/m)
      return true
    end
    
    false
  end

  def requires_unavailable_credentials?(plugin_content, plugin_name)
    # Extract credential references from the plugin code
    required_creds = extract_credential_requirements(plugin_content, plugin_name)

    return false if required_creds.empty?

    # Check if any required credentials are missing
    missing_creds = required_creds.select { |cred| !credential_available?(cred) }

    missing_creds.any?
  end

  def extract_credential_requirements(content, plugin_name)
    credentials = []
    
    # Pattern 1: Rails.application.credentials.plugins[:service][:key]
    # Used for OAuth credentials like client_id, client_secret
    oauth_matches = content.scan(/Rails\.application\.credentials\.plugins\[:(\w+)\]\[:(\w+)\]/)
    oauth_matches.each do |service, key|
      credentials << { type: :oauth, service: service.to_sym, key: key.to_sym }
    end
    
    # Pattern 2: Rails.application.credentials.plugins[:service] (direct API key)
    # Used for simple API keys like marketdata_app, currency_api
    api_key_matches = content.scan(/Rails\.application\.credentials\.plugins\[:(\w+)\](?!\[)/)
    api_key_matches.each do |service_match|
      service = service_match.first
      credentials << { type: :api_key, service: service.to_sym }
    end
    
    # Pattern 3: Rails.application.credentials.plugins.dig(:service)
    # Alternative way to access credentials
    dig_matches = content.scan(/Rails\.application\.credentials\.plugins(?:&)?\.dig\(:(\w+)\)/)
    dig_matches.each do |service_match|
      service = service_match.first
      credentials << { type: :api_key, service: service.to_sym }
    end
    
    # Pattern 4 (check nested BEFORE simple): Rails.application.credentials.plugins.service.key (nested dot notation)  
    # Used for OAuth-like credentials like full_calendar.license_key
    nested_dot_matches = content.scan(/Rails\.application\.credentials\.plugins\.(\w+)\.(\w+)/)
    nested_dot_matches.each do |service, key|
      credentials << { type: :oauth, service: service.to_sym, key: key.to_sym }
    end
    
    # Pattern 5 (check simple AFTER nested): Rails.application.credentials.plugins.service_name (dot notation)
    # Used for simple API keys like github_commit_graph_token, currency_api
    # Fixed regex: now properly matches single-level credentials with underscores
    dot_notation_matches = content.scan(/Rails\.application\.credentials\.plugins\.([a-z_]+)(?!\.\w)/)
    dot_notation_matches.each do |service_match|
      service = service_match.first
      credentials << { type: :api_key, service: service.to_sym }
    end
    
    # Pattern 6: ENV['SERVICE_CLIENT_ID'] and ENV['SERVICE_CLIENT_SECRET'] patterns
    # Used for OAuth credentials stored in environment variables
    env_oauth_matches = content.scan(/ENV\[['"](\w+)_CLIENT_ID['"]\]/)
    env_oauth_matches.each do |service_match|
      service = service_match.first.downcase
      credentials << { type: :env_oauth, service: service.to_sym, key: :client_id }
      credentials << { type: :env_oauth, service: service.to_sym, key: :client_secret }
    end
    
    # Pattern 7: ENV['SERVICE_API_TOKEN'] and other API key patterns
    # Used for API keys stored in environment variables
    env_api_matches = content.scan(/ENV\[['"](\w+(?:_API)?(?:_TOKEN|_KEY))['"]\]/)
    env_api_matches.each do |env_var_match|
      env_var = env_var_match.first
      credentials << { type: :env_api_key, env_var: env_var.to_sym }
    end
    
    credentials.uniq
  rescue => e
    Rails.logger.warn "Failed to extract credential requirements for #{plugin_name}: #{e.message}"
    []
  end

  def credential_available?(cred_info)
    # First try Rails credentials system
    rails_available = check_rails_credentials(cred_info)
    
    # If Rails credentials work, use them
    return true if rails_available
    
    # Fallback to direct environment variable checks
    check_environment_variables(cred_info)
  rescue => e
    Rails.logger.warn "Failed to check credential availability for #{cred_info}: #{e.message}"
    false
  end
  
  private
  
  def check_rails_credentials(cred_info)
    # Check if Rails credentials are configured
    return false unless Rails.application.credentials.respond_to?(:plugins)
    
    creds = Rails.application.credentials.plugins
    return false unless creds
    
    case cred_info[:type]
    when :oauth
      if cred_info[:key]
        # OAuth credentials with specific keys like client_id, client_secret
        service_creds = creds[cred_info[:service]]
        return false unless service_creds
        service_creds.is_a?(Hash) && service_creds[cred_info[:key]].present?
      else
        # This shouldn't happen for OAuth, but handle gracefully
        false
      end
    when :api_key
      # Check if credential exists using dot notation access pattern
      if creds.respond_to?(cred_info[:service])
        credential_value = creds.public_send(cred_info[:service])
        credential_value.present?
      else
        # Fallback to hash access for bracket notation patterns
        service_creds = creds[cred_info[:service]]
        service_creds.present? && (service_creds.is_a?(String) || service_creds.is_a?(Hash))
      end
    else
      false
    end
  rescue => e
    # If Rails credentials fail, we'll try environment variables
    false
  end
  
  def check_environment_variables(cred_info)
    case cred_info[:type]
    when :oauth
      case cred_info[:service]
      when :google
        # Check Google OAuth credentials
        case cred_info[:key]
        when :client_id
          ENV['GOOGLE_CLIENT_ID'].present?
        when :client_secret  
          ENV['GOOGLE_CLIENT_SECRET'].present?
        else
          false
        end
      when :todoist
        # Check Todoist OAuth credentials
        case cred_info[:key]
        when :client_id
          ENV['TODOIST_CLIENT_ID'].present?
        when :client_secret
          ENV['TODOIST_CLIENT_SECRET'].present?
        else
          false
        end
      else
        false
      end
    when :env_oauth
      # Handle ENV-based OAuth credentials
      case cred_info[:service]
      when :google
        case cred_info[:key]
        when :client_id
          ENV['GOOGLE_CLIENT_ID'].present?
        when :client_secret
          ENV['GOOGLE_CLIENT_SECRET'].present?
        else
          false
        end
      when :todoist
        case cred_info[:key]
        when :client_id
          ENV['TODOIST_CLIENT_ID'].present?
        when :client_secret
          ENV['TODOIST_CLIENT_SECRET'].present?
        else
          false
        end
      else
        # Generic service check - construct env var name
        service_upper = cred_info[:service].to_s.upcase
        case cred_info[:key]
        when :client_id
          ENV["#{service_upper}_CLIENT_ID"].present?
        when :client_secret
          ENV["#{service_upper}_CLIENT_SECRET"].present?
        else
          false
        end
      end
    when :env_api_key
      # Handle ENV-based API keys
      env_var = cred_info[:env_var].to_s
      ENV[env_var].present?
    when :api_key
      case cred_info[:service]
      when :github_commit_graph_token
        ENV['GITHUB_API_TOKEN'].present?
      when :currency_api
        ENV['CURRENCY_API_KEY'].present?
      when :marketdata_app
        ENV['MARKETDATA_API_TOKEN'].present?
      else
        false
      end
    else
      false
    end
  end
  
  def extract_oauth_config(content, plugin_name)
    # Look for OAuth configuration patterns in plugin class methods

    # Check for client_options method which contains OAuth config
    # Matches both "def client_options" and "def self.client_options"
    client_options_match = content.match(/def\s+(?:self\.)?client_options\s*\n(.*?)\n\s*end/m)
    unless client_options_match
      return nil
    end
    
    client_options_content = client_options_match[1]

    # Extract OAuth configuration from client_options hash
    oauth_config = {}
    
    # Extract authorization URI
    if auth_uri_match = client_options_content.match(/authorization_uri:\s*['"]([^'"]+)['"]/)
      oauth_config['auth_url'] = auth_uri_match[1]
    end
    
    # Extract token credential URI
    if token_uri_match = client_options_content.match(/token_credential_uri:\s*['"]([^'"]+)['"]/)
      oauth_config['token_url'] = token_uri_match[1]
    end
    
    # Extract scopes - handle both array and single scope formats
    if scope_match = client_options_content.match(/scope:\s*\[(.*?)\]/m)
      # Array format: scope: [Google::Apis::AnalyticsdataV1beta::AUTH_ANALYTICS_READONLY]
      scope_content = scope_match[1].strip
      if scope_content.include?('::AUTH_')
        # Extract scope constant and map to actual scope string
        scopes = extract_google_scopes(scope_content)
      else
        # Handle quoted string scopes: scope: ["data:read", "data:write"]
        scopes = scope_content.scan(/['"]([^'"]+)['"]/).flatten
      end
      oauth_config['scopes'] = scopes
    elsif scope_match = client_options_content.match(/scope:\s*['"]([^'"]+)['"]/)
      # Single scope format: scope: "data:read"
      oauth_config['scopes'] = [scope_match[1]]
    end
    
    # Determine provider based on OAuth URLs or plugin name
    provider = determine_oauth_provider(oauth_config['auth_url'], plugin_name)
    return nil unless provider
    
    oauth_config['provider'] = provider
    
    # Include actual credentials from environment variables
    case provider
    when 'google'
      client_id = ENV['GOOGLE_CLIENT_ID']
      client_secret = ENV['GOOGLE_CLIENT_SECRET']

      if client_id.present? && client_secret.present?
        oauth_config['client_id'] = client_id
        oauth_config['client_secret'] = client_secret
      else
        Rails.logger.warn "Google OAuth credentials not found in environment variables for #{plugin_name}"
        return nil
      end
    when 'todoist'
      client_id = ENV['TODOIST_CLIENT_ID']
      client_secret = ENV['TODOIST_CLIENT_SECRET']

      if client_id.present? && client_secret.present?
        oauth_config['client_id'] = client_id
        oauth_config['client_secret'] = client_secret
      else
        Rails.logger.warn "Todoist OAuth credentials not found in environment variables for #{plugin_name}"
        return nil
      end
    else
      Rails.logger.warn "Unknown OAuth provider: #{provider} for plugin #{plugin_name}"
      return nil
    end
    
    oauth_config
  rescue => e
    Rails.logger.warn "Failed to extract OAuth config from #{plugin_name}: #{e.message}"
    nil
  end
  
  def extract_google_scopes(scope_content)
    # Map Google API scope constants to actual scope URLs
    scope_mapping = {
      'AUTH_ANALYTICS_READONLY' => 'https://www.googleapis.com/auth/analytics.readonly',
      'AUTH_YT_ANALYTICS_READONLY' => 'https://www.googleapis.com/auth/yt-analytics.readonly',
      'AUTH_CALENDAR_READONLY' => 'https://www.googleapis.com/auth/calendar.readonly',
      'AUTH_CALENDAR_EVENTS_READONLY' => 'https://www.googleapis.com/auth/calendar.events.readonly'
    }

    scopes = []
    scope_mapping.each do |constant, scope_url|
      if scope_content.include?(constant)
        scopes << scope_url
      end
    end

    scopes
  end
  
  def determine_oauth_provider(auth_url, plugin_name)
    return nil unless auth_url
    
    case auth_url
    when /accounts\.google\.com/
      'google'
    when /todoist\.com/
      'todoist'
    when /github\.com/
      'github'
    when /shopify/
      'shopify'
    else
      nil
    end
  end
end