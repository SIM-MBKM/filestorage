FROM nginx:alpine

FROM nginx:alpine

# Remove default config FIRST
RUN rm -f /etc/nginx/conf.d/default.conf

# Copy custom config
COPY app.conf /etc/nginx/conf.d/app.conf

# Create log directory
RUN mkdir -p /var/log/nginx && chmod 755 /var/log/nginx

RUN chmod 444 /etc/nginx/conf.d/app.conf

RUN sed -i 's|/etc/nginx/conf.d/default.conf|/etc/nginx/conf.d/default.conf.disabled|g' /docker-entrypoint.sh 2>/dev/null || true
RUN ls -la /etc/nginx/conf.d/ && echo "=== app.conf content ===" && cat /etc/nginx/conf.d/app.conf

EXPOSE 80 443
CMD ["nginx", "-g", "daemon off;"]