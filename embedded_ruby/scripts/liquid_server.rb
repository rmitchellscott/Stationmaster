#!/usr/bin/env ruby

require 'socket'
require 'json'
require 'pathname'
require 'liquid'
require 'trmnl/liquid'
require_relative 'plugin_runtime'

SOCKET_PATH = '/tmp/liquid-renderer.sock'
MAX_THREADS = 3

# Remove socket if it exists
File.delete(SOCKET_PATH) if File.exist?(SOCKET_PATH)

# Create Unix socket server
server = UNIXServer.new(SOCKET_PATH)
puts "Liquid renderer listening on #{SOCKET_PATH}"

# Thread pool to handle concurrent requests
threads = []

def handle_request(client)
  begin
    # Read JSON request from client
    request_data = client.read
    return if request_data.nil? || request_data.empty?

    request = JSON.parse(request_data)

    # 'type' is absent on Liquid requests, which predate it, so its absence means render
    response = case request['type']
               when 'execute'
                 {
                   success: true,
                   html: PluginRuntime.execute(
                     request['plugin'],
                     request['layout'] || 'full',
                     request['settings'] || {},
                     request['trmnl'] || {}
                   )
                 }
               when 'discover'
                 { success: true, plugins: PluginRuntime.discover }
               when 'dynamic_options'
                 {
                   success: true,
                   options: PluginRuntime.dynamic_options(
                     request['plugin'],
                     request['field'],
                     request['access_token']
                   )
                 }
               else
                 environment = TRMNL::Liquid.build_environment
                 template = Liquid::Template.parse(request['template'], environment: environment)
                 { success: true, html: template.render(request['data'] || {}) }
               end

    client.write(JSON.generate(response))

  rescue => e
    # Send error response
    error_response = {
      success: false,
      error: e.message,
      backtrace: e.backtrace[0..5]
    }
    client.write(JSON.generate(error_response))
    puts "Error rendering template: #{e.message}"
  ensure
    client.close unless client.closed?
  end
end

# Main server loop
loop do
  # Accept new connection
  client = server.accept

  # Clean up finished threads
  threads.delete_if { |t| !t.alive? }

  # If we have capacity, spawn new thread
  if threads.size < MAX_THREADS
    threads << Thread.new(client) do |conn|
      handle_request(conn)
    end
  else
    # At capacity, wait for a thread to finish
    threads.first.join
    # Retry accepting this client
    threads << Thread.new(client) do |conn|
      handle_request(conn)
    end
  end
end
