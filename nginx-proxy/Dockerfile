FROM nginx:alpine

# Install dependencies
RUN apk add --no-cache ca-certificates curl apache2-utils gettext

# Copy configuration files
COPY nginx.conf /etc/nginx/nginx.conf
COPY static.conf /etc/nginx/static.conf

# Copy static files
COPY static /usr/share/nginx/static

# Set default base URL
ENV MARKET_BASE_URL="https://test.market.status.im"

# Specify the port to be used
EXPOSE 8080

# Nginx startup command with envsubst
CMD sh -c "envsubst '\$MARKET_BASE_URL' < /usr/share/nginx/static/lists.json.template > /usr/share/nginx/static/lists.json && \
    nginx -g 'daemon off;'" 