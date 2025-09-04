#!/bin/sh

if [ -z "$1" ] || [ -z "$2" ]; then
    echo "Usage: $0 <path_to_nginx_conf> <cors_origin_url>"
    echo "Example: $0 /etc/nginx/nginx.conf http://localhost:3000"
    exit 1
fi

NGINX_CONF="$1"
CORS_ORIGIN="$2"

# Create a backup of the original file
cp "$NGINX_CONF" "${NGINX_CONF}.bak"

# Function to add CoinGecko local proxy endpoint
add_coingecko_local_proxy() {
    local file="$1"
    local temp_file=$(mktemp)
    
    # Find the last location block and add the new proxy location before the error page location
    awk -v origin="$CORS_ORIGIN" '
        /location = \/50x\.html/ {
            print "        # CoinGecko Local Proxy endpoint"
            print "        location /coingecko-local-proxy/ {"
            print "            proxy_hide_header Access-Control-Allow-Origin;"
            print "            proxy_hide_header Access-Control-Allow-Methods;"
            print "            proxy_hide_header Access-Control-Allow-Headers;"
            print "            proxy_hide_header Access-Control-Expose-Headers;"
            print "            proxy_hide_header Access-Control-Max-Age;"
            print "            "
            print "            rewrite ^/coingecko-local-proxy/(.*) /$1 break;"
            print "            proxy_pass https://api.coingecko.com;"
            print "            proxy_set_header Host api.coingecko.com;"
            print "            proxy_set_header X-Real-IP $remote_addr;"
            print "            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;"
            print "            proxy_set_header X-Forwarded-Proto $scheme;"
            print "            proxy_ssl_server_name on;"
            print "            proxy_ssl_verify off;"
            print "        }"
            print ""
        }
        { print }
    ' "$file" > "$temp_file"
    
    mv "$temp_file" "$file"
}

# Function to add CORS configuration to a location block
add_cors_config() {
    local file="$1"
    local location="$2"
    
    # Create a temporary file with the CORS configuration
    local temp_file=$(mktemp)

    # Extract the location block - support all location types (=, ~, ~*, prefix)
    awk -v loc="$location" '
        index($0, "location " loc " {") > 0 { p=1; print; next }
        p==1 && $0 ~ "}" { p=0; print; next }
        p==1 { print; next }
        { print }
    ' "$file" > "$temp_file"
    
    # Create a new file with the CORS headers inserted
    local new_file=$(mktemp)
    awk -v loc="$location" -v origin="$CORS_ORIGIN" '
        index($0, "location " loc " {") > 0 {
            print
            print "            # CORS headers for regular requests"
            print "            add_header '\''Access-Control-Allow-Origin'\'' '\''" origin "'\'' always;"
            print "            add_header '\''Access-Control-Allow-Methods'\'' '\''GET, POST, OPTIONS'\'' always;"
            print "            add_header '\''Access-Control-Allow-Headers'\'' '\''DNT,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Range,Authorization,If-None-Match,Accept-Encoding'\'' always;"
            print "            add_header '\''Access-Control-Expose-Headers'\'' '\''Content-Length,Content-Range,X-Proxy-Cache,X-Response-Size,ETag,Content-Encoding,Vary,Cache-Status'\'' always;"
            print ""
            print "            # Handle OPTIONS method"
            print "            if (\$request_method = '\''OPTIONS'\'') {"
            print "                add_header '\''Access-Control-Allow-Origin'\'' '\''" origin "'\'' always;"
            print "                add_header '\''Access-Control-Allow-Methods'\'' '\''GET, POST, OPTIONS'\'' always;"
            print "                add_header '\''Access-Control-Allow-Headers'\'' '\''DNT,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Range,Authorization,If-None-Match,Accept-Encoding'\'' always;"
            print "                add_header '\''Access-Control-Expose-Headers'\'' '\''Content-Length,Content-Range,X-Proxy-Cache,X-Response-Size,ETag,Content-Encoding,Vary,Cache-Status'\'' always;"
            print "                add_header '\''Access-Control-Max-Age'\'' 1728000;"
            print "                add_header '\''Content-Type'\'' '\''text/plain; charset=utf-8'\'';"
            print "                return 204;"
            print "            }"
            next
        }
        { print }
    ' "$temp_file" > "$new_file"
    
    # Replace the original file with the new one
    mv "$new_file" "$file"
    rm "$temp_file"
}

# Add CoinGecko local proxy endpoint
add_coingecko_local_proxy "$NGINX_CONF"

# Add CORS configuration for all endpoints
add_cors_config "$NGINX_CONF" "= /v1/simple/price"
add_cors_config "$NGINX_CONF" "= /v1/leaderboard/prices"
add_cors_config "$NGINX_CONF" "= /v1/leaderboard/simpleprices"
add_cors_config "$NGINX_CONF" "= /v1/leaderboard/markets"
add_cors_config "$NGINX_CONF" "/v1/coins/list"
add_cors_config "$NGINX_CONF" "/v1/coins/markets"
add_cors_config "$NGINX_CONF" "= /v1/asset_platforms"
add_cors_config "$NGINX_CONF" "/coingecko-local-proxy/"
add_cors_config "$NGINX_CONF" "~ ^/v1/coins/([^/]+)/market_chart$"
add_cors_config "$NGINX_CONF" "~ ^/v1/token_lists/([^/]+)/all\\\.json$"

echo "Added CORS configuration to $NGINX_CONF for local development environment with origin $CORS_ORIGIN"
echo "Added CoinGecko local proxy endpoint at /coingecko-local-proxy/" 