package main

import (
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"gitlab.com/phamhonganh12062000/redis-go/internal/logger"
)

var cache sync.Map // Store and retrieve values here

type Cache struct {
	listener net.Listener
	logger   *logger.Logger
	done     chan os.Signal
	wg       sync.WaitGroup // 	Tracking active connections
}

func main() {
	listener, err := net.Listen("tcp", ":6380")

	loggerConfig := logger.LoggerConfig{MinLevel: logger.LevelInfo, StackDepth: 3, ShowCaller: true}
	logger := logger.New(os.Stdout, loggerConfig)

	if err != nil {
		logger.Fatal(err, nil)
		os.Exit(1)
	}

	defer listener.Close()

	logger.Info("Listening on tcp://0.0.0.0:6380", nil)

	cache := &Cache{listener: listener, logger: logger, done: make(chan os.Signal, 1)}

	// Handle signals concurrently while the main thread listen to new connections
	go func() {
		signal.Notify(cache.done, syscall.SIGINT, syscall.SIGTERM)
		s := <-cache.done // block until a signal is received
		logger.Info("caught signal!", map[string]string{"signal": s.String()})
		os.Exit(0)
	}()

	cache.listen()
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
			startSession(conn, c.logger)
		}(conn)
	}
}
