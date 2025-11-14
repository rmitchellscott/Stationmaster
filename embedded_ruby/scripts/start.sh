#!/bin/sh
set -e

# Start Ruby Liquid renderer in background
ruby /app/scripts/liquid_server.rb &
RUBY_PID=$!

# Give Ruby server a moment to start
sleep 1

# Start Go application (this will run in foreground)
exec /app/stationmaster

# If Go exits, kill Ruby server
kill $RUBY_PID 2>/dev/null || true
