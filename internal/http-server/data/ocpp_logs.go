package data

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

const DefaultOCPPLogDBPath = "./data/ocpp-tester.db"

type OCPPMessageDirection string

const (
	OCPPMessageDirectionReceived OCPPMessageDirection = "received"
	OCPPMessageDirectionSent     OCPPMessageDirection = "sent"
)

type OCPPLogMessage struct {
	ID            int64                `json:"id"`
	ChargePointID string               `json:"chargePointId"`
	Protocol      string               `json:"protocol"`
	Direction     OCPPMessageDirection `json:"direction"`
	Message       string               `json:"message"`
	CreatedAt     time.Time            `json:"createdAt"`
	MessageTypeID *int                 `json:"messageTypeId,omitempty"`
	MessageType   *string              `json:"messageType,omitempty"`
	UniqueID      *string              `json:"uniqueId,omitempty"`
	Action        *string              `json:"action,omitempty"`
}

type OCPPLogStore struct {
	db *sql.DB
}

func configureOCPPLogDB(db *sql.DB) {
	if db == nil {
		return
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
}

func OpenOCPPLogStore(ctx context.Context, path string) (*OCPPLogStore, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create data directory: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open ocpp log database: %w", err)
	}
	configureOCPPLogDB(db)

	store := NewOCPPLogStore(db)
	if err := store.configure(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := store.Migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return store, nil
}

func NewOCPPLogStore(db *sql.DB) *OCPPLogStore {
	configureOCPPLogDB(db)
	return &OCPPLogStore{db: db}
}

func (s *OCPPLogStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *OCPPLogStore) Migrate(ctx context.Context) error {
	if s == nil || s.db == nil {
		return errors.New("ocpp log store is nil")
	}

	statements := []string{
		`CREATE TABLE IF NOT EXISTS ocpp_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			charge_point_id TEXT NOT NULL,
			protocol TEXT NOT NULL,
			direction TEXT NOT NULL,
			message TEXT NOT NULL,
			created_at TEXT NOT NULL,
			message_type_id INTEGER,
			message_type TEXT,
			unique_id TEXT,
			action TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS idx_ocpp_logs_charge_point_created_at
			ON ocpp_logs (charge_point_id, created_at DESC, id DESC)`,
	}

	for _, statement := range statements {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("migrate ocpp logs: %w", err)
		}
	}

	return nil
}

func (s *OCPPLogStore) configure(ctx context.Context) error {
	statements := []string{
		`PRAGMA busy_timeout = 5000`,
		`PRAGMA journal_mode = WAL`,
		`PRAGMA synchronous = NORMAL`,
	}

	for _, statement := range statements {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("configure ocpp log database: %w", err)
		}
	}

	return nil
}

func (s *OCPPLogStore) RecordOCPPMessage(ctx context.Context, message OCPPLogMessage) error {
	if s == nil || s.db == nil {
		return errors.New("ocpp log store is nil")
	}
	if message.ChargePointID == "" {
		return errors.New("charge point id is required")
	}
	if message.CreatedAt.IsZero() {
		message.CreatedAt = time.Now().UTC()
	}
	message.CreatedAt = message.CreatedAt.UTC()

	parseOCPPMessageMetadata(&message)

	_, err := s.db.ExecContext(ctx, `INSERT INTO ocpp_logs (
			charge_point_id,
			protocol,
			direction,
			message,
			created_at,
			message_type_id,
			message_type,
			unique_id,
			action
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		message.ChargePointID,
		message.Protocol,
		string(message.Direction),
		message.Message,
		message.CreatedAt.Format(time.RFC3339Nano),
		nullableInt(message.MessageTypeID),
		nullableString(message.MessageType),
		nullableString(message.UniqueID),
		nullableString(message.Action),
	)
	if err != nil {
		return fmt.Errorf("record ocpp message: %w", err)
	}

	return nil
}

func (s *OCPPLogStore) ListOCPPMessages(ctx context.Context, chargePointID string, limit int) ([]OCPPLogMessage, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("ocpp log store is nil")
	}
	if chargePointID == "" {
		return nil, errors.New("charge point id is required")
	}

	rows, err := s.db.QueryContext(ctx, `SELECT
			id,
			charge_point_id,
			protocol,
			direction,
			message,
			created_at,
			message_type_id,
			message_type,
			unique_id,
			action
		FROM ocpp_logs
		WHERE charge_point_id = ?
		ORDER BY created_at DESC, id DESC
		LIMIT ?`, chargePointID, limit)
	if err != nil {
		return nil, fmt.Errorf("list ocpp messages: %w", err)
	}
	defer rows.Close()

	var messages []OCPPLogMessage
	for rows.Next() {
		var message OCPPLogMessage
		var direction string
		var createdAt string
		var messageTypeID sql.NullInt64
		var messageType sql.NullString
		var uniqueID sql.NullString
		var action sql.NullString

		if err := rows.Scan(
			&message.ID,
			&message.ChargePointID,
			&message.Protocol,
			&direction,
			&message.Message,
			&createdAt,
			&messageTypeID,
			&messageType,
			&uniqueID,
			&action,
		); err != nil {
			return nil, fmt.Errorf("scan ocpp message: %w", err)
		}

		parsedCreatedAt, err := time.Parse(time.RFC3339Nano, createdAt)
		if err != nil {
			return nil, fmt.Errorf("parse ocpp message created_at: %w", err)
		}

		message.Direction = OCPPMessageDirection(direction)
		message.CreatedAt = parsedCreatedAt
		if messageTypeID.Valid {
			value := int(messageTypeID.Int64)
			message.MessageTypeID = &value
		}
		if messageType.Valid {
			message.MessageType = &messageType.String
		}
		if uniqueID.Valid {
			message.UniqueID = &uniqueID.String
		}
		if action.Valid {
			message.Action = &action.String
		}

		messages = append(messages, message)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate ocpp messages: %w", err)
	}

	return messages, nil
}

func parseOCPPMessageMetadata(message *OCPPLogMessage) {
	var frame []json.RawMessage
	if err := json.Unmarshal([]byte(message.Message), &frame); err != nil {
		return
	}
	if len(frame) < 2 {
		return
	}

	var messageTypeID int
	if err := json.Unmarshal(frame[0], &messageTypeID); err != nil {
		return
	}

	message.MessageTypeID = &messageTypeID
	if messageType, ok := ocppMessageType(messageTypeID); ok {
		message.MessageType = &messageType
	}

	var uniqueID string
	if err := json.Unmarshal(frame[1], &uniqueID); err == nil {
		message.UniqueID = &uniqueID
	}

	if messageTypeID == 2 && len(frame) >= 3 {
		var action string
		if err := json.Unmarshal(frame[2], &action); err == nil {
			message.Action = &action
		}
	}
}

func ocppMessageType(messageTypeID int) (string, bool) {
	switch messageTypeID {
	case 2:
		return "Call", true
	case 3:
		return "CallResult", true
	case 4:
		return "CallError", true
	default:
		return "", false
	}
}

func nullableString(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableInt(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}
