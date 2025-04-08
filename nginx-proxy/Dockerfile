FROM nginx:alpine

ARG ENVIRONMENT=prod

ENV ENVIRONMENT=${ENVIRONMENT}

# Install dependencies
RUN apk add --no-cache ca-certificates curl apache2-utils

# Copy files
COPY nginx.conf /etc/nginx/nginx.conf
COPY generate-${ENVIRONMENT}-conf.sh /nginx-config/generate-conf.sh

# Make the script executable
RUN chmod +x /nginx-config/generate-conf.sh

# Specify the port to be used
EXPOSE 8080

# Nginx startup command with configuration generation
CMD ["sh", "-c", "/nginx-config/generate-conf.sh /etc/nginx/nginx.conf && nginx -g 'daemon off;'"] 