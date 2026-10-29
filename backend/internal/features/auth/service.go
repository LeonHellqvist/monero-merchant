package auth

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/monero-merchant/monero-merchant/backend/internal/core/config"
	"github.com/monero-merchant/monero-merchant/backend/internal/core/utils"
	"golang.org/x/crypto/bcrypt"
)

const (
	vendorRefreshTokenTTL = 30 * 24 * time.Hour
	posRefreshTokenTTL    = 90 * 24 * time.Hour
	adminRefreshTokenTTL  = 24 * time.Hour
)

var (
	ErrInvalidCurrentPassword = errors.New("invalid current password")
	ErrInvalidRefreshToken    = errors.New("invalid refresh token")
)

type AuthService struct {
	repo   AuthRepository
	config *config.Config
	// New on every start, so a restart (needed to change ADMIN_PASSWORD) logs out every admin
	adminStamp string
}

func NewAuthService(repo AuthRepository, cfg *config.Config) *AuthService {
	return &AuthService{repo: repo, config: cfg, adminStamp: rand.Text()}
}

/*
	 func (s *AuthService) RegisterDevice(name, password string) error {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		device := &models.Device{
			Name:         name,
			PasswordHash: string(hashedPassword),
		}

		return s.repo.CreateDevice(device)
	}
*/
func (s *AuthService) AuthenticateAdmin(ctx context.Context, name string, password string) (accessToken string, refreshToken string, err error) {
	// Compare both so a wrong name takes as long as a wrong password
	nameMatch := utils.SecureCompare(name, s.config.AdminName)
	passMatch := utils.SecureCompare(password, s.config.AdminPassword)
	if !nameMatch || !passMatch {
		return "", "", errors.New("invalid credentials")
	}

	accessToken, refreshToken, err = s.generateAdminToken()
	if err != nil {
		return "", "", errors.New("failed to generate tokens")
	}

	return accessToken, refreshToken, nil
}

func (s *AuthService) AuthenticateVendor(ctx context.Context, name string, password string) (accessToken string, refreshToken string, vendorID uint, err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	vendor, err := s.repo.FindVendorByName(ctx, name)
	if err != nil {
		return "", "", 0, errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(vendor.PasswordHash), []byte(password)); err != nil {
		return "", "", 0, errors.New("invalid credentials")
	}

	accessToken, refreshToken, err = s.generateVendorToken(vendor.ID, vendor.PasswordVersion)
	if err != nil {
		return "", "", 0, errors.New("failed to generate tokens")
	}

	return accessToken, refreshToken, vendor.ID, nil
}

func (s *AuthService) AuthenticatePos(ctx context.Context, vendorID uint, name string, password string) (accessToken string, refreshToken string, err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	pos, err := s.repo.FindPosByVendorIDAndName(ctx, vendorID, name)
	if err != nil {
		return "", "", errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(pos.PasswordHash), []byte(password)); err != nil {
		return "", "", errors.New("invalid credentials")
	}

	accessToken, refreshToken, err = s.generatePosToken(vendorID, pos.ID, pos.PasswordVersion)
	if err != nil {
		return "", "", errors.New("failed to generate tokens")
	}

	return accessToken, refreshToken, nil
}

func (s *AuthService) UpdateVendorPassword(ctx context.Context, vendorID uint, currentPassword string, newPassword string) (accessToken string, newRefreshToken string, err error) {
	if ctx == nil {
		ctx = context.Background()
	}

	// check if the old password is correct
	vendor, err := s.repo.FindVendorByID(ctx, vendorID)
	if err != nil {
		return "", "", errors.New("vendor not found")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(vendor.PasswordHash), []byte(currentPassword)); err != nil {
		return "", "", ErrInvalidCurrentPassword
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return "", "", err
	}

	passwordVersion, err := s.repo.UpdateVendorPasswordHash(ctx, vendorID, string(hashedPassword))
	if err != nil {
		return "", "", err
	}

	accessToken, newRefreshToken, err = s.generateVendorToken(vendorID, passwordVersion)
	if err != nil {
		return "", "", err
	}

	return accessToken, newRefreshToken, nil
}

func (s *AuthService) UpdatePosPassword(ctx context.Context, posID uint, vendorID uint, currentPassword string, newPassword string) (accessToken string, newRefreshToken string, err error) {
	if ctx == nil {
		ctx = context.Background()
	}

	// check if the old password is correct
	pos, err := s.repo.FindPosByID(ctx, posID)
	if err != nil {
		return "", "", errors.New("pos not found")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(pos.PasswordHash), []byte(currentPassword)); err != nil {
		return "", "", ErrInvalidCurrentPassword
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return "", "", err
	}

	passwordVersion, err := s.repo.UpdatePosPasswordHash(ctx, posID, string(hashedPassword))
	if err != nil {
		return "", "", err
	}

	accessToken, newRefreshToken, err = s.generatePosToken(vendorID, posID, passwordVersion)
	if err != nil {
		return "", "", err
	}

	return accessToken, newRefreshToken, nil
}

