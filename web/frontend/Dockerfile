FROM nginx:1.27-alpine

COPY web/frontend/nginx.conf /etc/nginx/conf.d/default.conf
COPY web/frontend/index.html /usr/share/nginx/html/app/index.html
COPY web/frontend/styles.css /usr/share/nginx/html/app/styles.css
COPY web/frontend/app.js /usr/share/nginx/html/app/app.js

EXPOSE 80
