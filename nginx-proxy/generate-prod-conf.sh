#!/bin/sh

if [ -z "$1" ]; then
    echo "Usage: $0 <path_to_nginx_conf>"
    exit 1
fi

NGINX_CONF="$1"

# Create a backup of the original file
cp "$NGINX_CONF" "${NGINX_CONF}.bak"

# Add CORS headers for the CoinGecko prices endpoint
sed -i '/location = \/v1\/leaderboard\/prices {/a\
            # CORS headers\
            add_header '\''Access-Control-Allow-Origin'\'' '\''https://topcg.callfry.com'\'' always;\
            add_header '\''Access-Control-Allow-Methods'\'' '\''GET, POST, OPTIONS'\'' always;\
            add_header '\''Access-Control-Allow-Headers'\'' '\''DNT,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Range,Authorization,If-None-Match,Accept-Encoding'\'' always;\
            add_header '\''Access-Control-Expose-Headers'\'' '\''Content-Length,Content-Range,X-Proxy-Cache,X-Response-Size,ETag,Content-Encoding,Vary'\'' always;\
\
            # Handle OPTIONS method\
            if ($request_method = '\''OPTIONS'\'') {\
                add_header '\''Access-Control-Allow-Origin'\'' '\''https://topcg.callfry.om'\'';\
                add_header '\''Access-Control-Allow-Methods'\'' '\''GET, POST, OPTIONS'\'';\
                add_header '\''Access-Control-Allow-Headers'\'' '\''DNT,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Range,Authorization,If-None-Match,Accept-Encoding'\'';\
                add_header '\''Access-Control-Expose-Headers'\'' '\''Content-Length,Content-Range,X-Proxy-Cache,X-Response-Size,ETag,Content-Encoding,Vary'\'';\
                add_header '\''Access-Control-Max-Age'\'' 1728000;\
                add_header '\''Content-Type'\'' '\''text/plain; charset=utf-8'\'';\
                add_header '\''Content-Length'\'' 0;\
                return 204;\
            }' "$NGINX_CONF"

# Add CORS headers for the CoinGecko markets endpoint
sed -i '/location = \/v1\/leaderboard\/markets {/a\
            # CORS headers\
            add_header '\''Access-Control-Allow-Origin'\'' '\''https://topcg.callfry.om'\'' always;\
            add_header '\''Access-Control-Allow-Methods'\'' '\''GET, POST, OPTIONS'\'' always;\
            add_header '\''Access-Control-Allow-Headers'\'' '\''DNT,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Range,Authorization,If-None-Match,Accept-Encoding'\'' always;\
            add_header '\''Access-Control-Expose-Headers'\'' '\''Content-Length,Content-Range,X-Proxy-Cache,X-Response-Size,ETag,Content-Encoding,Vary'\'' always;\
\
            # Handle OPTIONS method\
            if ($request_method = '\''OPTIONS'\'') {\
                add_header '\''Access-Control-Allow-Origin'\'' '\''https://topcg.callfry.om'\'';\
                add_header '\''Access-Control-Allow-Methods'\'' '\''GET, POST, OPTIONS'\'';\
                add_header '\''Access-Control-Allow-Headers'\'' '\''DNT,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Range,Authorization,If-None-Match,Accept-Encoding'\'';\
                add_header '\''Access-Control-Expose-Headers'\'' '\''Content-Length,Content-Range,X-Proxy-Cache,X-Response-Size,ETag,Content-Encoding,Vary'\'';\
                add_header '\''Access-Control-Max-Age'\'' 1728000;\
                add_header '\''Content-Type'\'' '\''text/plain; charset=utf-8'\'';\
                add_header '\''Content-Length'\'' 0;\
                return 204;\
            }' "$NGINX_CONF"

echo "Added CORS headers to $NGINX_CONF for production environment" 