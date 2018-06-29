if [[ "$HEALTHCHECK" == "true" ]]; then

    wget -qO- "http://localhost:8080/v1/docker-flow-proxy/ping"

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
        wget -qO- "http://localhost:8080/v1/docker-flow-proxy/successfulinitreload"

        if [[ $? -ne 0 ]]; then
            echo "ERROR: Initial reload was not successful"
            exit 1
        fi
    fi
fi

exit 0
