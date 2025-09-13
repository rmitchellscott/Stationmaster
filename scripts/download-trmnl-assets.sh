#!/bin/bash

# Script to download TRMNL CSS and JS assets at build time
# These will be self-hosted instead of loaded from external URLs

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
ASSETS_DIR="$PROJECT_ROOT/assets/trmnl"

echo "Downloading TRMNL assets for self-hosting..."
echo "Assets directory: $ASSETS_DIR"

# Create directories
mkdir -p "$ASSETS_DIR/css"
mkdir -p "$ASSETS_DIR/js"
mkdir -p "$ASSETS_DIR/fonts"
mkdir -p "$ASSETS_DIR/plugin-render"
mkdir -p "$ASSETS_DIR/images/layout"
mkdir -p "$ASSETS_DIR/images/grayscale"
mkdir -p "$ASSETS_DIR/images/borders"

# Download CSS
echo "Downloading CSS files..."
curl -s "https://usetrmnl.com/css/latest/plugins.css" -o "$ASSETS_DIR/css/plugins.css"

# Download TRMNL-specific fonts (Nico, Block, Dogica fonts)
echo "Downloading TRMNL-specific fonts..."
FONTS_TO_DOWNLOAD=(
    "BlockKie.ttf"
    "dogicapixel.ttf"
    "dogicapixelbold.ttf"
    "NicoClean-Regular.ttf"
    "NicoBold-Regular.ttf"
    "NicoPups-Regular.ttf"
)

for font in "${FONTS_TO_DOWNLOAD[@]}"; do
    echo "Downloading font: $font"
    curl -s "https://usetrmnl.com/fonts/$font" -o "$ASSETS_DIR/fonts/$font" || echo "⚠️  Warning: Failed to download $font"
done

# Automatically extract and download assets referenced in CSS/JS files
echo "Parsing CSS and JS files for usetrmnl.com asset references..."

# Function to extract and download relative URL assets from CSS/JS files
extract_and_download_relative_assets() {
    local source_dir="$1"
    local url_pattern="$2"
    local base_url="$3"
    local target_subdir="$4"
    
    # Find all CSS and JS files and extract relative URLs
    find "$source_dir" -name "*.css" -o -name "*.js" | while read -r file; do
        echo "Scanning $file for relative asset references..."
        
        # Extract URLs matching the pattern (relative paths starting with /)
        grep -oE "$url_pattern" "$file" | sort | uniq | while read -r url_match; do
            # Extract the path from different URL formats
            asset_path=""
            
            # Handle url("/path") or url('/path') or url(/path)
            if echo "$url_match" | grep -q "url("; then
                asset_path=$(echo "$url_match" | sed -E 's/url\(["'"'"']?([^"'"'"']+)["'"'"']?\)/\1/')
            # Handle src="/path" or href="/path"
            elif echo "$url_match" | grep -qE "(src|href)="; then
                asset_path=$(echo "$url_match" | sed -E 's/(src|href)=["'"'"']([^"'"'"']+)["'"'"'].*/\2/')
            fi
            
            # Only process paths that start with / (relative to root)
            if [[ -n "$asset_path" && "$asset_path" =~ ^/(.+) ]]; then
                clean_path="${BASH_REMATCH[1]}"  # Remove leading /
                
                # Create directory structure
                asset_dir=$(dirname "$ASSETS_DIR/$target_subdir/$clean_path")
                mkdir -p "$asset_dir"
                
                full_url="$base_url/$clean_path"
                echo "Downloading: $full_url -> $target_subdir/$clean_path"
                curl -s "$full_url" -o "$ASSETS_DIR/$target_subdir/$clean_path" || echo "⚠️  Warning: Failed to download $clean_path"
            fi
        done
    done
}

# Extract various asset types from downloaded CSS/JS files using relative URLs
echo "Extracting CSS url() references (fonts and images)..."
extract_and_download_relative_assets "$ASSETS_DIR" 'url\([^)]*\)' "https://usetrmnl.com" "."

echo "Extracting script src references..."  
extract_and_download_relative_assets "$ASSETS_DIR" 'src=['"'"'"][^'"'"'"]*['"'"'"]' "https://usetrmnl.com" "."

echo "Extracting href references..."
extract_and_download_relative_assets "$ASSETS_DIR" 'href=['"'"'"][^'"'"'"]*['"'"'"]' "https://usetrmnl.com" "."

# Download main JS files
echo "Downloading JavaScript files..."
curl -s "https://usetrmnl.com/js/latest/plugins.js" -o "$ASSETS_DIR/js/plugins.js"

# Parse TRMNL framework page to extract asset URLs dynamically
echo "Parsing TRMNL framework page for asset URLs..."
FRAMEWORK_HTML=$(curl -s "https://usetrmnl.com/framework")

