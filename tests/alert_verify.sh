#!/bin/bash
# Prueba simple de umbrales

echo "🎯 Prueba de Umbrales de Sentinel"
echo "=================================="
echo ""

# Configurar umbrales
echo "Configurando umbrales: CPU=50%, MEM=30%"

# Actualizar config.json
cat > ../config.json <<EOF
{
  "cpu_threshold": 50.0,
  "mem_threshold": 30.0,
  "check_interval": 1,
  "active_webhook": "test",
  "webhooks": {
    "test": "${WEBHOOK_URL:-http://localhost:8080/webhook}"
  }
}
EOF

echo "✅ Configuración actualizada"
echo ""

# Iniciar daemon en background
echo "▶️  Iniciando daemon..."
cd ..
timeout 30s ./sentinel daemon > test/daemon.log 2>&1 &
DAEMON_PID=$!
cd test

sleep 3
echo "✅ Daemon iniciado (PID: $DAEMON_PID)"
echo ""

# Generar carga CPU
echo "🔥 Generando carga CPU..."
timeout 10s bash load_cpu.sh 2 > /dev/null 2>&1 &
CPU_PID=$!

sleep 5

# Generar carga memoria
echo "💾 Generando carga memoria..."
timeout 10s bash load_mem.sh 200 > /dev/null 2>&1 &
MEM_PID=$!

echo ""
echo "⏳ Ejecutando prueba por 30 segundos..."
sleep 30

echo ""
echo "✅ Prueba completada"
echo ""
echo "📊 Logs del daemon:"
cat daemon.log

# Cleanup
kill $DAEMON_PID 2>/dev/null || true