package model

import (
	"time"

	"github.com/shopspring/decimal"
)

func (dataEntity *LedgerDataEntity) ToDomain() Ledger {
	return Ledger(*dataEntity)
}

type LedgerDataEntity struct {
	Id              int64           `gorm:"column:id"`
	UserId          int64           `gorm:"column:user_id"`
	TransactionType string          `gorm:"column:transaction_type"`
	Token           string          `gorm:"column:token"`
	Amount          decimal.Decimal `gorm:"column:amount"`
	CreatedAt       time.Time       `gorm:"column:created_at"`
}

func (dataEntity *LedgerDataEntity) TableName() string {
	return "main.ledgers"
}

type Ledger struct {
	Id              int64
	UserId          int64
	TransactionType string
	Token           string
	Amount          decimal.Decimal
	CreatedAt       time.Time
}
