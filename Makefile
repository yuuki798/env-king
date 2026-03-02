.PHONY: build run dev web web-build clash-install

# 构建
build:
	go build -o env-king .

# 启动服务
run: build
	./env-king server

# 开发：先启后端，再启前端（另开终端 cd web && pnpm dev）
dev: build
	./env-king server

# 前端开发
web:
	cd web && pnpm dev

# 前端构建
web-build:
	cd web && pnpm build

# Clash 安装（仅 Linux，需在项目根执行）
clash-install:
	./env-king clash install
