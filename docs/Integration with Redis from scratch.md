# Integration with Redis from scratch

Four common commands: `GET/SET/DEL/QUIT`

```js
GET key
// NX sets the key if it does not exist | XX sets the key if it already exists
SET key value [NX | XX] [EX seconds | PX milliseconds] // Version 6.0.0
// Remove ANY number of keys
DEL key [key ...]
// Close the connection
QUIT
```

> [!NOTE]
>
> - Keywords like `GET` are not case-sensitive, so we can use `get`
> - Key values are case-sensitive, so if we do `set a 1` then `get A` will result in `-1` (nil value)

## Redis serialization protocol (RESP)

The most important addition is **arrays**.

We will use `ERR` as our _generic response_

## Implementation

Our parser will use **inline parsing** - interpreting the data as it is being read

Benefits: **Memory efficiency** (We can process the data at once or with streaming) and **Real-time processing** (Integrate some network protocols into the parser)

We send a command like this

```js
SET text "quoted \"text\" here"
```

## Use with our APIs

1. Create a client package inside `pkg` for shared code in the same directory hierarchy as `cmd`
2. Set up Redis client in the web app
