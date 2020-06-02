#!/bin/bash
set -xeo pipefail

rm -rf ./broker_mount/truststore/*
rm -f client.*.jks

if [ ! -f broker_mount/truststore/server.keystore.jks ]; then
    echo ""
    echo "Creating a server keystore"
    keytool -keystore broker_mount/truststore/server.keystore.jks \
            -alias localhost \
            -validity 365 \
            -storepass test-server-keystore \
            -keypass test-server-key \
            -genkey
fi
if [ ! -f broker_mount/truststore/server.truststore.jks ]; then
    echo ""
    echo "Creating a server truststore"
    keytool -keystore broker_mount/truststore/server.truststore.jks \
            -alias localhost \
            -validity 365 \
            -storepass test-server-truststore \
            -genkey
fi

echo ""
echo "Creating a CA"
openssl req -new -x509 -keyout ca-key -out ca-cert -days 365 -passin pass:test-ca

echo ""
echo "Adding CA to client truststore"
keytool -keystore client.truststore.jks -alias CARoot -import -file ca-cert --storepass test-client-truststore

echo ""
echo "Exporting a server certificate"
keytool -keystore broker_mount/truststore/server.keystore.jks -alias localhost -certreq -file cert-file -storepass test-server-keystore

echo ""
echo "Signing server cert"
openssl x509 -req -CA ca-cert -CAkey ca-key -in cert-file -out cert-signed -days 365 -CAcreateserial -passin pass:test-ca

echo "Importing CA cert and signed cert into the keystore"
keytool -keystore broker_mount/truststore/server.keystore.jks -alias CARoot -import -file ca-cert -storepass test-server-keystore
keytool -keystore broker_mount/truststore/server.keystore.jks -alias localhost -import -file cert-signed -storepass test-server-keystore

echo ""
echo "Adding generated server cert to a client trust store"
keytool -keystore client.truststore.jks \
        -storepass test-client-truststore \
        -alias broker \
        -import \
        -file server-cert

docker-compose up
