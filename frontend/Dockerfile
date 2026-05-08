FROM node:20-alpine AS base
WORKDIR /app

# Instala dependências
FROM base AS deps
COPY package.json ./
RUN npm install

# Build de produção
FROM deps AS builder
COPY . .
RUN npm run build

# Imagem de dev (hot reload)
FROM deps AS dev
COPY . .
EXPOSE 5173
CMD ["npm", "run", "dev"]

# Servidor estático para produção
FROM nginx:alpine AS prod
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
