FROM nginx:alpine

# Install dependencies
RUN apk add --no-cache ca-certificates curl apache2-utils

# Copy files
COPY nginx.conf /etc/nginx/nginx.conf

# Specify the port to be used
EXPOSE 8080

# Nginx startup command
CMD ["nginx", "-g", "daemon off;"] 