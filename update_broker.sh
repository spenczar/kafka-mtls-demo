#!/bin/bash

echo "Enter client username:"
read CLIENT_USERNAME
echo "Enter certificate file:"
read CERTFILE

SERVER_TRUST_STORE=broker_mount/broker.server.truststore.jks
SERVER_TRUST_STORE_PASS=abcdefgh

# Add the cert to the trust store for the server
keytool -keystore $SERVER_TRUST_STORE \
        -alias $CLIENT_USERNAME.client \
        -storepass $SERVER_TRUST_STORE_PASS \
        -import \
        -file $CERTFILE

docker-compose exec broker \
               kafka-configs.sh --bootstrap-server localhost:9092 \
               --alter \
               --broker 1 \
               --add-config listener.ssl.internal.ssl.truststore.location=/mnt/shared/broker.server.truststore.jks

echo "Broker updated."
