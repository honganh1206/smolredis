package command

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"gitlab.com/phamhonganh12062000/redis-go/internal/logger"
	"gitlab.com/phamhonganh12062000/redis-go/internal/store"
)

// TODO: Replace with a struct that receive values in handlers
type Command struct {
	Args []string
	Conn net.Conn
}

const (
	GET  = "GET"
	SET  = "SET"
	DEL  = "DEL"
	QUIT = "QUIT"
	PING = "PING"
	ECHO = "ECHO"
	NX   = "NX"
	XX   = "PX"
	EX   = "EX"
	PX   = "PX"
)

func (cmd Command) Handle(logger *logger.Logger, store *store.InMemoryStore) bool {
	switch strings.ToUpper(cmd.Args[0]) {
	case GET:
		return cmd.get(logger, store)
	case SET:
		return cmd.set(logger, store)
	case DEL:
		return cmd.del(store)
	case QUIT:
		return cmd.quit(logger)
	case PING:
		return cmd.ping(logger)
	case ECHO:
		return cmd.echo(logger)
	default:
		logger.Info("Command not supported", map[string]string{"command": cmd.Args[0]})
		cmd.Conn.Write([]uint8("-ERR unknown command '" + cmd.Args[0] + "'\r\n"))
	}
	return true
}

func (cmd *Command) quit(logger *logger.Logger) bool {
	if len(cmd.Args) != 1 {
		cmd.Conn.Write([]uint8("-ERR wrong number of arguments for '" + cmd.Args[0] + "' command\r\n"))
		return true
	}
	logger.Info("Handle QUIT", nil)
	cmd.Conn.Write([]uint8("+OK\r\n"))
	return false
}

func (cmd *Command) del(store *store.InMemoryStore) bool {
	count := 0
	for _, key := range cmd.Args[1:] {
		if _, ok := store.Data.LoadAndDelete(key); ok {
			count++
		}
	}
	// Write back to the client the number of keys deleted
	cmd.Conn.Write(fmt.Appendf(nil, ":%d\r\n", count))
	return true
}

func (cmd *Command) get(logger *logger.Logger, store *store.InMemoryStore) bool {
	if len(cmd.Args) != 2 {
		cmd.Conn.Write([]uint8("-ERR wrong number of arguments for '" + cmd.Args[0] + "' command\r\n"))
		return true
	}
	logger.Info("Handle GET", nil)
	val, _ := store.Data.Load(cmd.Args[1])
	if val != nil {
		res, _ := val.(string)
		if strings.HasPrefix(res, "\"") {
			res, _ = strconv.Unquote(res)
		}
		logger.Info("Response length", map[string]string{"length": strconv.Itoa(len(res))})
		cmd.Conn.Write(fmt.Appendf(nil, "$%d\r\n", len(res)))
		cmd.Conn.Write(append([]uint8(res), []uint8("\r\n")...)) // Write the key-value
	} else {
		cmd.Conn.Write([]uint8("$-1\r\n"))
	}
	return true
}

func (cmd *Command) set(logger *logger.Logger, store *store.InMemoryStore) bool {
	if len(cmd.Args) < 3 || len(cmd.Args) > 6 {
		cmd.Conn.Write([]uint8("-ERR wrong number of arguments for '" + cmd.Args[0] + "' command\r\n"))
		return true
	}
	logger.Info("Handle SET", nil)
	logger.Info("Value length", map[string]string{"length": strconv.Itoa(len(cmd.Args[2]))})
	if len(cmd.Args) > 3 {
		pos := 3
		option := strings.ToUpper(cmd.Args[pos])
		switch option {
		// Set the key if it does not exist before
		case NX:
			logger.Info("Handle NX", nil)
			if _, ok := store.Data.Load(cmd.Args[1]); ok {
				cmd.Conn.Write([]uint8("$-1\r\n"))
				return true
			}
			pos++
		// Only set the key if it it already exists
		case XX:
			logger.Info("Handle NX", nil)
			if _, ok := store.Data.Load(cmd.Args[1]); !ok {
				cmd.Conn.Write([]uint8("$-1\r\n"))
				return true
			}
			pos++
		}

		// Parse the expiration flag
		if len(cmd.Args) > pos {
			if err := cmd.setExpiration(pos, logger, store); err != nil {
				cmd.Conn.Write([]uint8("-ERR " + err.Error() + "\r\n"))
				return true
			}
		}

	}

	store.Data.Store(cmd.Args[1], cmd.Args[2])
	cmd.Conn.Write([]uint8("+OK\r\n"))
	return true
}

func (cmd *Command) ping(logger *logger.Logger) bool {
	if len(cmd.Args) != 1 {
		cmd.Conn.Write([]uint8("-ERR wrong number of arguments for '" + cmd.Args[0] + "' command\r\n"))
		return true
	}
	logger.Info("Handle PING", nil)
	cmd.Conn.Write([]uint8("$PONG\r\n"))
	return true
}

func (cmd *Command) echo(logger *logger.Logger) bool {
	if len(cmd.Args) != 2 {
		cmd.Conn.Write([]uint8("-ERR wrong number of arguments for '" + cmd.Args[0] + "' command\r\n"))
		return true
	}

	logger.Info("Handle ECHO", nil)
	cmd.Conn.Write(append([]uint8(cmd.Args[1]), []uint8("\r\n")...))
	return true
}

func (cmd *Command) setExpiration(pos int, logger *logger.Logger, store *store.InMemoryStore) error {
	option := strings.ToUpper(cmd.Args[pos])
	value, _ := strconv.Atoi(cmd.Args[pos+1])
	var duration time.Duration

	switch option {
	case EX:
		duration = time.Second * time.Duration(value)
	case PX:
		duration = time.Millisecond * time.Duration(value)
	default:
		return fmt.Errorf("expiration option not valid")
	}

	// Wait by sleeping then delete the key-value from the store.Data
	go func() {
		logger.Info("Handling expirations", map[string]string{"option": option, "duration": shortDur(duration)})
		time.Sleep(duration)
		store.Data.Delete(cmd.Args[1])
	}()
	return nil
}

func shortDur(d time.Duration) string {
	s := d.String()
	if strings.HasSuffix(s, "m0s") || strings.HasSuffix(s, "h0m") {
		s = s[:len(s)-2]
	}
	return s
}
