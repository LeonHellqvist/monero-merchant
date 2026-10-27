package vendor

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/monero-merchant/monero-merchant/backend/internal/core/models"
)

type registrationRepository struct {
	VendorRepository
	saved *models.Vendor
}

func (r *registrationRepository) VendorByNameExists(context.Context, string) (bool, error) {
	return false, nil
}

func (r *registrationRepository) FindInviteByCode(context.Context, string) (*models.Invite, error) {
	return &models.Invite{ValidUntil: time.Now().Add(time.Hour)}, nil
}

func (r *registrationRepository) CreateVendorWithInvite(_ context.Context, vendor *models.Vendor, _ uint) error {
	r.saved = vendor
	vendor.ID = 1
	return nil
}

func TestRegistrationEmail(t *testing.T) {
	for _, tc := range []struct {
		name, input, want string
		invalid           bool
	}{
		{name: "optional"},
		{name: "normalized", input: " Store+Sales@EXAMPLE.COM ", want: "Store+Sales@example.com"},
		{name: "invalid domain", input: "vendor@-example.com", invalid: true},
		{name: "invalid", input: "not-an-email", invalid: true},
		{name: "oversized", input: strings.Repeat("a", 255), invalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &registrationRepository{}
			service := NewVendorService(repo, nil, nil, nil)
			_, httpErr := service.CreateVendor(context.Background(), "Store", tc.input, "password123", "invite", "81"+strings.Repeat("1", 93))
			if tc.invalid {
				if httpErr == nil || httpErr.Code != 400 || repo.saved != nil {
					t.Fatal("invalid email must be rejected before saving")
				}
				return
			}
			if httpErr != nil {
				t.Fatal(httpErr)
			}
			if repo.saved == nil {
				t.Fatal("vendor was not saved")
			}
			if tc.want == "" {
				if repo.saved.Email != nil {
					t.Fatal("omitted email must be nil")
				}
			} else if repo.saved.Email == nil || *repo.saved.Email != tc.want {
				t.Fatal("normalized email was not saved")
			}
		})
	}
}
