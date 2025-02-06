package model

import (
	"time"
)

type Account struct {
	Base
	Name      string    `db:"name" json:"name"`
	Email     string    `db:"email" json:"email"`
	Status    string    `db:"status" json:"status"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type Organization struct {
	Base
	AccountID string    `db:"account_id" json:"account_id"`
	Name      string    `db:"name" json:"name"`
	Status    string    `db:"status" json:"status"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}
