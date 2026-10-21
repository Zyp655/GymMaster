# ---- build ----
FROM node:22-alpine AS build
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
ARG VITE_API_URL=http://localhost:8000/v1
ENV VITE_API_URL=$VITE_API_URL
RUN npm run build

# ---- serve ----
FROM nginx:alpine
ARG VITE_API_URL=http://localhost:8000/v1
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY security-headers.conf /etc/nginx/snippets/security-headers.conf
# CSP cần origin của API (khác origin admin): lấy scheme://host[:port] từ VITE_API_URL.
# VITE_API_URL tương đối (vd /v1, API cùng origin qua reverse proxy) → 'self' là đủ.
RUN case "$VITE_API_URL" in \
      http://*|https://*) API_ORIGIN=$(echo "$VITE_API_URL" | sed -E 's#^(https?://[^/]+).*#\1#') ;; \
      *) API_ORIGIN="" ;; \
    esac && \
    sed -i "s#__API_ORIGIN__#${API_ORIGIN}#g" /etc/nginx/snippets/security-headers.conf && \
    nginx -t
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
