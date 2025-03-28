# malloc-server
[![test CI](https://github.com/yohhoy/malloc-server/actions/workflows/test.yml/badge.svg)](https://github.com/yohhoy/malloc-server/actions/workflows/test.yml)
[![MIT License](http://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

malloc REST Server - A RESTful server that allows you to allocate(`malloc`)/deallocate(`free`) memory blocks, and perform byte-wise read/write operation on memory address via an HTTP interface.

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

Try following client implementation.

- [yohhoy/malloc-client-cpp](https://github.com/yohhoy/malloc-client-cpp) - C++

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

## Security Considerations

The memory management of this server is sandboxed and utilizes "virtual" addresses to represent memory locations. Therefore, clients can NEVER access arbitrary memory on the host machine.

However, note that if a malicious client makes an excessive number of memory allocation requests, the host system's memory may become exhausted.


# License

MIT License
