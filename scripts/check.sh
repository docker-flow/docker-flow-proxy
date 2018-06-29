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
        wget -qO- "http://${LISTENER_ADDRESS}:8080/v1/docker-flow-swarm-listener/ping"

        if [[ $? -ne 0 ]]; then
            echo "ERROR: Unable to ping ${LISTENER_ADDRESS}"
            exit 1
        fi
    fi
fi

exit 0
