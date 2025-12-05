#!/bin/bash
# Script maestro para ejecutar todas las pruebas de validación

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}"
echo "╔════════════════════════════════════════════╗"
echo "║   SENTINEL - Suite de Validación          ║"
echo "╚════════════════════════════════════════════╝"
echo -e "${NC}"

# Verificar que estamos en el directorio correcto
if [ ! -f "../cmd/main.go" ]; then
    echo -e "${RED}Error: Ejecuta este script desde el directorio test/${NC}"
    exit 1
fi

# Crear directorio de resultados
RESULTS_DIR="results_$(date +%Y%m%d_%H%M%S)"
mkdir -p "$RESULTS_DIR"

echo -e "${GREEN}📁 Resultados se guardarán en: $RESULTS_DIR${NC}"
echo ""

# 1. Test de Umbrales
echo -e "${YELLOW}═══ Test 1: Verificación de Umbrales ═══${NC}"
echo "Configurando umbrales CPU=50%, MEM=30%..."
go run threshold_test.go 2>&1 | tee "$RESULTS_DIR/threshold_test.log"
echo -e "${GREEN}✅ Test 1 completado${NC}"
echo ""
sleep 3

# 2. Comparación con top
echo -e "${YELLOW}═══ Test 2: Comparación con top ═══${NC}"
echo "Iniciando proceso de prueba..."
sleep 300 &
TEST_PID=$!
sleep 2
echo "Comparando métricas para PID $TEST_PID..."
go run compare_metrics.go $TEST_PID 2>&1 | tee "$RESULTS_DIR/comparison.log"
kill $TEST_PID 2>/dev/null || true
echo -e "${GREEN}✅ Test 2 completado${NC}"
echo ""
sleep 3

# 3. Carga CPU
echo -e "${YELLOW}═══ Test 3: Carga CPU Controlada ═══${NC}"
echo "Generando carga en 2 núcleos por 20 segundos..."
timeout 20 bash load_cpu.sh 2 2>&1 | tee "$RESULTS_DIR/cpu_load.log" || true
echo -e "${GREEN}✅ Test 3 completado${NC}"
echo ""
sleep 3

# 4. Carga Memoria
echo -e "${YELLOW}═══ Test 4: Carga Memoria Controlada ═══${NC}"
echo "Consumiendo 300MB por 15 segundos..."
timeout 15 bash load_mem.sh 300 2>&1 | tee "$RESULTS_DIR/mem_load.log" || true
echo -e "${GREEN}✅ Test 4 completado${NC}"
echo ""
sleep 3

# 5. Estrés Completo
echo -e "${YELLOW}═══ Test 5: Prueba de Estrés ═══${NC}"
echo "Ejecutando escenario de estrés (30 segundos)..."
bash stress_test.sh 4 3 150 30 2>&1 | tee "$RESULTS_DIR/stress_test.log" || true
echo -e "${GREEN}✅ Test 5 completado${NC}"
echo ""

# Resumen
echo -e "${BLUE}"
echo "╔════════════════════════════════════════════╗"
echo "║         Validación Completada              ║"
echo "╚════════════════════════════════════════════╝"
echo -e "${NC}"
echo ""
echo "Resultados guardados en: $RESULTS_DIR/"
echo ""
echo "Archivos generados:"
ls -lh "$RESULTS_DIR/"
echo ""
echo -e "${GREEN}✅ Todos los tests ejecutados exitosamente${NC}"