# Setup
1. Build the gcb binary
```
go build -ldflags="-w -s" -o ./bin/gcb .
```

2. Run a local instance of docker
```
docker run -d --name gcb-os-docker \
    -p 9200:9200 -p 9600:9600 \
    -e "discovery.type=single-node" \
    -e "OPENSEARCH_INITIAL_ADMIN_PASSWORD={password}" \
    opensearchproject/opensearch:latest
```

3. Populate the initial data
```
./bin/gcb data init -c \
    --os_address "https://localhost:9200" \
    --filepath "{local gencon event csv}" \
    --os_username "admin" \
    --os_password "{password}"
```

4. Run the api server
```
./bin/gcb api \
    --os_address "https://localhost:9200" \
    --os_username "admin" \
    --os_password "{password}"
```