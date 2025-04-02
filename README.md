# redis-go

Redis built in Golang. Intend to use this for my web apps.

## TODOs

- [x] In-memory key-value storage
- [>] Support for basic Redis commands (SET, GET, PING, ECHO)
- [x] Key expiration with millisecond precision
- [ ] Make cache sync.Map part of a inMemoryStore struct instead of a global variable
- [ ] Leader-Follower replication
- [>] Client
- [~] RESP (Redis Serialization Protocol) implementation
- [ ] RDB Persistence: Save and load the database to and from an RDB file for data persistence
- [ ] Logger v2

## Refs

1. [Simple Redis clone](https://github.com/therahulbhati/go-redis-clonea)
2. [Logger from Ardan's lab](https://github.com/ardanlabs/usdl)
3. [Miniredis](https://github.com/alicebob/miniredis)
