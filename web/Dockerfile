FROM node:22-alpine AS builder

WORKDIR /app

RUN corepack enable

COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

COPY . .

# 在构建阶段注入后端 API 地址，默认指向宿主机的 8080 端口
ENV VITE_API_URL=http://localhost:8080/api

RUN pnpm build

FROM node:22-alpine

WORKDIR /app

RUN corepack enable && pnpm add -g serve

COPY --from=builder /app/dist ./dist

EXPOSE 3001

CMD ["serve", "-s", "dist", "-l", "3001"]

