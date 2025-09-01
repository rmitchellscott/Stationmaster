#!/bin/bash

# Download weather icons from TRMNL help page
echo "Downloading weather icons from TRMNL..."

# Create directory
mkdir -p /Users/mitchell/github/rmitchellscott/stationmaster/images/plugins/weather/
cd /Users/mitchell/github/rmitchellscott/stationmaster/images/plugins/weather/

# Fetch the help page and extract SVG URLs
echo "Fetching weather icons help page..."
curl -s "https://help.usetrmnl.com/en/articles/11823386-weather-icons" | \
grep -o '/images/plugins/weather/[^"<]*\.svg' | \
sort | uniq | \
while read -r path; do
    filename=$(basename "$path")
    echo "Downloading $filename..."
    curl -s "https://usetrmnl.com$path" -o "$filename"
    
    # Check if download was successful
    if [ -f "$filename" ] && [ -s "$filename" ]; then
        echo "✓ Successfully downloaded $filename"
    else
        echo "✗ Failed to download $filename"
    fi
done

echo "Download complete!"
echo "Downloaded $(ls -1 | wc -l) weather icons to $(pwd)"

# List the specific moon phase icons we need
echo ""
echo "Checking for required moon phase icons:"
required_icons=(
    "wi-moon-alt-new.svg"
    "wi-moon-alt-waxing-crescent-3.svg"
    "wi-moon-alt-first-quarter.svg"
    "wi-moon-alt-waxing-gibbous-3.svg"
    "wi-moon-alt-full.svg"
    "wi-moon-alt-waning-gibbous-3.svg"
    "wi-moon-alt-third-quarter.svg"
    "wi-moon-alt-waning-crescent-3.svg"
)

for icon in "${required_icons[@]}"; do
    if [ -f "$icon" ]; then
        echo "✓ $icon found"
    else
        echo "✗ $icon missing"
    fi
done