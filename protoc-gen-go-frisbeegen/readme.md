```
go get -u . && \
    protoc --go_out=. --go_opt=paths=source_relative \
    --go-frisbeegen_out=. --go-frisbeegen_opt=paths=source_relative \
    ./exampleproto/pubSub.proto
```