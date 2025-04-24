package main

import (
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"gitlab.com/phamhonganh12062000/redis-go/internal/logger"
	"gitlab.com/phamhonganh12062000/redis-go/internal/session"
	"gitlab.com/phamhonganh12062000/redis-go/internal/store"
)

type Cache struct {
	listener net.Listener
	logger   *logger.Logger
	done     chan os.Signal
	wg       sync.WaitGroup // 	Tracking active connections
	store    *store.InMemoryStore
}

func main() {
	loggerConfig := logger.LoggerConfig{MinLevel: logger.LevelInfo, StackDepth: 3, ShowCaller: true}
	logger := logger.New(os.Stdout, loggerConfig)

	listener, err := net.Listen("tcp", ":6380")
	if err != nil {
		logger.Fatal(err, nil)
		os.Exit(1)
	}

	defer listener.Close()

	logger.Info("Listening on tcp://0.0.0.0:6380", nil)

	store := store.NewInMemoryStore()

	c := &Cache{listener: listener, logger: logger, done: make(chan os.Signal, 1), store: store}

	// Handle signals concurrently while the main thread listen to new connections
	go func() {
		signal.Notify(c.done, syscall.SIGINT, syscall.SIGTERM)
		s := <-c.done // block until a signal is received
		logger.Info("caught signal!", map[string]string{"signal": s.String()})
		os.Exit(0)
	}()

	c.listen()
}

func (c *Cache) listen() {
	for {
		conn, err := c.listener.Accept()
		c.logger.Info("New connection", map[string]string{"connection": conn.LocalAddr().String()})
		if err != nil {
			select {
			case <-c.done:
				return // Shut down the server
			default:
				c.logger.Error(err, nil)
				continue
			}
		}
		c.wg.Add(1)
		// Each new connection needs its own goroutine
		// Alloing multiple clients to be served simultaneously
		go func(conn net.Conn) {
			defer c.wg.Done()
			session.Start(conn, c.logger, c.store)
		}(conn)
	}
}
