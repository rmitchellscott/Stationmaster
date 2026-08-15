# The plugins were written against Rails and reach for four things from it: credentials,
# a logger, ActiveSupport date arithmetic, and I18n. The last two are real gems that load
# on their own, so only credentials and the logger need standing in for.
require 'logger'
require 'active_support/all'
require 'i18n'
require 'ostruct'

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
    # Same variable and default as config.GetAssetBaseURL on the Go side. It has to be
    # an address browserless can resolve, not localhost: the URLs built from it are
    # fetched by browserless when it renders the HTML, not by this process.
    def base_url = ENV.fetch('ASSET_BASE_URL', 'http://stationmaster:8000')
    def plugins = @plugins ||= PluginCredentials.new
    def method_missing(name, *) = ENV[name.to_s.upcase]
    def respond_to_missing?(*) = true
  end

  # The environment variable names do not follow from the credential names, so this
  # mirrors config/initializers/oauth_credentials.rb from the Rails service key by key.
  # An unknown key returns nil rather than raising, because templates interpolate these
  # directly and a nil renders empty where a raise would lose the whole plugin.
  class PluginCredentials
    def github_commit_graph_token = ENV['GITHUB_API_TOKEN']
    def marketdata_app = ENV['MARKETDATA_API_TOKEN']
    def currency_api = ENV['CURRENCY_API_KEY']
    def google = oauth_pair('GOOGLE_CLIENT_ID', 'GOOGLE_CLIENT_SECRET')
    def todoist = oauth_pair('TODOIST_CLIENT_ID', 'TODOIST_CLIENT_SECRET')
    def full_calendar = OpenStruct.new(license_key: ENV['FULL_CALENDAR_LICENSE_KEY'].to_s)

    def [](key) = respond_to?(key.to_sym) ? public_send(key.to_sym) : nil
    def to_h = {}
    def method_missing(name, *) = nil
    def respond_to_missing?(*) = true

    private

    def oauth_pair(id_var, secret_var)
      return nil unless ENV[id_var] && ENV[secret_var]

      { client_id: ENV[id_var], client_secret: ENV[secret_var] }
    end
  end
end

I18n.backend = I18n::Backend::Simple.new
I18n.load_path += Dir[File.join(ENV.fetch('PLUGIN_LOCALE_DIR', '/app/locales'), '*.yml')]
I18n.default_locale = :en
I18n.backend.load_translations
