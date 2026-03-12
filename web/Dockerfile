FROM node:22-alpine AS builder

WORKDIR /app

# 使用国内镜像源，避免连接 registry.npmjs.org 超时（可构建时覆盖：--build-arg NPM_REGISTRY=...）
ARG NPM_REGISTRY=https://registry.npmmirror.com
RUN npm config set registry $NPM_REGISTRY && npm install -g pnpm

COPY package.json pnpm-lock.yaml ./
RUN pnpm config set registry $NPM_REGISTRY && pnpm install --frozen-lockfile

COPY . .

# 在构建阶段注入后端 API 地址，默认指向宿主机的 8080 端口
ENV VITE_API_URL=http://localhost:8080/api

RUN pnpm build

FROM node:22-alpine

WORKDIR /app

ARG NPM_REGISTRY=https://registry.npmmirror.com
RUN npm config set registry ${NPM_REGISTRY} && npm install -g serve

COPY --from=builder /app/dist ./dist

EXPOSE 3001

CMD ["serve", "-s", "dist", "-l", "3001"]

