package parser

import (
	"bufio"
	"errors"
	"io"
	"net"
	"strconv"

	"gitlab.com/phamhonganh12062000/redis-go/internal/command"
	"gitlab.com/phamhonganh12062000/redis-go/internal/logger"
)

type Parser struct {
	conn net.Conn
	r    *bufio.Reader // Bufffered I/O
	// Used for inline parsing - interpret data as it is being read
	// Without storing the input in memory first
	line   []byte
	pos    int
	logger *logger.Logger
}

func NewParser(conn net.Conn, logger *logger.Logger) *Parser {
	return &Parser{
		conn:   conn,
		r:      bufio.NewReader(conn),
		line:   make([]byte, 0),
		pos:    0,
		logger: logger,
	}
}

func (p *Parser) Command(logger *logger.Logger) (command.Command, error) {
	b, err := p.r.ReadByte()
	if err != nil {
		return command.Command{}, err
	}
	if b == '*' {
		logger.Info("resp array", nil)
		return p.respArray()

	} else {
		line, err := p.readLine()
		if err != nil {
			return command.Command{}, err
		}
		p.pos = 0
		p.line = append([]byte{}, b)
		p.line = append(p.line, line...)
		return p.inline()
	}
}

func (p *Parser) current() byte {
	if p.atEnd() {
		return '\r'
	}
	return p.line[p.pos]
}

func (p *Parser) advance() {
	p.pos++
}

func (p *Parser) atEnd() bool {
	return p.pos >= len(p.line)
}

func (p *Parser) readLine() ([]byte, error) {
	// Read until reaching the delimiter \r
	line, err := p.r.ReadBytes('\r')
	if err != nil {
		return nil, err
	}
	if _, err := p.r.ReadByte(); err != nil {
		return nil, err
	}
	// Remove the carriage return \r
	return line[:len(line)-1], nil
}

// Consume the line one char at a time
// We need to take care of chars like \ and " inside a string
func (p *Parser) consumeString() (s []byte, err error) {
	// Assume that the initial " has been consumed before this method is invoked
	for p.current() != '"' && !p.atEnd() {
		cur := p.current()
		p.advance()
		next := p.current()
		if cur == '\\' && next == '"' {
			s = append(s, '"')
			// Advance the pointer twice
			// Since we need to consume the backlash and the quote
			p.advance()
		} else {
			s = append(s, cur)
		}
	}

	if p.current() != '"' {
		return nil, errors.New("unbalanced quotes in request")
	}
	p.advance()
	return
}

// *3\r\n - Num of elems to be consumed
// $3\r\n
// SET\r\n
// $4\r\n
// name\r\n
// $4\r\n
// John\r\n

// This represents: SET name John
func (p *Parser) respArray() (command.Command, error) {
	cmd := command.Command{}
	elementStr, err := p.readLine()
	if err != nil {
		return cmd, err
	}

	// 1st line contains the number of elems to be consumed
	elems, _ := strconv.Atoi(string(elementStr))

	for range elems {
		tp, err := p.r.ReadByte()
		if err != nil {
			return cmd, err
		}
		switch tp {
		case ':': // Integer case
			// :1000\r\n
			arg, err := p.readLine()
			if err != nil {
				return cmd, err
			}
			cmd.Args = append(cmd.Args, string(arg))
		case '$': // Bulk string case
			// $4\r\n
			// name\r\n
			arg, err := p.readLine()
			if err != nil {
				return cmd, err
			}
			length, _ := strconv.Atoi(string(arg))
			text := make([]byte, length)
			// TODO: This is either dark magic or total bullshit
			// Be sure to test this thoroughly
			_, err = io.ReadFull(p.r, text) // Read exactly length bytes
			if err != nil {
				return cmd, err
			}
			// Discard the \r\n by reading them off to a to-be-discarded buffer
			p.r.Read(make([]byte, 2))
			cmd.Args = append(cmd.Args, string(text))
		case '*':
			// Read the next RESP array recursively
			next, err := p.respArray()
			if err != nil {
				return cmd, err
			}
			cmd.Args = append(cmd.Args, next.Args...)
		}
	}

	return cmd, nil
}

// Parse an inline message
func (p *Parser) inline() (command.Command, error) {
	// In case the user sends a ' GET a'
	for p.current() == ' ' {
		p.advance()
	}

	cmd := command.Command{Conn: p.conn}
	for !p.atEnd() {
		arg, err := p.consumeArg()
		if err != nil {
			return cmd, nil
		}
		if arg != "" {
			cmd.Args = append(cmd.Args, arg)
		}
	}

	return cmd, nil
}

func (p *Parser) consumeArg() (s string, err error) {
	for p.current() == ' ' {
		p.advance()
	}

	// Handle quoted string
	if p.current() == '"' {
		p.advance()
		buf, err := p.consumeString()
		return string(buf), err
	}

	// Append everything to the output until reaching EOL
	for !p.atEnd() && p.current() != ' ' && p.current() != '\r' {
		s += string(p.current())
		p.advance()
	}
	return
}
