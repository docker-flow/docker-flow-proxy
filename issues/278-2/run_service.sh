#!/bin/bash

SERVICE_NAME="web"
SERVICE_YML=`pwd`/services/web/site.yml
export SRVC_PATH=`pwd`/services/web/site
export SRVC_HOSTS_NAMES="web.com"
export SRVC_PORT="443"

docker stack deploy -c $SERVICE_YML $SERVICE_NAME
