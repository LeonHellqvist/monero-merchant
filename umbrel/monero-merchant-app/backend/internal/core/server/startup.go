package server

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
)

const atomicUnitsPerXMR uint64 = 1_000_000_000_000

func (s *Server) runStartupSequence(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}

	s.logMoneroNodeInfo(ctx)
	s.ensureWalletReady(ctx)
	s.logMoneroPayHealth(ctx)
}

func (s *Server) logMoneroNodeInfo(parentCtx context.Context) {
	if s.daemonRPC == nil {
		log.Println("Monero daemon RPC client not configured; skipping node info logging")
		return
	}

	ctx, cancel := context.WithTimeout(parentCtx, 10*time.Second)
	defer cancel()

	var result struct {
		Height              uint64 `json:"height"`
		Nettype             string `json:"nettype"`
		Synchronized        bool   `json:"synchronized"`
		IncomingConnections uint64 `json:"incoming_connections_count"`
		OutgoingConnections uint64 `json:"outgoing_connections_count"`
	}

	if err := s.daemonRPC.Call(ctx, "get_info", nil, &result); err != nil {
		log.Printf("Failed to query Monero node info: %v", err)
		return
	}

	log.Printf(
		"Monero node: height=%d nettype=%s synchronized=%t incoming_connections=%d outgoing_connections=%d",
		result.Height,
		result.Nettype,
		result.Synchronized,
		result.IncomingConnections,
		result.OutgoingConnections,
	)
}

func (s *Server) ensureWalletReady(parentCtx context.Context) {
	if s.walletRPC == nil {
		log.Println("Monero wallet RPC client not configured; skipping wallet checks")
		return
	}

	// Retry with backoff — wallet-rpc may not be ready immediately on startup
	var locked, unlocked uint64
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			log.Printf("Retrying wallet open (attempt %d/3) after 5s...", attempt+1)
			time.Sleep(5 * time.Second)
		}
		locked, unlocked, err = s.fetchWalletBalances(parentCtx)
		if err != nil {
			if isWalletNotOpenErr(err) {
				if openErr := s.openWallet(parentCtx); openErr != nil {
					log.Printf("Failed to open wallet %q: %v", s.config.WalletName, openErr)
					continue
				}
				locked, unlocked, err = s.fetchWalletBalances(parentCtx)
			}
		}
		if err == nil {
			break
		}
		log.Printf("Wallet balance fetch error (attempt %d/3): %v", attempt+1, err)
	}

	if err != nil {
		log.Printf("Failed to fetch wallet balances after retries: %v", err)
		return
	}

	total := locked + unlocked
	log.Printf(
		"Wallet balance: total=%s XMR unlocked=%s XMR locked=%s XMR",
		formatAtomic(total),
		formatAtomic(unlocked),
		formatAtomic(locked),
	)

	if err := s.configureAutoRefresh(parentCtx); err != nil {
		log.Printf("Failed to configure wallet auto refresh: %v", err)
	}
}

func (s *Server) fetchWalletBalances(parentCtx context.Context) (uint64, uint64, error) {
	ctx, cancel := context.WithTimeout(parentCtx, 10*time.Second)
	defer cancel()

	var resp struct {
		Balance         uint64 `json:"balance"`
		UnlockedBalance uint64 `json:"unlocked_balance"`
	}

	params := map[string]any{"account_index": 0}
	if err := s.walletRPC.Call(ctx, "get_balance", params, &resp); err != nil {
		return 0, 0, err
	}

	locked := resp.Balance
	if resp.UnlockedBalance <= resp.Balance {
		locked = resp.Balance - resp.UnlockedBalance
	}

	return locked, resp.UnlockedBalance, nil
}

