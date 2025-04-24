package session

import (
	"fmt"
	"net"

	"gitlab.com/phamhonganh12062000/redis-go/internal/logger"
	"gitlab.com/phamhonganh12062000/redis-go/internal/parser"
	"gitlab.com/phamhonganh12062000/redis-go/internal/store"
)

// Handle the client's session
// Parse and execute commands
// Then write responses back to the client
func Start(conn net.Conn, logger *logger.Logger, store *store.InMemoryStore) {
	// Ensure the connection will ALWAYS be closed
	defer func() {
		logger.Info("Closing connection", map[string]string{"connection": conn.LocalAddr().String()})
		conn.Close()
	}()

	// At some point we might be reading from a closed connection
	// And we do not want the server to die in case of an error
	defer func() {
		if err := recover(); err != nil {
			logger.Error(fmt.Errorf("Error: %s", err), nil)
		}
	}()

	p := parser.NewParser(conn, logger)

	for {
		cmd, err := p.Command(logger)
		if err != nil {
			logger.Error(fmt.Errorf("Error: %s", err), nil)
			conn.Write([]uint8("-ERR " + err.Error() + "\r\n"))
			break
		}
		// End of a session
		if !cmd.Handle(logger, store) {
			break
		}
	}
}
