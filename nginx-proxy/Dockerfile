FROM openresty/openresty:alpine

# Install dependencies
RUN apk add --no-cache ca-certificates curl apache2-utils gettext

# Copy configuration files
COPY nginx.conf /usr/local/openresty/nginx/conf/nginx.conf
COPY static.conf /usr/local/openresty/nginx/conf/static.conf
COPY auth.conf /usr/local/openresty/nginx/conf/auth.conf
COPY lua /usr/local/openresty/nginx/lua

# Copy static files
COPY static /usr/share/nginx/static

# Set default base URL
ENV MARKET_BASE_URL="https://test.market.status.im"

# Specify the port to be used
EXPOSE 8080

# Nginx startup command with envsubst
CMD sh -c "set -e && \
    envsubst '\$MARKET_BASE_URL' < /usr/share/nginx/static/lists.json.template > /usr/share/nginx/static/lists.json && \
    openresty -g 'daemon off;'"
