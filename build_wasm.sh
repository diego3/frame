#!/usr/bin/env bash
# build_wasm.sh — compiles the engine to WebAssembly and stages it in web/
set -euo pipefail

OUT_DIR="web"

echo "Building game.wasm..."
GOOS=js GOARCH=wasm go build -o "${OUT_DIR}/game.wasm" .

echo "Copying wasm_exec.js..."
WASM_EXEC="$(go env GOROOT)/misc/wasm/wasm_exec.js"
if [ ! -f "${WASM_EXEC}" ]; then
  # Go 1.24+ moved it
  WASM_EXEC="$(go env GOROOT)/lib/wasm/wasm_exec.js"
fi
cp "${WASM_EXEC}" "${OUT_DIR}/wasm_exec.js"

echo ""
echo "Done. Serve the web/ directory with any static file server, e.g.:"
echo "  python3 -m http.server --directory web 8080"
echo "Then open http://localhost:8080 in your browser."
