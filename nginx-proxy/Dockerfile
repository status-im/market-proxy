FROM nginx:alpine

# Install dependencies
RUN apk add --no-cache ca-certificates curl apache2-utils

# Copy configuration files
COPY nginx.conf /etc/nginx/nginx.conf
COPY static.conf /etc/nginx/static.conf

# Copy static files
COPY static /usr/share/nginx/static

# Specify the port to be used
EXPOSE 8080

# Nginx startup command
CMD ["nginx", "-g", "daemon off;"] 