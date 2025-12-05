#!/bin/bash
# Script para consumir memoria controladamente

echo "💾 Generando carga de memoria..."

# Cantidad de MB a consumir (por defecto 512MB)
MEM_MB=${1:-512}

echo "Consumiendo ${MEM_MB}MB de memoria..."

python3 << EOF
import time
import signal
import sys

# Manejar Ctrl+C
def signal_handler(sig, frame):
    print('\n🛑 Liberando memoria...')
    sys.exit(0)

signal.signal(signal.SIGINT, signal_handler)

# Consumir memoria
mem_mb = $MEM_MB
chunk_size = 1024 * 1024  # 1MB
data = []

print(f'Asignando {mem_mb}MB en bloques de 1MB...')

for i in range(mem_mb):
    data.append(bytearray(chunk_size))
    if (i + 1) % 100 == 0:
        print(f'Asignados {i + 1}MB...')

print(f'✅ {mem_mb}MB asignados. Manteniendo en memoria...')
print('Presiona Ctrl+C para liberar')

# Mantener memoria asignada
while True:
    time.sleep(1)
EOF