#!/bin/bash

ROOT_CERTS="services/web/site/ssl"
TARGET_NAME="web.com.pem"
SECRET_NAME="cert-"$TARGET_NAME

[ ! -r "$ROOT_CERTS/fullchain.pem" ]  && echo "No fullchain.pem in $ROOT_CERTS" && exit 2
[ ! -r "$ROOT_CERTS/privkey.pem" ] && echo "No privkey.pem in $ROOT_CERTS" && exit 2

cat $ROOT_CERTS/fullchain.pem $ROOT_CERTS/privkey.pem | tee $ROOT_CERTS/$TARGET_NAME
docker secret create $SECRET_NAME $ROOT_CERTS/$TARGET_NAME
[ "$?" -eq 0 ] && echo "Secret $SECRET_NAME created from $ROOT_CERTS/$TARGET_NAME"
