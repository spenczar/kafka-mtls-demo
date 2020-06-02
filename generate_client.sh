#!/bin/bash
set -euo pipefail

echo "Generating new client"
echo "Enter client name:"
read CLIENT_USERNAME
echo "Enter password:"
read CLIENT_PASSWORD

SERVER_TRUST_STORE="broker_mount/broker.server.truststore.jks"
SERVER_TRUST_STORE_PASS=abcdefgh

C=NN
ST=NN
L=NN
O=NN
OU=NN
CN=$CLIENT_USERNAME

# Standard OpenSSL keys

# Make a key
KEYFILE=$CLIENT_USERNAME.key
echo "Creating key at $KEYFILE"
openssl genrsa -des3 -passout "pass:$CLIENT_PASSWORD" -out $KEYFILE 2048

# Make a self-signed cert from the key
CERTFILE=$CLIENT_USERNAME.pem
echo "Creating cert at $CERTFILE"
openssl req -key $KEYFILE -passin "pass:$CLIENT_PASSWORD" -new -x509 -days 365 -outform pem -out $CERTFILE

CONFIGFILE=$CLIENT_USERNAME.config
echo "Creating client config at $CONFIGFILE"
cat <<EOF > $CONFIGFILE
metadata.broker.list=localhost:9093
security.protocol=ssl
ssl.certificate.location=$CERTFILE
ssl.key.location=$KEYFILE
ssl.key.password=$CLIENT_PASSWORD
EOF
