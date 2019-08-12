if [[ "$HEALTHCHECK" == "true" ]]; then

    wget -qO- "http://localhost:${PORT:-8080}/v1/docker-flow-proxy/ping"

    if [[ $? -ne 0 ]]; then
        echo "ERROR: Failed to ping docker-flow-proxy"
        exit 1
    fi

    pgrep -x "haproxy"

    if [[ $? -ne 0 ]]; then
        echo "ERROR: haproxy process is not running"
        exit 1
    fi

    if [[ "$LISTENER_ADDRESS" != "" ]]; then

        while true; do
            wget -qO- "http://localhost:${PORT:-8080}/v1/docker-flow-proxy/successfulinitreload"

            if [[ $? -eq 0 ]]; then
                exit 0
            fi
            sleep 0.5
        done
    fi
fi

exit 0
