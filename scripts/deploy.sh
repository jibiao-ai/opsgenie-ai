#!/bin/bash
# ============================================================
# Cloud Agent Platform - 一键部署脚本
# 使用方式: bash scripts/deploy.sh
# ============================================================

set -e

echo "=========================================="
echo " Cloud Agent Platform 部署脚本"
echo "=========================================="

# 1. 拉取最新代码
echo "[1/5] 拉取最新代码..."
git pull origin genspark_ai_developer

# 2. 停止旧容器（保留数据卷）
echo "[2/5] 停止旧容器..."
docker compose down --remove-orphans || true

# 3. 清理旧镜像（可选，节省磁盘空间）
echo "[3/5] 清理旧构建缓存..."
docker compose build --no-cache

# 4. 启动所有服务
echo "[4/5] 启动服务..."
docker compose up -d

# 5. 等待并检查服务状态
echo "[5/5] 检查服务状态..."
sleep 15
docker compose ps

echo ""
echo "=========================================="
echo " 部署完成！"
echo " 访问地址: http://$(curl -s ifconfig.me 2>/dev/null || echo '<公网IP>')"
echo " 默认账号: admin / Admin@2024!"
echo ""
echo " 注意: 如果云平台使用域名接入，请确保宿主机 /etc/hosts"
echo "       包含对应的域名解析记录（已自动挂载到 backend 容器）"
echo "=========================================="

# 检查后端健康
echo ""
echo "后端健康检查:"
curl -s http://localhost:8080/api/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"Admin@2024!"}' | python3 -m json.tool 2>/dev/null || \
  curl -s http://localhost:8080/api/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"Admin@2024!"}'
