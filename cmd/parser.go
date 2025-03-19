package main

import (
	"bufio"
	"errors"
	"io"
	"net"
	"strconv"

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

// TODO: Replace with a struct that receive values in handlers
type Command struct {
	args []string
	conn net.Conn
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

func (p *Parser) command(logger *logger.Logger) (Command, error) {
	b, err := p.r.ReadByte()
	if err != nil {
		return Command{}, err
	}
	if b == '*' {
		logger.Info("resp array", nil)
		return p.respArray()

	} else {
		line, err := p.readLine()
		if err != nil {
			return Command{}, err
		}
		p.pos = 0
		p.line = append([]byte{}, b)
		p.line = append(p.line, line...)
		return p.inline()
	}
}

// *3\r\n - Num of elems to be consumed
// $3\r\n
// SET\r\n
// $4\r\n
// name\r\n
// $4\r\n
// John\r\n

// This represents: SET name John
func (p *Parser) respArray() (Command, error) {
	cmd := Command{}
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
			cmd.args = append(cmd.args, string(arg))
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
			cmd.args = append(cmd.args, string(text))
		case '*':
			// Read the next RESP array recursively
			next, err := p.respArray()
			if err != nil {
				return cmd, err
			}
			cmd.args = append(cmd.args, next.args...)
		}
	}

	return cmd, nil
}

// Parse an inline message
func (p *Parser) inline() (Command, error) {
	// In case the user sends a ' GET a'
	for p.current() == ' ' {
		p.advance()
	}

	cmd := Command{conn: p.conn}
	for !p.atEnd() {
		arg, err := p.consumeArg()
		if err != nil {
			return cmd, nil
		}
		if arg != "" {
			cmd.args = append(cmd.args, arg)
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
