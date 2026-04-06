# Build stage
FROM node:20-alpine AS builder

WORKDIR /app

# Set npm registry for reliable downloads (switch to npmmirror if in China mainland)
# RUN npm config set registry https://registry.npmmirror.com

COPY package.json package-lock.json* ./
RUN npm install --legacy-peer-deps || npm install --force

COPY . .
RUN npm run build

# ========================
# Runtime stage - serve with nginx
FROM nginx:alpine

COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf

# SSL cert directory — mount real certs via docker-compose volumes.
# Create the directory so nginx doesn't fail if volume is missing.
RUN mkdir -p /etc/nginx/ssl

EXPOSE 80 443

CMD ["nginx", "-g", "daemon off;"]
