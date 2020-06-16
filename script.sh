#!/bin/bash

wait_for_enter() {
    echo -n \$ "$@"
    read
    $@
}

echo "Clean up any docker remnants from previous runs"
wait_for_enter docker-compose rm
wait_for_enter rm -rf './broker_mount/*'

echo "Make TLS certs for the broker side"
mkdir -p ./broker_mount
cd ./broker_mount
wait_for_enter ../gen_server_tls.sh ca ca-cert localhost
echo ""
wait_for_enter ../gen_server_tls.sh server ca-cert broker. localhost
cd ../

echo "Bring up Kafka broker and zookeeper"
wait_for_enter 'docker-compose up &'

echo "Run Go server"
wait_for_enter go run main.go &

echo "Open Go auth server in browser and create a user named 'spencer'"
wait_for_enter firefox http://localhost:8080/

echo "Copy credentials over here"
wait_for_enter cp ~/Downloads/spencer-credentials.zip .

echo "Unzip credentials"
wait_for_enter unzip spencer-credentials.zip

echo "Modify credentials to match broker config"
wait_for_enter 'echo "ssl.ca.location=librdk/ca-cert" >> client-spencer.config'

echo "Try to use the creds"
wait_for_enter hop publish -F client-spencer.config kafka://localhost:9093/test-topic example.gcn3

echo "Load credentials into Kafka"
wait_for_enter ./update_broker.sh

echo "Try again to use the creds"
wait_for_enter hop publish -F client-spencer.config kafka://localhost:9093/test-topic example.gcn3
