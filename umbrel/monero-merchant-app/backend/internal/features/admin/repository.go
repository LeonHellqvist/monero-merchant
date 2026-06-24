package admin

import (
	"context"

	"github.com/monero-merchant/monero-merchant/backend/internal/core/models"
	"gorm.io/gorm"
)

type AdminRepository interface {
	CreateInvite(ctx context.Context, invite *models.Invite) (*models.Invite, error)
	ListVendorsWithBalances(ctx context.Context) ([]VendorSummary, error)
	ListInvites(ctx context.Context) ([]models.Invite, error)
	ListAllTransactions(ctx context.Context) ([]*models.Transaction, error)
	ListAllPosDevices(ctx context.Context) ([]*models.Pos, error)
	GetVendorByID(ctx context.Context, vendorID uint) (*models.Vendor, error)
}

type adminRepository struct {
	db *gorm.DB
}

func NewAdminRepository(db *gorm.DB) AdminRepository {
	return &adminRepository{db: db}
}

func (r *adminRepository) CreateInvite(ctx context.Context, invite *models.Invite) (*models.Invite, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := r.db.WithContext(ctx).Create(invite).Error; err != nil {
		return nil, err
	}
	return invite, nil
}

func (r *adminRepository) ListVendorsWithBalances(ctx context.Context) ([]VendorSummary, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	var results []VendorSummary
	err := r.db.WithContext(ctx).
		Model(&models.Vendor{}).
		Select("vendors.id AS id, vendors.name AS name, vendors.monero_subaddress AS monero_subaddress, COALESCE(SUM(CASE WHEN transactions.confirmed = ? AND transactions.transferred = ? THEN transactions.amount ELSE 0 END), 0) AS balance", true, false).
		Joins("LEFT JOIN transactions ON transactions.vendor_id = vendors.id").
		Group("vendors.id, vendors.name, vendors.monero_subaddress").
		Order("vendors.id ASC").
		Scan(&results).Error
	if err != nil {
		return nil, err
	}

	return results, nil
}

func (r *adminRepository) ListInvites(ctx context.Context) ([]models.Invite, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	var invites []models.Invite
	err := r.db.WithContext(ctx).
		Order("created_at DESC").
		Find(&invites).Error
	if err != nil {
		return nil, err
	}
	return invites, nil
}

func (r *adminRepository) ListAllTransactions(ctx context.Context) ([]*models.Transaction, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	var transactions []*models.Transaction
	err := r.db.WithContext(ctx).
		Preload("SubTransactions").
		Preload("Pos").
		Preload("Vendor").
		Order("created_at DESC").
		Find(&transactions).Error
	if err != nil {
		return nil, err
	}
	return transactions, nil
}

func (r *adminRepository) ListAllPosDevices(ctx context.Context) ([]*models.Pos, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	var devices []*models.Pos
	err := r.db.WithContext(ctx).
		Preload("Vendor").
		Where("deleted_at IS NULL").
		Order("created_at DESC").
		Find(&devices).Error
	if err != nil {
		return nil, err
	}
	return devices, nil
}

func (r *adminRepository) GetVendorByID(ctx context.Context, vendorID uint) (*models.Vendor, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	var vendor models.Vendor
	if err := r.db.WithContext(ctx).First(&vendor, vendorID).Error; err != nil {
		return nil, err
	}
	return &vendor, nil
}
