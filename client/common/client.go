package common

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("log")

// ClientConfig Configuration used by the client
type ClientConfig struct {
	ID            string
	ServerAddress string
	LoopAmount    int
	LoopPeriod    time.Duration
}

// Client Entity that encapsulates how
type Client struct {
	config ClientConfig
	conn   net.Conn
}

// NewClient Initializes a new client receiving the configuration
// as a parameter
func NewClient(config ClientConfig) *Client {
	client := &Client{
		config: config,
	}
	return client
}

// CreateClientSocket Initializes client socket. In case of
// failure, error is printed in stdout/stderr and exit 1
// is returned
func (c *Client) createClientSocket(ctx context.Context) error {
	dialer := net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", c.config.ServerAddress)
	if err != nil {
		log.Criticalf(
			"action: connect | result: fail | client_id: %v | error: %v",
			c.config.ID,
			err,
		)
	}
	c.conn = conn
	return nil
}

// StartClientLoop Send messages to the client until some time threshold is met
func (c *Client) StartClientLoop(ctx context.Context) {
	select {
	case <-ctx.Done():
		log.Infof("closing client connection due to received signal, client_id: %v", c.config.ID)
		return
	default:
	}

	if err := c.createClientSocket(ctx); err != nil {
		return
	}

	defer func() {
		if c.conn != nil {
			c.conn.Close()
		}
	}()

	bet, err := GetBetFromEnv()
	if err != nil {
		log.Errorf("action: building bet message | result: fail | client_id: %v | error: %w",
			c.config.ID,
			err,
		)

		return
	}

	bet.ClientID = c.config.ID

	err = c.sendMessage("bet", bet)
	if err != nil {
		return
	}

	response, err := c.receiveMessage()
	if err != nil {
		return
	}

	expectedACK := fmt.Sprintf(BetACK, bet.ClientID)

	if expectedACK == response {
		log.Infof("action: apuesta_enviada | result: success | dni: %v | numero: %v", bet.DocumentNumber, bet.Number)
	}
}

func (c *Client) sendMessage(messageType string, bet *Bet) error {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf(
		"%s|%s|%s|%s|%s|%s\n",
		bet.ClientID, bet.FirstName, bet.LastName, bet.DocumentNumber, bet.BirthDate, bet.Number,
	))

	message := fmt.Sprintf("%s\n%s", messageType, sb.String())
	messageBytes := []byte(message)

	msgLength := uint16(len(messageBytes))
	lengthBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(lengthBytes, msgLength)

	writer := bufio.NewWriter(c.conn)
	_, err := writer.Write(append(lengthBytes, messageBytes...))
	if err != nil {
		log.Errorf("action: send_message | result: fail | client_id: %v | error: %v",
			c.config.ID, err)
		return err
	}

	// Flush avoids short write
	err = writer.Flush()
	if err != nil {
		log.Errorf("action: send_message | result: fail | client_id: %v | error: %v",
			c.config.ID, err)
		return err
	}

	log.Infof("action: send_message | message_type: %s | result: success", messageType)
	return nil
}

func (c *Client) receiveMessage() (string, error) {
	msg, err := bufio.NewReader(c.conn).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		log.Errorf("action: receive_message | result: fail | client_id: %v | error: %w",
			c.config.ID,
			err,
		)

		return "", err
	}

	return msg, nil
}
