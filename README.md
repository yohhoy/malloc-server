# malloc-server

malloc REST Server.

## Demo
```sh
$ go build
$ GIN_MODE=release ./malloc-server

$ ADDR=$(curl -X POST -s http://localhost:8080/memory/malloc -d '{"size":10}' | jq .addr)
$ echo $ADDR
$ curl -X PUT -s http://localhost:8080/memory/$ADDR -d '{"val":123}'
$ curl -X GET -s http://localhost:8080/memory/$ADDR | jq .val
$ curl -X POST -s http://localhost:8080/memory/free -d "{\"addr\":$ADDR}"
```

## API
```
POST /memory/malloc
    Req  {"size": <number>}
    Resp {"addr": <number>, "size": <number>}

POST /memory/free
    Req  {"addr": <number>}
    Resp (none)

PUT /memory/:addr
    Req  {"val": <0-255>}
    Resp (none)

GET /memory/:addr
    Req  (none)
    Resp {"val": <0-255>}
```


# License

MIT License
