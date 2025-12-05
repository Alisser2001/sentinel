#!/bin/bash
# Script para generar carga mixta CPU + Memoria

echo "⚡ Generando carga mixta CPU + Memoria..."

CPU_CORES=${1:-2}
MEM_MB=${2:-256}

echo "CPU: $CPU_CORES núcleo(s), Memoria: ${MEM_MB}MB"

# Iniciar carga CPU en background
./load_cpu.sh $CPU_CORES &
CPU_PID=$!

# Esperar 2 segundos
sleep 2

# Iniciar carga memoria
./load_mem.sh $MEM_MB &
MEM_PID=$!

echo ""
echo "Procesos activos:"
echo "  - CPU: PID $CPU_PID"
echo "  - MEM: PID $MEM_PID"
echo ""
echo "Presiona Ctrl+C para detener todo..."

# Limpiar al salir
trap "kill $CPU_PID $MEM_PID 2>/dev/null; echo 'Todos los procesos detenidos'; exit" SIGINT SIGTERM

wait