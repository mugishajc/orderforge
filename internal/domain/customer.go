package domain

import (
	"time"

	"github.com/google/uuid"
)

type Customer struct {
	ID        uuid.UUID
	Email     string
	Name      string
	CreatedAt time.Time
}
