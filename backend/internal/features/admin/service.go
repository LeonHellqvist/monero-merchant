package admin

import (
	"context"
	"net/http"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/monero-merchant/monero-merchant/backend/internal/core/config"
	"github.com/monero-merchant/monero-merchant/backend/internal/core/models"
	"github.com/monero-merchant/monero-merchant/backend/internal/core/rpc"
	"github.com/monero-merchant/monero-merchant/backend/internal/core/utils"
	vendorfeature "github.com/monero-merchant/monero-merchant/backend/internal/features/vendor"
)

type AdminService struct {
	repo          AdminRepository
	config        *config.Config
	rpcClient     *rpc.Client
	vendorService *vendorfeature.VendorService
}

type VendorSummary struct {
	ID               uint    `json:"id"`
	Name             string  `json:"name"`
	MoneroSubaddress string  `json:"monero_subaddress"`
	Balance          int64   `json:"balance"`
	Email            *string `json:"email"`
}

func NewAdminService(repo AdminRepository, cfg *config.Config, rpcClient *rpc.Client, vendorService *vendorfeature.VendorService) *AdminService {
	return &AdminService{repo: repo, config: cfg, rpcClient: rpcClient, vendorService: vendorService}
}

func (s *AdminService) CreateInvite(ctx context.Context, validUntil time.Time, forcedName *string) (inviteCode string, err error) {
	if ctx == nil {
		ctx = context.Background()
	}

	inviteCode, err = gonanoid.New()

	if err != nil {
		return "", err
	}

	invite := &models.Invite{
		Used:       false,
		InviteCode: inviteCode,
		ForcedName: forcedName,
		ValidUntil: validUntil,
	}

	_, err = s.repo.CreateInvite(ctx, invite)
	if err != nil {
		return "", err
	}

	return inviteCode, nil
}

func (s *AdminService) ListVendorsWithBalances(ctx context.Context) ([]VendorSummary, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	return s.repo.ListVendorsWithBalances(ctx)
}

func (s *AdminService) DeleteVendor(ctx context.Context, vendorID uint) (httpErr *models.HTTPError) {
	if ctx == nil {
		ctx = context.Background()
	}

	if s.vendorService == nil {
		return models.NewHTTPError(http.StatusInternalServerError, "vendor service not configured")
	}

	if vendorID == 0 {
		return models.NewHTTPError(http.StatusBadRequest, "vendor_id is required")
	}

	return s.vendorService.DeleteVendor(ctx, vendorID)
}

type InviteSummary struct {
	ID         uint    `json:"id"`
	InviteCode string  `json:"invite_code"`
	ForcedName *string `json:"forced_name"`
	Used       bool    `json:"used"`
	ValidUntil string  `json:"valid_until"`
	CreatedAt  string  `json:"created_at"`
}

type TransactionSummary struct {
	ID               uint    `json:"id"`
	VendorID         uint    `json:"vendor_id"`
	VendorName       string  `json:"vendor_name"`
	PosID            uint    `json:"pos_id"`
	PosName          string  `json:"pos_name"`
	Amount           int64   `json:"amount"`
	Currency         string  `json:"currency"`
	AmountInCurrency float64 `json:"amount_in_currency"`
	Description      *string `json:"description"`
	Accepted         bool    `json:"accepted"`
	Confirmed        bool    `json:"confirmed"`
	Transferred      bool    `json:"transferred"`
	TxHash           string  `json:"tx_hash,omitempty"`
	CreatedAt        string  `json:"created_at"`
}

type PosDeviceSummary struct {
	ID         uint   `json:"id"`
	Name       string `json:"name"`
	VendorID   uint   `json:"vendor_id"`
	VendorName string `json:"vendor_name"`
	CreatedAt  string `json:"created_at"`
}

func (s *AdminService) ListInvites(ctx context.Context) ([]InviteSummary, error) {
	invites, err := s.repo.ListInvites(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]InviteSummary, len(invites))
	for i, inv := range invites {
		result[i] = InviteSummary{
			ID:         inv.ID,
			InviteCode: inv.InviteCode,
			ForcedName: inv.ForcedName,
			Used:       inv.Used,
			ValidUntil: inv.ValidUntil.Format(time.RFC3339),
			CreatedAt:  inv.CreatedAt.Format(time.RFC3339),
		}
	}
	return result, nil
}

func (s *AdminService) ListTransactions(ctx context.Context, limit int, offset int) ([]TransactionSummary, int64, error) {
	transactions, total, err := s.repo.ListTransactions(ctx, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	result := make([]TransactionSummary, len(transactions))
	for i, tx := range transactions {
		result[i] = TransactionSummary{
			ID:               tx.ID,
			VendorID:         tx.VendorID,
			VendorName:       tx.Vendor.Name,
			PosID:            tx.PosID,
			PosName:          tx.Pos.Name,
			Amount:           tx.Amount,
			Currency:         tx.Currency,
			AmountInCurrency: tx.AmountInCurrency,
			Description:      tx.Description,
			Accepted:         tx.Accepted,
			Confirmed:        tx.Confirmed,
			Transferred:      tx.Transferred,
			CreatedAt:        tx.CreatedAt.Format(time.RFC3339),
		}
		if len(tx.SubTransactions) > 0 {
			result[i].TxHash = tx.SubTransactions[0].TxHash
		}
	}
	return result, total, nil
}

func (s *AdminService) ListPosDevices(ctx context.Context) ([]PosDeviceSummary, error) {
	devices, err := s.repo.ListPosDevices(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]PosDeviceSummary, len(devices))
	for i, d := range devices {
		result[i] = PosDeviceSummary{
			ID:         d.ID,
			Name:       d.Name,
			VendorID:   d.VendorID,
			VendorName: d.Vendor.Name,
			CreatedAt:  d.CreatedAt.Format(time.RFC3339),
		}
	}
	return result, nil
}

func (s *AdminService) VerifyPassword(password string) bool {
	return utils.SecureCompare(password, s.config.AdminPassword)
}