func (s *Server) openWallet(parentCtx context.Context) error {
	if s.config.WalletName == "" {
		return fmt.Errorf("wallet name not configured")
	}

	ctx, cancel := context.WithTimeout(parentCtx, 10*time.Second)
	defer cancel()

	type openParams struct {
		Filename string `json:"filename"`
		Password string `json:"password,omitempty"`
	}

	req := openParams{Filename: s.config.WalletName}
	if s.config.WalletPassword != "" {
		req.Password = s.config.WalletPassword
	}

	if err := s.walletRPC.Call(ctx, "open_wallet", req, nil); err != nil {
		// If wallet doesn't exist, try creating it
		if isWalletNotOpenErr(err) {
			createCtx, createCancel := context.WithTimeout(parentCtx, 15*time.Second)
			defer createCancel()

			type createParams struct {
				Filename string `json:"filename"`
				Password string `json:"password,omitempty"`
				Language string `json:"language"`
			}
			createReq := createParams{
				Filename: s.config.WalletName,
				Language: "English",
			}
			if s.config.WalletPassword != "" {
				createReq.Password = s.config.WalletPassword
			}

			if createErr := s.walletRPC.Call(createCtx, "create_wallet", createReq, nil); createErr != nil {
				return fmt.Errorf("failed to create wallet: %w", createErr)
			}
			log.Printf("Wallet %q created successfully", s.config.WalletName)
			// Try opening again
			openCtx, openCancel := context.WithTimeout(parentCtx, 10*time.Second)
			defer openCancel()
			if openErr := s.walletRPC.Call(openCtx, "open_wallet", req, nil); openErr != nil {
				return fmt.Errorf("failed to open newly created wallet: %w", openErr)
			}
		} else {
			return err
		}
	}

	log.Printf("Wallet %q opened successfully", s.config.WalletName)
	return nil
}

func (s *Server) configureAutoRefresh(parentCtx context.Context) error {
	ctx, cancel := context.WithTimeout(parentCtx, 5*time.Second)
	defer cancel()

	type autoRefreshParams struct {
		Enable bool    `json:"enable"`
		Period *uint32 `json:"period,omitempty"`
	}

	params := autoRefreshParams{Enable: true}
	if s.config.WalletAutoRefreshPeriod > 0 {
		period := s.config.WalletAutoRefreshPeriod
		params.Period = &period
	}

	if err := s.walletRPC.Call(ctx, "auto_refresh", params, nil); err != nil {
		return err
	}

	if params.Period != nil {
		log.Printf("Wallet auto refresh enabled (period=%d seconds)", *params.Period)
	} else {
		log.Printf("Wallet auto refresh enabled using daemon default period")
	}

	return nil
}

func (s *Server) logMoneroPayHealth(parentCtx context.Context) {
	if s.moneroPay == nil {
		log.Println("MoneroPay client not configured; skipping health check")
		return
	}

	ctx, cancel := context.WithTimeout(parentCtx, 5*time.Second)
	defer cancel()

	health, err := s.moneroPay.GetHealth(ctx)
	if err != nil {
		log.Printf("Failed to fetch MoneroPay health: %v", err)
		return
	}

	log.Printf(
		"MoneroPay health: status=%d wallet_rpc=%t postgresql=%t",
		health.Status,
		health.Services.Walletrpc,
		health.Services.Postgresql,
	)
}

func formatAtomic(amount uint64) string {
	const decimalPlaces = 6
	integer := amount / atomicUnitsPerXMR
	remainder := amount % atomicUnitsPerXMR

	decimalDivisor := atomicUnitsPerXMR
	for i := 0; i < decimalPlaces; i++ {
		decimalDivisor /= 10
	}

	roundingOffset := decimalDivisor / 2

	decimals := (remainder + roundingOffset) / decimalDivisor
	maxDecimals := uint64(1)
	for i := 0; i < decimalPlaces; i++ {
		maxDecimals *= 10
	}

	if decimals == maxDecimals {
		integer++
		decimals = 0
	}

	return fmt.Sprintf("%d.%0*d", integer, decimalPlaces, decimals)
}

func isWalletNotOpenErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no wallet") ||
		strings.Contains(msg, "wallet does not exist") ||
		strings.Contains(msg, "wallet open") ||
		strings.Contains(msg, "failed to open wallet") ||
		strings.Contains(msg, "file not found") ||
		strings.Contains(msg, "not available") ||
		strings.Contains(msg, "cannot be opened") ||
		strings.Contains(msg, "doesn't exist")
}
