#!/bin/bash
# Script para generar carga CPU controlada

echo "🔥 Generando carga CPU..."

# Función para consumir CPU
burn_cpu() {
    while true; do
        : $((i++))
    done
}

# Número de núcleos a saturar (por defecto 2)
CORES=${1:-2}

echo "Saturando $CORES núcleo(s)..."
echo "Presiona Ctrl+C para detener..."

# Array para PIDs
PIDS=()

for i in $(seq 1 $CORES); do
    burn_cpu &
    pid=$!
    PIDS+=($pid)
    echo "Proceso $i iniciado (PID: $pid)"
done

# Limpiar al salir
trap "kill ${PIDS[@]} 2>/dev/null; echo ''; echo 'Procesos detenidos'; exit" SIGINT SIGTERM

wait