package service

import (
	"context"

	"github.com/johnmaddison/ocpp-tester/internal/http-server/data"
)

const (
	DefaultOCPPLogLimit = 200
	MaxOCPPLogLimit     = 1000
)

type OCPPLogStore interface {
	ListOCPPMessages(ctx context.Context, chargePointID string, limit int) ([]data.OCPPLogMessage, error)
}

type OCPPLogService struct {
	store OCPPLogStore
}

func NewOCPPLogService(store OCPPLogStore) *OCPPLogService {
	return &OCPPLogService{store: store}
}

func (s *OCPPLogService) ListMessages(ctx context.Context, chargePointID string, limit int) ([]data.OCPPLogMessage, error) {
	return s.store.ListOCPPMessages(ctx, chargePointID, limit)
}
