package common

import (
	"bufio"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("log")

// ClientConfig Configuration used by the client
type ClientConfig struct {
	ID               string
	ServerAddress    string
	LoopAmount       int
	LoopPeriod       time.Duration
	FilePath         string
	BatchSize        int
	BatchLimitAmount int
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
	if err := c.createClientSocket(ctx); err != nil {
		log.Infof("failed to create client socked")
		return
	}

	defer func() {
		if c.conn != nil {
			c.conn.Close()
		}
	}()

	log.Infof("socket established")

	file, err := os.Open(c.config.FilePath)
	if err != nil {
		log.Infof("failed to open file: %w", err)
		return
	}
	defer file.Close()

	log.Infof("file opened")

	reader := csv.NewReader(file)

	log.Infof("reader created")

	var currentBatch []*Bet
	var currentBetIDs []string
	var batchSizeBytes int

	for {
		select {
		case <-ctx.Done():
			log.Infof("closing client connection due to received signal, client_id: %v", c.config.ID)
			return
		default:
		}

		record, err := reader.Read()
		if err != nil {
			log.Infof("failed trying to read file: %w", err)
			break
		}

		log.Infof("record read: %s", record)

		bet := &Bet{
			ClientID:       c.config.ID,
			FirstName:      record[0],
			LastName:       record[1],
			DocumentNumber: record[2],
			BirthDate:      record[3],
			Number:         record[4],
		}

		betSize, _ := json.Marshal(bet)

		log.Infof("current-batch len: %d, current-size-length: %d", len(currentBatch), batchSizeBytes+len(betSize))
		log.Infof("configurations: batch-size: %d, batch-limit-amount: %d", c.config.BatchSize, c.config.BatchLimitAmount)

		if len(currentBatch) >= c.config.BatchSize || batchSizeBytes+len(betSize) > c.config.BatchLimitAmount {
			err := c.sendMessage("bets", currentBatch)
			if err != nil {
				log.Infof("action: apuestas_enviadas | result: fail | error: %w", err)
			} else {
				log.Infof("action: apuestas_enviadas | result: success | dni: %v | numeros: %v", bet.DocumentNumber, strings.Join(currentBetIDs, "-"))
			}

			currentBatch = []*Bet{}
			currentBetIDs = []string{}
			batchSizeBytes = 0
		}

		log.Infof("adding bet: %v", bet)

		currentBatch = append(currentBatch, bet)
		currentBetIDs = append(currentBetIDs, bet.Number)
		batchSizeBytes += len(betSize)
	}

	if len(currentBatch) > 0 {
		err := c.sendMessage("bets", currentBatch)
		if err != nil {
			log.Infof("action: apuestas_enviadas | result: fail | error: %w", err)
		} else {
			log.Infof("action: apuestas_enviadas | result: success | dni: %v | numeros: %v", currentBatch[0].DocumentNumber, strings.Join(currentBetIDs, "-"))
		}
	}
}

func (c *Client) sendMessage(messageType string, data interface{}) error {
	msg := Message{
		Type: messageType,
		Data: data,
	}

	bytes, err := json.Marshal(msg)
	if err != nil {
		log.Errorf("action: send_message | result: fail | client_id: %v | error: %w",
			c.config.ID,
			err,
		)

		return err
	}

	writer := bufio.NewWriter(c.conn)
	writer.Write(append(bytes, '\n'))

	// Flush avoids short write
	err = writer.Flush()
	if err != nil {
		log.Errorf("action: send_message | result: fail | client_id: %v | error: %w",
			c.config.ID,
			err,
		)

		return err
	}

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

// func (c *Client) GetBets() []*Bet {
// 	// file, err := os.Open()
// }
