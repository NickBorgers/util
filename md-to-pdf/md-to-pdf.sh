#!/bin/bash
set -e

if [ -z "$1" ]; then
    echo "Usage: md-to-pdf <input.md> [output.pdf]"
    echo ""
    echo "Converts markdown to PDF using Pandoc and headless Chromium."
    echo "If output.pdf is not specified, uses input filename with .pdf extension."
    exit 1
fi

INPUT="$1"
OUTPUT="${2:-${INPUT%.md}.pdf}"

if [ ! -f "$INPUT" ]; then
    echo "Error: Input file '$INPUT' not found"
    exit 1
fi

# Use custom CSS if present in working directory, otherwise use default
if [ -f "print-style.css" ]; then
    CSS_FILE="print-style.css"
else
    CSS_FILE="/opt/print-style.css"
fi

TEMP_HTML="/tmp/md-to-pdf-$$-$(date +%s).html"
trap "rm -f $TEMP_HTML" EXIT

echo "Converting $INPUT -> HTML..."
pandoc "$INPUT" \
    --standalone \
    --embed-resources \
    --css="$CSS_FILE" \
    -o "$TEMP_HTML"

echo "Converting HTML -> $OUTPUT..."
chromium-browser \
    --headless \
    --no-sandbox \
    --disable-gpu \
    --disable-software-rasterizer \
    --disable-dev-shm-usage \
    --print-to-pdf="$OUTPUT" \
    --no-pdf-header-footer \
    "$TEMP_HTML"

echo "Done: $OUTPUT"
