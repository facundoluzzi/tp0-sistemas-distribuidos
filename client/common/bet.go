package common

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

const dateFormat = "2006-01-02"

type Bet struct {
	ClientID       string `json:"client_id"`
	FirstName      string `json:"first_name"`
	LastName       string `json:"last_name"`
	DocumentNumber string `json:"document_number"`
	BirthDate      string `json:"birth_date"`
	Number         string `json:"number"`
}

func GetBetFromEnv() (*Bet, error) {
	birthDate := os.Getenv("NACIMIENTO")

	if _, err := time.Parse(dateFormat, birthDate); err != nil {
		return nil, fmt.Errorf("invalid birthdate, must be in format YYYY-MM-DD (actual: %s)", birthDate)
	}

	number := os.Getenv("NUMERO")
	if _, err := strconv.Atoi(number); err != nil {
		return nil, fmt.Errorf("invalid number, must be an integer (actual: %s)", number)
	}

	return &Bet{
		FirstName:      os.Getenv("NOMBRE"),
		LastName:       os.Getenv("APELLIDO"),
		DocumentNumber: os.Getenv("DOCUMENTO"),
		BirthDate:      birthDate,
		Number:         number,
	}, nil
}
