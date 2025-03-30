package common

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
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
		return
	}

	defer func() {
		if c.conn != nil {
			c.conn.Close()
		}
	}()

	file, err := os.Open(c.config.FilePath)
	if err != nil {
		return
	}

	defer file.Close()

	reader := csv.NewReader(file)

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
			if errors.Is(err, io.EOF) {
				break
			}

			log.Infof("failed trying to read file: %w", err)
			break
		}

		bet := &Bet{
			ClientID:       c.config.ID,
			FirstName:      record[0],
			LastName:       record[1],
			DocumentNumber: record[2],
			BirthDate:      record[3],
			Number:         record[4],
		}

		betSize, _ := json.Marshal(bet)

		if len(currentBatch) >= c.config.BatchSize || batchSizeBytes+len(betSize) > c.config.BatchLimitAmount {
			// betNumbersFormatted := strings.Join(currentBetIDs, "-")

			err := c.sendMessage(c.config.ID, "bets", currentBatch)
			if err != nil {
				// log.Infof("action: apuestas_enviadas | result: fail | error: %w", err)
				return
			}

			_, err = c.receiveMessage()
			if err != nil {
				return
			}

			// expectedACK := fmt.Sprintf(BetsACK, betNumbersFormatted)

			// if expectedACK == strings.TrimSpace(response) {
			// log.Infof("action: apuestas_enviadas | result: success | numeros: %v", betNumbersFormatted)
			// }

			currentBatch = []*Bet{}
			currentBetIDs = []string{}
			batchSizeBytes = 0
		}

		currentBatch = append(currentBatch, bet)
		currentBetIDs = append(currentBetIDs, bet.Number)
		batchSizeBytes += len(betSize)
	}

	if len(currentBatch) > 0 && len(currentBatch) != c.config.BatchSize {
		err := c.sendMessage(c.config.ID, "bets", currentBatch)
		if err != nil {
			log.Infof("action: apuestas_enviadas | result: fail | error: %w", err)
		}

		_, err = c.receiveMessage()
		if err != nil {
			return
		}

		// betNumbersFormatted := strings.Join(currentBetIDs, "-")

		// expectedACK := fmt.Sprintf(BetsACK, betNumbersFormatted)

		// if expectedACK == strings.TrimSpace(response) {
		// 	log.Infof("action: apuestas_enviadas | result: success | numeros: %v", betNumbersFormatted)
		// }
	}

	_ = c.sendMessage(c.config.ID, "delivery-ended", nil)
}

func (c *Client) closeConn() {
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
}

func (c *Client) sendMessage(clientID string, messageType string, data []*Bet) error {
	var sb strings.Builder

	if len(data) == 0 {
		sb.WriteString(fmt.Sprintf("%s", clientID))
	} else {
		for _, bet := range data {
			sb.WriteString(fmt.Sprintf(
				"%s|%s|%s|%s|%s|%s\n",
				bet.ClientID, bet.FirstName, bet.LastName, bet.DocumentNumber, bet.BirthDate, bet.Number,
			))
		}
	}

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

	// log.Infof("action: send_message | message_type: %s | result: success", messageType)
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
