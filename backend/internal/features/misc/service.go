package misc

import (
	"context"
	"log"

	"github.com/monero-merchant/monero-merchant/backend/internal/core/config"
	"github.com/monero-merchant/monero-merchant/backend/internal/core/rpc"
	"github.com/monero-merchant/monero-merchant/backend/internal/thirdparty/moneropay"
)

type MiscService struct {
	repo      MiscRepository
	config    *config.Config
	moneroPay *moneropay.MoneroPayAPIClient
	daemonRPC *rpc.Client
}

func NewMiscService(repo MiscRepository, cfg *config.Config, moneroPay *moneropay.MoneroPayAPIClient, daemonRPC *rpc.Client) *MiscService {
	return &MiscService{repo: repo, config: cfg, moneroPay: moneroPay, daemonRPC: daemonRPC}
}

// Check if the vendor and POS are authorized for the transaction
func (s *MiscService) GetHealth(ctx context.Context) HealthResponse {
	h := HealthResponse{}

	// Check MoneroPay service health
	mp, mpErr := s.moneroPay.GetHealth(ctx)
	if mpErr != nil {
		h.Services.MoneroPay.Status = 503
	} else {
		h.Services.MoneroPay = *mp
	}

	// Check PostgreSQL service health
	postgresqlStatus, pgErr := s.repo.GetPostgresqlHealth(ctx)
	h.Services.Postgresql = postgresqlStatus && pgErr == nil

	// Set overall status
	if h.Services.Postgresql && h.Services.MoneroPay.Status == 200 {
		h.Status = 200
	} else {
		h.Status = 503
	}

	return h
}

// GetDaemonStatus reports the sync state of the configured Monero daemon
func (s *MiscService) GetDaemonStatus(ctx context.Context) DaemonStatusResponse {
	resp := DaemonStatusResponse{Status: 503}
	if s.daemonRPC == nil {
		return resp
	}

	var info struct {
		Height              uint64 `json:"height"`
		TargetHeight        uint64 `json:"target_height"`
		Synchronized        bool   `json:"synchronized"`
		Nettype             string `json:"nettype"`
		IncomingConnections uint64 `json:"incoming_connections_count"`
		OutgoingConnections uint64 `json:"outgoing_connections_count"`
	}
	if err := s.daemonRPC.Call(ctx, "get_info", nil, &info); err != nil {
		log.Printf("Failed to query Monero daemon status: %v", err)
		return resp
	}

	resp.Status = 200
	resp.Synced = info.Synchronized
	resp.Height = info.Height
	resp.TargetHeight = info.TargetHeight
	resp.Nettype = info.Nettype
	resp.IncomingConnections = info.IncomingConnections
	resp.OutgoingConnections = info.OutgoingConnections
	return resp
}
