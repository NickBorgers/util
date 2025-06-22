#!/bin/bash

# Thanks GPT-4o

# Defaults
MODE="${MODE:-heavy}"
DURATION="${DURATION:-10}"
IO_THREADS="${IO_THREADS:-0}"

# Determine scale factor
case "$MODE" in
  heavy) SCALE=1 ;;
  medium) SCALE=0.5 ;;
  light) SCALE=0.25 ;;
  *)
    echo "Invalid MODE: $MODE. Use heavy, medium, or light."
    exit 1
    ;;
esac

# Get number of physical CPU cores or fallback to logical
get_physical_cores() {
  if command -v lscpu &>/dev/null; then
    cores=$(lscpu | awk '/^Core\(s\) per socket:/ { c=$4 } /^Socket\(s\):/ { s=$2 } END { print c * s }')
  elif [[ -f /proc/cpuinfo ]]; then
    cores=$(awk -F: '/^processor/ {++n} END {print n}' /proc/cpuinfo)
  else
    cores=$(sysctl -n hw.logicalcpu 2>/dev/null || echo 1)
  fi

  if [[ -z "$cores" || "$cores" -le 0 ]]; then
    cores=$(nproc 2>/dev/null || sysctl -n hw.logicalcpu 2>/dev/null || echo 1)
  fi

  echo "$cores"
}

# Get available memory in MB
get_available_memory_mb() {
  if [[ -f /proc/meminfo ]]; then
    awk '/^MemAvailable:/ { print int($2 / 1024) }' /proc/meminfo
  else
    vm_stat | awk '/Pages free/ {free=$3} /Pages inactive/ {inactive=$3} END {print int((free + inactive) * 4096 / 1048576)}'
  fi
}

CORES=$(get_physical_cores)
MEM_MB=$(get_available_memory_mb)

CPU_WORKERS=$(awk -v c="$CORES" -v s="$SCALE" 'BEGIN { print int(c * s) }')
MEM_TO_USE=$(awk -v m="$MEM_MB" -v s="$SCALE" 'BEGIN { print int(m * s) "M" }')
VM_WORKERS=$(( CPU_WORKERS / 2 ))
[[ "$VM_WORKERS" -lt 1 ]] && VM_WORKERS=1

echo "Running stress test:"
echo "- Mode: $MODE"
echo "- Duration: ${DURATION}s"
echo "- CPU Workers: $CPU_WORKERS"
echo "- VM Workers: $VM_WORKERS"
echo "- Memory Use: $MEM_TO_USE"
echo "- IO Threads: $IO_THREADS"

# Build stress arguments
args=(--cpu "$CPU_WORKERS" --vm "$VM_WORKERS" --vm-bytes "$MEM_TO_USE" --timeout "$DURATION")

# Add IO only if non-zero
if [[ "$IO_THREADS" -gt 0 ]]; then
  args+=(--io "$IO_THREADS")
fi

# Run stress
stress "${args[@]}"
