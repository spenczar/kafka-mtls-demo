Demonstration of how you can do mutual TLS in Kafka with certs generated in a
web application.

```bash
echo "Clean up any docker remnants from previous runs"
docker-compose rm
rm -rf './broker_mount/*'

echo "Make TLS certs for the broker side"
mkdir -p ./broker_mount
cd ./broker_mount
../gen_server_tls.sh ca ca-cert localhost
echo ""
../gen_server_tls.sh server ca-cert broker. localhost
cd ../

echo "Bring up Kafka broker and zookeeper"
'docker-compose up &'

echo "Run Go server"
go run main.go &

echo "Open Go auth server in browser and create a user named 'spencer'"
firefox http://localhost:8080/

echo "Copy credentials over here"
cp ~/Downloads/spencer-credentials.zip .

echo "Unzip credentials"
unzip spencer-credentials.zip

echo "Modify credentials to match broker config"
'echo "ssl.ca.location=librdk/ca-cert" >> client-spencer.config'

echo "Try to use the creds"
hop publish -F client-spencer.config kafka://localhost:9093/test-topic example.gcn3

echo "Load credentials into Kafka"
./update_broker.sh

echo "Try again to use the creds"
hop publish -F client-spencer.config kafka://localhost:9093/test-topic example.gcn3
```
