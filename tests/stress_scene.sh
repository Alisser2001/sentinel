#!/bin/bash
# Script para ejecutar escenarios de estrés

set -e

echo "🔥 SENTINEL - Prueba de Estrés"
echo "================================"
echo ""

# Colores
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuración
NUM_CPU_PROCS=${1:-4}
NUM_MEM_PROCS=${2:-3}
MEM_PER_PROC=${3:-100}  # MB
TEST_DURATION=${4:-60}  # segundos

echo "Configuración:"
echo "  - Procesos CPU: $NUM_CPU_PROCS"
echo "  - Procesos MEM: $NUM_MEM_PROCS ($MEM_PER_PROC MB cada uno)"
echo "  - Duración: $TEST_DURATION segundos"
echo ""

# Array para almacenar PIDs
declare -a PIDS

# Función de limpieza
cleanup() {
    echo -e "\n${YELLOW}🛑 Deteniendo todos los procesos...${NC}"
    for pid in "${PIDS[@]}"; do
        kill "$pid" 2>/dev/null || true
    done
    echo -e "${GREEN}✅ Limpieza completada${NC}"
    exit 0
}

trap cleanup SIGINT SIGTERM EXIT

# 1. Iniciar Sentinel TUI en background (opcional)
echo -e "${GREEN}▶️  Iniciando Sentinel TUI...${NC}"
timeout $TEST_DURATION ../sentinel tui > /dev/null 2>&1 &
SENTINEL_PID=$!
PIDS+=($SENTINEL_PID)
sleep 2

# 2. Generar carga CPU
echo -e "${GREEN}🔥 Generando carga CPU ($NUM_CPU_PROCS procesos)...${NC}"
for i in $(seq 1 $NUM_CPU_PROCS); do
    (while true; do echo "scale=1000; a(1)*4" | bc -l > /dev/null; done) &
    pid=$!
    PIDS+=($pid)
    echo "  ✓ Proceso CPU $i iniciado (PID: $pid)"
done

sleep 3

# 3. Generar carga memoria
echo -e "${GREEN}💾 Generando carga memoria ($NUM_MEM_PROCS procesos)...${NC}"
for i in $(seq 1 $NUM_MEM_PROCS); do
    python3 -c "
import time
data = bytearray($MEM_PER_PROC * 1024 * 1024)
print(f'Proceso MEM $i: ${MEM_PER_PROC}MB asignados')
while True:
    time.sleep(1)
" &
    pid=$!
    PIDS+=($pid)
    echo "  ✓ Proceso MEM $i iniciado (PID: $pid)"
done

sleep 3

# 4. Procesos con I/O intensivo
echo -e "${GREEN}💿 Generando I/O intensivo (2 procesos)...${NC}"
for i in 1 2; do
    (while true; do
        dd if=/dev/zero of=/tmp/sentinel_io_$i bs=1M count=10 2>/dev/null
        rm /tmp/sentinel_io_$i
    done) &
    pid=$!
    PIDS+=($pid)
    echo "  ✓ Proceso I/O $i iniciado (PID: $pid)"
done

echo ""
echo -e "${YELLOW}⚡ Prueba de estrés en curso...${NC}"
echo "Total de procesos: ${#PIDS[@]}"
echo "PIDs activos: ${PIDS[*]}"
echo ""

# 5. Monitorear cada 5 segundos
echo "Tiempo | Procesos | Load Avg       | Memoria Libre"
echo "-------|----------|----------------|---------------"

for t in $(seq 0 5 $TEST_DURATION); do
    procs=$(ps aux | wc -l)
    load=$(uptime | awk -F'load average:' '{print $2}')
    mem=$(free -h | awk '/^Mem:/ {print $4}')
    printf "%6ds | %8d | %14s | %13s\n" "$t" "$procs" "$load" "$mem"
    sleep 5
done

echo ""
echo -e "${GREEN}✅ Prueba completada${NC}"
echo ""
echo "Revisa:"
echo "  - Sentinel TUI para verificar que detectó todos los procesos"
echo "  - Logs del daemon para verificar alertas de umbrales"
echo "  - Estabilidad del sistema durante la prueba"