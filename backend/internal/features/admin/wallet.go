package admin

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/monero-merchant/monero-merchant/backend/internal/core/models"
	"github.com/monero-merchant/monero-merchant/backend/internal/core/rpc"
)

const walletRPCErrNotOpen = -13

type WalletInfo struct {
	Exists     bool   `json:"exists"`
	Address    string `json:"address,omitempty"`
	WalletName string `json:"wallet_name"`
}

func isWalletNotOpen(err error) bool {
	var rpcErr *rpc.Error
	return errors.As(err, &rpcErr) && rpcErr.Code == walletRPCErrNotOpen
}

// create_wallet returns -21 and restore_deterministic_wallet -1, both say "already exists"
func isWalletExists(err error) bool {
	var rpcErr *rpc.Error
	return errors.As(err, &rpcErr) && strings.Contains(strings.ToLower(rpcErr.Message), "already exists")
}

func (s *AdminService) walletAddress(ctx context.Context) (string, error) {
	var resp struct {
		Address string `json:"address"`
	}
	err := s.rpcClient.Call(ctx, "get_address", map[string]any{"account_index": 0}, &resp)
	return resp.Address, err
}

func (s *AdminService) walletParams() map[string]any {
	return map[string]any{
		"filename": s.config.WalletName,
		"password": s.config.WalletPassword,
		"language": "English",
	}
}

func (s *AdminService) GetWalletInfo(ctx context.Context) (*WalletInfo, *models.HTTPError) {
	info := &WalletInfo{WalletName: s.config.WalletName}

	address, err := s.walletAddress(ctx)
	if isWalletNotOpen(err) {
		// wallet-rpc may have restarted, try opening the configured wallet
		params := map[string]any{"filename": s.config.WalletName, "password": s.config.WalletPassword}
		if s.rpcClient.Call(ctx, "open_wallet", params, nil) != nil {
			return info, nil
		}
		address, err = s.walletAddress(ctx)
	}
	if err != nil {
		return nil, models.NewHTTPError(http.StatusBadGateway, err.Error())
	}

	info.Exists = true
	info.Address = address
	return info, nil
}

// createWallet runs create_wallet or restore_deterministic_wallet, refusing to replace an open
// or existing wallet. Both RPCs would otherwise close the open one.
func (s *AdminService) createWallet(ctx context.Context, method string, params map[string]any) *models.HTTPError {
	_, err := s.walletAddress(ctx)
	if err == nil {
		return models.NewHTTPError(http.StatusConflict, "a wallet is already open")
	}
	if !isWalletNotOpen(err) {
		return models.NewHTTPError(http.StatusBadGateway, err.Error())
	}

	if err := s.rpcClient.Call(ctx, method, params, nil); err != nil {
		if isWalletExists(err) {
			return models.NewHTTPError(http.StatusConflict, "a wallet with this name already exists")
		}
		return models.NewHTTPError(http.StatusBadGateway, err.Error())
	}
	return nil
}

func (s *AdminService) SetupWallet(ctx context.Context) *models.HTTPError {
	if httpErr := s.createWallet(ctx, "create_wallet", s.walletParams()); httpErr != nil {
		return httpErr
	}
	log.Printf("Wallet %q created", s.config.WalletName)
	return nil
}

func (s *AdminService) RestoreWallet(ctx context.Context, seed string, restoreHeight uint64) *models.HTTPError {
	seed = strings.Join(strings.Fields(seed), " ")
	if seed == "" {
		return models.NewHTTPError(http.StatusBadRequest, "seed is required")
	}

	params := s.walletParams()
	params["seed"] = seed
	params["restore_height"] = restoreHeight
	if httpErr := s.createWallet(ctx, "restore_deterministic_wallet", params); httpErr != nil {
		return httpErr
	}
	log.Printf("Wallet %q restored from seed", s.config.WalletName)
	return nil
}

func (s *AdminService) GetWalletSeed(ctx context.Context) (string, *models.HTTPError) {
	var resp struct {
		Key string `json:"key"`
	}
	if err := s.rpcClient.Call(ctx, "query_key", map[string]any{"key_type": "mnemonic"}, &resp); err != nil {
		if isWalletNotOpen(err) {
			return "", models.NewHTTPError(http.StatusNotFound, "no wallet is open")
		}
		return "", models.NewHTTPError(http.StatusBadGateway, err.Error())
	}
	return resp.Key, nil
}