func (s *AuthService) UpdatePosPasswordFromVendor(ctx context.Context, posID uint, vendorID uint, newPassword string) (accessToken string, newRefreshToken string, err error) {
	if ctx == nil {
		ctx = context.Background()
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return "", "", err
	}

	passwordVersion, err := s.repo.UpdatePosPasswordHash(ctx, posID, string(hashedPassword))
	if err != nil {
		return "", "", err
	}

	accessToken, newRefreshToken, err = s.generatePosToken(vendorID, posID, passwordVersion)
	if err != nil {
		return "", "", err
	}

	return accessToken, newRefreshToken, nil
}

type refreshClaims struct {
	Role            string `json:"role"`
	VendorID        uint   `json:"vendor_id"`
	PosID           uint   `json:"pos_id"`
	PasswordVersion uint32 `json:"password_version"`
	AdminStamp      string `json:"admin_stamp"`
	jwt.RegisteredClaims
}

func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (accessToken string, newRefreshToken string, err error) {
	claims := &refreshClaims{}
	token, err := jwt.ParseWithClaims(refreshToken, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.config.JWTRefreshSecret), nil
	}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired())

	if err != nil || !token.Valid {
		return "", "", ErrInvalidRefreshToken
	}

	switch claims.Role {
	case "admin":
		if !utils.SecureCompare(claims.AdminStamp, s.adminStamp) {
			return "", "", fmt.Errorf("%w: server restarted", ErrInvalidRefreshToken)
		}
		return s.generateAdminToken()
	case "vendor":
		vendor, err := s.repo.FindVendorByID(ctx, claims.VendorID)
		if errors.Is(err, ErrNotFound) {
			return "", "", fmt.Errorf("%w: vendor not found", ErrInvalidRefreshToken)
		}
		if err != nil {
			return "", "", err
		}
		if vendor.PasswordVersion != claims.PasswordVersion {
			return "", "", fmt.Errorf("%w: password changed", ErrInvalidRefreshToken)
		}
		return s.generateVendorToken(claims.VendorID, claims.PasswordVersion)
	case "pos":
		pos, err := s.repo.FindPosByID(ctx, claims.PosID)
		if errors.Is(err, ErrNotFound) {
			return "", "", fmt.Errorf("%w: pos not found", ErrInvalidRefreshToken)
		}
		if err != nil {
			return "", "", err
		}
		if pos.VendorID != claims.VendorID {
			return "", "", fmt.Errorf("%w: pos belongs to another vendor", ErrInvalidRefreshToken)
		}
		if pos.PasswordVersion != claims.PasswordVersion {
			return "", "", fmt.Errorf("%w: password changed", ErrInvalidRefreshToken)
		}
		return s.generatePosToken(claims.VendorID, claims.PosID, claims.PasswordVersion)
	default:
		return "", "", fmt.Errorf("%w: unknown role", ErrInvalidRefreshToken)
	}
}

func (s *AuthService) generateVendorToken(vendorID uint, passwordVersion uint32) (accessToken string, refreshToken string, err error) {
	accessTokenJWT := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"vendor_id":        vendorID,
		"role":             "vendor",
		"password_version": passwordVersion,
		"exp":              time.Now().Add(time.Minute * 5).Unix(),
	})

	refreshTokenJWT := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"vendor_id":        vendorID,
		"role":             "vendor",
		"password_version": passwordVersion,
		"exp":              time.Now().Add(vendorRefreshTokenTTL).Unix(),
	})

	accessToken, err = accessTokenJWT.SignedString([]byte(s.config.JWTSecret))
	if err != nil {
		return "", "", err
	}

	refreshToken, err = refreshTokenJWT.SignedString([]byte(s.config.JWTRefreshSecret))
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func (s *AuthService) generatePosToken(vendorID uint, posID uint, passwordVersion uint32) (accessToken string, refreshToken string, err error) {
	accessTokenJWT := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"vendor_id":        vendorID,
		"role":             "pos",
		"password_version": passwordVersion,
		"pos_id":           posID,
		"exp":              time.Now().Add(time.Minute * 5).Unix(),
	})

	refreshTokenJWT := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"vendor_id":        vendorID,
		"role":             "pos",
		"password_version": passwordVersion,
		"pos_id":           posID,
		"exp":              time.Now().Add(posRefreshTokenTTL).Unix(),
	})

	accessToken, err = accessTokenJWT.SignedString([]byte(s.config.JWTSecret))
	if err != nil {
		return "", "", err
	}

	refreshToken, err = refreshTokenJWT.SignedString([]byte(s.config.JWTRefreshSecret))
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func (s *AuthService) generateAdminToken() (accessToken string, refreshToken string, err error) {
	accessTokenJWT := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"vendor_id":        0,
		"role":             "admin",
		"password_version": 0,
		"exp":              time.Now().Add(time.Minute * 5).Unix(),
	})

	refreshTokenJWT := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"vendor_id":        0,
		"role":             "admin",
		"password_version": 0,
		"admin_stamp":      s.adminStamp,
		"exp":              time.Now().Add(adminRefreshTokenTTL).Unix(),
	})

	accessToken, err = accessTokenJWT.SignedString([]byte(s.config.JWTSecret))
	if err != nil {
		return "", "", err
	}

	refreshToken, err = refreshTokenJWT.SignedString([]byte(s.config.JWTRefreshSecret))
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}
