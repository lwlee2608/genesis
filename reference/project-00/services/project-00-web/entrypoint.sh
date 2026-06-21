#!/bin/sh

# Extract DNS resolver from /etc/resolv.conf for nginx dynamic resolution
# (needed for dynamic upstreams like docker-compose service names or
# Railway internal hostnames such as <service>.railway.internal)
RAW=$(grep nameserver /etc/resolv.conf | head -1 | awk '{print $2}')
if [ -z "$RAW" ]; then
    RAW="127.0.0.11"
fi
if echo "$RAW" | grep -q ':'; then
    export DNS_RESOLVER="[$RAW]"
else
    export DNS_RESOLVER="$RAW"
fi

echo "[entrypoint] PORT=${PORT} BACKEND_URL=${BACKEND_URL} DNS_RESOLVER=${DNS_RESOLVER}"

exec /docker-entrypoint.sh "$@"
