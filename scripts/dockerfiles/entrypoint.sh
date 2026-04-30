#!/bin/sh
set -e

PUID="${PUID:-0}"
PGID="${PGID:-0}"

chmod +x /app/octopus

if [ "$PUID" != "0" ] || [ "$PGID" != "0" ]; then
    chown -R "$PUID:$PGID" /app
fi

cd /app

if command -v su-exec >/dev/null 2>&1; then
    exec su-exec "$PUID:$PGID" ./octopus start
else
    echo "Warning: su-exec is not available; running as root." >&2
    exec ./octopus start
fi
