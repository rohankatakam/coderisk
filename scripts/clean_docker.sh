#!/bin/bash
# Clean Docker environment - remove all CodeRisk containers and volumes
# This ensures a completely fresh start for testing

set -e

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🧹 CodeRisk Docker Cleanup"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Stop all CodeRisk containers
echo ""
echo "📦 Stopping CodeRisk containers..."
docker compose down 2>/dev/null || echo "  No containers running"

# Remove CodeRisk containers (if they exist)
echo ""
echo "🗑️  Removing CodeRisk containers..."
for container in coderisk-neo4j coderisk-postgres coderisk-redis; do
    if docker ps -a --format '{{.Names}}' | grep -q "^${container}$"; then
        echo "  Removing ${container}..."
        docker rm -f ${container} 2>/dev/null || true
    fi
done

# Remove CodeRisk volumes
echo ""
echo "💾 Removing CodeRisk volumes..."
for volume in coderisk-go_neo4j_data coderisk-go_postgres_data coderisk-go_redis_data \
              coderisk_neo4j_data coderisk_postgres_data coderisk_redis_data; do
    if docker volume ls --format '{{.Name}}' | grep -q "^${volume}$"; then
        echo "  Removing ${volume}..."
        docker volume rm ${volume} 2>/dev/null || true
    fi
done

# Optional: Clean up dangling volumes
echo ""
echo "🧹 Cleaning up dangling volumes..."
docker volume prune -f 2>/dev/null || true

echo ""
echo "✅ Docker cleanup complete!"
echo ""
echo "Next steps:"
echo "  1. Run: make build"
echo "  2. Start services: docker compose up -d"
echo "  3. Wait ~10 seconds for services to initialize"
echo "  4. Run: ./crisk init-local"
echo ""
