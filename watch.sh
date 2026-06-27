#!/bin/bash

BIN_PATH="bin/tmp_server"
ARGS=("$@")

# Clean up on exit
trap "rm -f $BIN_PATH; pkill -P \$\$; exit" SIGINT SIGTERM

echo "Watching for changes in .go and .json files..."

PID=""

build_and_restart() {
    echo "[watcher] Building..."
    mkdir -p bin
    go build -o $BIN_PATH .
    if [ $? -eq 0 ]; then
        if [ -n "$PID" ]; then
            echo "[watcher] Build successful. Killing old server..."
            kill -TERM $PID 2>/dev/null
            wait $PID 2>/dev/null
        fi
        echo "[watcher] Starting new server..."
        ./$BIN_PATH "${ARGS[@]}" &
        PID=$!
    else
        echo "[watcher] Build failed. Keeping old server running..."
    fi
}

# Initial build and start
build_and_restart

while true; do
    # Wait for the next file change (monitoring .go and .json files)
    fswatch -1 -r -e ".*" -i "\\.go$" -i "\\.json$" . > /dev/null

    echo "[watcher] Change detected."
    build_and_restart
done