# Extract asset URLs and download with stable names
echo "Extracting and downloading JavaScript assets..."

# Extract plugin assets (those that start with /assets/plugin-)
echo "$FRAMEWORK_HTML" | grep -o '/assets/plugin[^"]*\.js' | sort | uniq | while read -r asset_path; do
    # Get the base filename without hash (e.g., plugin_legacy.js from plugin_legacy-hash.js)
    if [[ $asset_path =~ /assets/(plugin[^-]*) ]]; then
        base_name="${BASH_REMATCH[1]}.js"
    else
        # Fallback: extract filename after last slash
        base_name=$(echo "$asset_path" | rev | cut -d'/' -f1 | rev)
    fi
    
    full_url="https://usetrmnl.com$asset_path"
    echo "Downloading $full_url -> $base_name"
    curl -s "$full_url" -o "$ASSETS_DIR/js/$base_name"
done

# Extract plugin-render assets  
echo "$FRAMEWORK_HTML" | grep -o '/assets/plugin-render/[^"]*\.js' | sort | uniq | while read -r asset_path; do
    # Extract base name (e.g., dithering.js from dithering-hash.js)
    filename=$(echo "$asset_path" | rev | cut -d'/' -f1 | rev)
    if [[ $filename =~ ^([^-]+)- ]]; then
        base_name="${BASH_REMATCH[1]}.js"
    else
        base_name="$filename"
    fi
    
    full_url="https://usetrmnl.com$asset_path"
    echo "Downloading $full_url -> $base_name"
    curl -s "$full_url" -o "$ASSETS_DIR/plugin-render/$base_name"
done

# Download Google Fonts Inter
echo "Downloading Google Fonts Inter..."
# First get the CSS which contains the font URLs
FONT_CSS=$(curl -s "https://fonts.googleapis.com/css2?family=Inter:wght@100..900&display=swap")
echo "$FONT_CSS" > "$ASSETS_DIR/fonts/inter.css"

# Extract and download font files
echo "$FONT_CSS" | grep -o 'url([^)]*' | sed 's/url(//' | while read -r font_url; do
    if [[ $font_url == https://* ]]; then
        # Get the filename from the URL (last part after /)
        font_file=$(echo "$font_url" | rev | cut -d'/' -f1 | rev)
        echo "Downloading font: $font_file"
        curl -s "$font_url" -o "$ASSETS_DIR/fonts/$font_file"
        
        # Update the CSS to use local paths
        sed -i.bak "s|$font_url|/assets/trmnl/fonts/$font_file|g" "$ASSETS_DIR/fonts/inter.css"
    fi
done

# Clean up backup file
rm -f "$ASSETS_DIR/fonts/inter.css.bak"

# Verify downloads
echo ""
echo "Verification:"
echo "============="

check_file() {
    if [ -f "$1" ] && [ -s "$1" ]; then
        echo "✓ $(basename "$1") ($(wc -c < "$1") bytes)"
    else
        echo "✗ $(basename "$1") - FAILED"
        exit 1
    fi
}

echo "CSS files:"
check_file "$ASSETS_DIR/css/plugins.css"
check_file "$ASSETS_DIR/fonts/inter.css"

echo ""
echo "JavaScript files:"
check_file "$ASSETS_DIR/js/plugins.js"
# Check for any plugin JS files that were downloaded
ls -1 "$ASSETS_DIR/js/"plugin*.js 2>/dev/null | while read -r file; do
    check_file "$file"
done

echo ""
echo "Plugin-render files:"
# Check for any plugin-render JS files that were downloaded  
ls -1 "$ASSETS_DIR/plugin-render/"*.js 2>/dev/null | while read -r file; do
    check_file "$file"
done

echo ""
echo "Font files:"
ls -la "$ASSETS_DIR/fonts/" | grep -E '\.(woff2?|ttf|otf)' || echo "No font files found (will be downloaded during CSS processing)"

echo ""
echo "Image files:"
echo "Layout images:"
ls -la "$ASSETS_DIR/images/layout/" 2>/dev/null | grep -E '\.(png|jpg|gif)' || echo "No layout images found"
echo "Grayscale images:"
ls -la "$ASSETS_DIR/images/grayscale/" 2>/dev/null | grep -E '\.(png|jpg|gif)' || echo "No grayscale images found"  
echo "Border images:"
ls -la "$ASSETS_DIR/images/borders/" 2>/dev/null | grep -E '\.(png|jpg|gif)' || echo "No border images found"

echo ""
echo "✅ Asset download complete!"
echo "Total assets: $(find "$ASSETS_DIR" -type f | wc -l) files"
echo "Total size: $(du -sh "$ASSETS_DIR" | cut -f1)"