require 'logger'
require 'active_support/all'
require 'i18n'
require 'ostruct'

def plugin_env(key, default = nil)
  value = ENV[key]
  return value unless value.nil? || value.empty?

  path = ENV["#{key}_FILE"]
  if path && !path.empty?
    begin
      return File.read(path).strip
    rescue SystemCallError => e
      warn "[plugins] could not read #{key}_FILE at #{path}: #{e.message}"
    end
  end

  default
end

module Rails
  class << self
    def logger
      @logger ||= Logger.new($stdout).tap do |l|
        l.level = ENV.fetch('PLUGIN_LOG_LEVEL', 'warn').upcase
        l.formatter = ->(sev, _t, _p, msg) { "[plugins] #{sev} #{msg}\n" }
      end
    end

    def application = @application ||= Application.new
    def root = @root ||= Pathname.new(ENV.fetch('PLUGIN_ROOT', '/app/plugins')).parent
  end

  class Application
    def credentials = @credentials ||= Credentials.new
  end

  class Credentials
    def base_url = plugin_env('ASSET_BASE_URL', 'http://stationmaster:8000')
    def plugins = @plugins ||= PluginCredentials.new
    def method_missing(name, *) = plugin_env(name.to_s.upcase)
    def respond_to_missing?(*) = true
  end

  class PluginCredentials
    def github_commit_graph_token = plugin_env('GITHUB_API_TOKEN')
    def marketdata_app = plugin_env('MARKETDATA_API_TOKEN')
    def currency_api = plugin_env('CURRENCY_API_KEY')
    def google = oauth_pair('GOOGLE_CLIENT_ID', 'GOOGLE_CLIENT_SECRET')
    def todoist = oauth_pair('TODOIST_CLIENT_ID', 'TODOIST_CLIENT_SECRET')
    def full_calendar = OpenStruct.new(license_key: plugin_env('FULL_CALENDAR_LICENSE_KEY').to_s)

    def [](key) = respond_to?(key.to_sym) ? public_send(key.to_sym) : nil
    def to_h = {}
    def method_missing(name, *) = nil
    def respond_to_missing?(*) = true

    private

    def oauth_pair(id_var, secret_var)
      client_id = plugin_env(id_var)
      client_secret = plugin_env(secret_var)
      return nil unless client_id && client_secret

      { client_id: client_id, client_secret: client_secret }
    end
  end
end

I18n.backend = I18n::Backend::Simple.new
I18n.load_path += Dir[File.join(ENV.fetch('PLUGIN_LOCALE_DIR', '/app/locales'), '*.yml')]
I18n.default_locale = :en
I18n.backend.load_translations
