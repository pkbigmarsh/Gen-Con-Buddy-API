```
go run . data init -c \
    --os_address "http://localhost:9200" \
    -v debug \
    --filepath "./testdata/changelog_tests/00_10-initial-events.csv"
```

```
go run . data update \
    --os_address "http://localhost:9200" \
    -v debug \
    --local_file "/mnt/c/workspace/gencon_buddy_api/testdata/changelog_tests/01_no-changes.csv"
```

```
docker run -d --name gcb-os-docker \
    -p 9200:9200 -p 9600:9600 \
    -e "discovery.type=single-node" \
    -e "DISABLE_SECURITY_PLUGIN=true" \
    opensearchproject/opensearch:latest
```