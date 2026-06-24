package vendor

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/monero-merchant/monero-merchant/backend/internal/core/config"
	"github.com/monero-merchant/monero-merchant/backend/internal/core/models"
	"github.com/monero-merchant/monero-merchant/backend/internal/core/rpc"
	"github.com/monero-merchant/monero-merchant/backend/internal/thirdparty/moneropay"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type VendorService struct {
	repo      VendorRepository
	db        *gorm.DB
	config    *config.Config
	rpcClient *rpc.Client
	moneroPay *moneropay.MoneroPayAPIClient
	mu        sync.Mutex
}

type WalletBalance struct {
	Total    uint64 `json:"total"`
	Unlocked uint64 `json:"unlocked"`
	Locked   uint64 `json:"locked"`
}

func NewVendorService(repo VendorRepository, db *gorm.DB, cfg *config.Config, rpcClient *rpc.Client, moneroPay *moneropay.MoneroPayAPIClient) *VendorService {
	return &VendorService{repo: repo, db: db, config: cfg, rpcClient: rpcClient, moneroPay: moneroPay}
}

const moneroSubaddressPattern = "^8[0-9AB][1-9A-HJ-NP-Za-km-z]{93}$"

var moneroSubaddressRegex = regexp.MustCompile(moneroSubaddressPattern)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

func isValidEmail(email string) bool {
	return emailRegex.MatchString(email)
}

func (s *VendorService) StartTransferCompleter(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				// bound each sweep to avoid piling up
				sweepCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
				s.completeTransfers(sweepCtx)
				cancel()
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (s *VendorService) completeTransfers(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// We use transfer intead of transfer_split because we need support for subtract_fee_from_outputs
	// We need that because we do not want the server operator to be responsible for covering transaction fees
	// When transfer_split is supported, we can switch to it for a much more efficient transfer process

	// For loop to try and complete transfers
	for i := 15; i > 0; i-- {
		dbTx := s.db.Begin()
		var batchErr error
		func() {
			defer func() {
				if r := recover(); r != nil {
					_ = dbTx.Rollback()
				}
			}()
			// fetch a safe number of transfers to complete
			transfers, err := s.repo.GetTransfersToComplete(ctx, i)
			if err != nil || transfers == nil {
				log.Println("Error fetching transfers to complete:", err)
				batchErr = err
				_ = dbTx.Rollback()
				return
			}

			if len(transfers) == 0 {
				_ = dbTx.Rollback()
				return
			}

			// Mark transactions as transferred
			for _, transfer := range transfers {
				transactionIDs := []uint{}
				for _, tx := range transfer.Transactions {
					transactionIDs = append(transactionIDs, tx.ID)
				}
				if err := s.repo.MarkTransactionsTransferred(ctx, dbTx, transfer.ID, transactionIDs); err != nil {
					log.Printf("Error marking transactions as transferred: %v", err)
					batchErr = err
					_ = dbTx.Rollback()
					return
				}
			}

			destinations := make([]moneropay.Destination, len(transfers))
			for i, transfer := range transfers {
				destinations[i] = moneropay.Destination{
					Amount:  transfer.Amount,
					Address: transfer.Address,
				}
			}

			txHash, amounts, err := s.executeTransfer(ctx, destinations)
			if err != nil {
				log.Printf("Transfer execution failed: %v", err)
				batchErr = err
				_ = dbTx.Rollback()
				return
			}
			if txHash == "" {
				log.Print("Transfer failed, no transaction hash returned")
				batchErr = fmt.Errorf("transfer failed, empty tx hash")
				_ = dbTx.Rollback()
				return
			}
			// We need to mark the transfer as completed

			for index, transfer := range transfers {
				amountTransferred := transfer.Amount
				if len(amounts) > index && amounts[index] != 0 {
					amountTransferred = amounts[index]
				}
				if err := s.repo.MarkTransferCompleted(ctx, dbTx, transfer.ID, amountTransferred, txHash); err != nil {
					log.Println("Error marking transfer as completed:", err)
					batchErr = err
					_ = dbTx.Rollback()
					return
				}
			}
			if err := dbTx.Commit().Error; err != nil {
				log.Println("Error committing transaction:", err)
				batchErr = err
				return
			}
			log.Println("Transfer completed successfully")
			return
		}() // end per-iteration scope
		if batchErr != nil {
			break
		}
	}

}

func (s *VendorService) executeTransfer(ctx context.Context, destinations []moneropay.Destination) (string, []int64, error) {
	if len(destinations) == 0 {
		return "", nil, fmt.Errorf("no destinations provided")
	}

	var rpcErr error
	if s.rpcClient != nil {
		if txHash, amounts, err := s.transferWithWalletRPC(ctx, destinations); err == nil {
			return txHash, amounts, nil
		} else {
			rpcErr = err
			log.Printf("Wallet RPC transfer failed, attempting MoneroPay transfer: %v", err)
		}
	}

	if s.moneroPay != nil {
		txHash, amounts, err := s.transferWithMoneroPay(ctx, destinations)
		if err == nil {
			return txHash, amounts, nil
		}
		if rpcErr != nil {
			return "", nil, fmt.Errorf("wallet RPC transfer failed (%v) and MoneroPay transfer failed (%w)", rpcErr, err)
		}
		return "", nil, err
	}

	if rpcErr != nil {
		return "", nil, fmt.Errorf("wallet RPC transfer failed (%v) and no MoneroPay client configured", rpcErr)
	}

	return "", nil, fmt.Errorf("no transfer backend configured")
}

func (s *VendorService) transferWithWalletRPC(ctx context.Context, destinations []moneropay.Destination) (string, []int64, error) {
	if s.rpcClient == nil {
		return "", nil, fmt.Errorf("wallet RPC client not configured")
	}

	type rpcDestination struct {
		Amount  int64  `json:"amount"`
		Address string `json:"address"`
	}

	type transferParams struct {
		Destinations           []rpcDestination `json:"destinations"`
		SubtractFeeFromOutputs []uint           `json:"subtract_fee_from_outputs,omitempty"`
		DoNotRelay             bool             `json:"do_not_relay,omitempty"`
		Priority               uint             `json:"priority,omitempty"`
	}

	params := transferParams{
		Destinations:           make([]rpcDestination, len(destinations)),
		SubtractFeeFromOutputs: make([]uint, 0, len(destinations)),
		DoNotRelay:             true,
		Priority:               0,
	}

	for i, dest := range destinations {
		params.Destinations[i] = rpcDestination{
			Amount:  dest.Amount,
			Address: dest.Address,
		}
		params.SubtractFeeFromOutputs = append(params.SubtractFeeFromOutputs, uint(i))
	}

	type transferResult struct {
		Amount        int64 `json:"amount"`
		AmountsByDest struct {
			Amounts []int64 `json:"amounts"`
		} `json:"amounts_by_dest"`
		Fee            int64  `json:"fee"`
		MultisigTxset  string `json:"multisig_txset"`
		SpentKeyImages struct {
			KeyImages []string `json:"key_images"`
		} `json:"spent_key_images"`
		TxBlob        string `json:"tx_blob"`
		TxHash        string `json:"tx_hash"`
		TxKey         string `json:"tx_key"`
		TxMetadata    string `json:"tx_metadata"`
		UnsignedTxset string `json:"unsigned_txset"`
		Weight        int64  `json:"weight"`
	}

	// dry-run to ensure the transfer fits in a single transaction
	var dryRun transferResult
	callCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	err := s.rpcClient.Call(callCtx, "transfer", params, &dryRun)
	cancel()
	if err != nil {
		return "", nil, err
	}

	params.DoNotRelay = false

	var result transferResult
	callCtx2, cancel2 := context.WithTimeout(ctx, 15*time.Second)
	err = s.rpcClient.Call(callCtx2, "transfer", params, &result)
	cancel2()
	if err != nil {
		return "", nil, err
	}

	if result.TxHash == "" {
		return "", nil, fmt.Errorf("wallet RPC transfer returned empty tx hash")
	}

	amounts := result.AmountsByDest.Amounts
	if len(amounts) == 0 {
		amounts = make([]int64, len(destinations))
		for i, dest := range destinations {
			amounts[i] = dest.Amount
		}
	}

	return result.TxHash, amounts, nil
}

func (s *VendorService) transferWithMoneroPay(ctx context.Context, destinations []moneropay.Destination) (string, []int64, error) {
	if s.moneroPay == nil {
		return "", nil, fmt.Errorf("MoneroPay client not configured")
	}

	req := &moneropay.TransferRequest{
		Destinations:           destinations,
		SubtractFeeFromOutputs: make([]uint, len(destinations)),
		DoNotRelay:             false,
		Priority:               0,
	}
	for i := range destinations {
		req.SubtractFeeFromOutputs[i] = uint(i)
	}
	callCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	resp, err := s.moneroPay.PostTransfer(callCtx, req)
	if err != nil {
		return "", nil, err
	}
	if resp == nil {
		return "", nil, fmt.Errorf("MoneroPay transfer returned nil response")
	}

	txHash := resp.TxHash
	if txHash == "" && len(resp.TxHashList) > 0 {
		txHash = resp.TxHashList[0]
	}
	if txHash == "" {
		return "", nil, fmt.Errorf("MoneroPay transfer returned empty tx hash")
	}

	amounts := make([]int64, len(resp.Destinations))
	for i, dest := range resp.Destinations {
		amounts[i] = dest.Amount
	}
	if len(amounts) == 0 {
		amounts = make([]int64, len(destinations))
		for i, dest := range destinations {
			amounts[i] = dest.Amount
		}
	}

	return txHash, amounts, nil
}

func (s *VendorService) CreateVendor(ctx context.Context, name string, email string, password string, inviteCode string, moneroSubaddress string) (id uint, httpErr *models.HTTPError) {

	if len(name) < 3 || len(name) > 50 {
		return 0, models.NewHTTPError(http.StatusBadRequest, "name must be at least 3 characters and no more than 50 characters")
	}

	if len(password) < 8 || len(password) > 50 {
		return 0, models.NewHTTPError(http.StatusBadRequest, "password must be at least 8 characters and no more than 50 characters")
	}

	email = strings.TrimSpace(email)
	if email != "" && !isValidEmail(email) {
		return 0, models.NewHTTPError(http.StatusBadRequest, "invalid email address")
	}

	nameTaken, err := s.repo.VendorByNameExists(ctx, name)

	if err != nil {
		return 0, models.NewHTTPError(http.StatusInternalServerError, "error checking if vendor name exists: "+err.Error())
	}

	if nameTaken {
		return 0, models.NewHTTPError(http.StatusBadRequest, "Vendor username already taken. Please choose a different name")
	}

	invite, err := s.repo.FindInviteByCode(ctx, inviteCode)
	if err != nil {
		return 0, models.NewHTTPError(http.StatusBadRequest, "invalid invite code")
	}

	if invite.Used {
		return 0, models.NewHTTPError(http.StatusBadRequest, "invite code already used")
	}

	if invite.ForcedName != nil && *invite.ForcedName != name {
		return 0, models.NewHTTPError(http.StatusBadRequest, "invite code is for a different name")
	}

	moneroSubaddress = strings.TrimSpace(moneroSubaddress)
	if moneroSubaddress == "" {
		return 0, models.NewHTTPError(http.StatusBadRequest, "monero_subaddress is required")
	}

	if !moneroSubaddressRegex.MatchString(moneroSubaddress) {
		return 0, models.NewHTTPError(http.StatusBadRequest, "monero_subaddress is invalid")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return 0, models.NewHTTPError(http.StatusInternalServerError, "error hashing password: "+err.Error())
	}

	vendor := &models.Vendor{
		Name:             name,
		Email:            email,
		PasswordHash:     string(hashedPassword),
		MoneroSubaddress: moneroSubaddress,
	}

	err = s.repo.CreateVendor(ctx, vendor)
	if err != nil {
		return 0, models.NewHTTPError(http.StatusInternalServerError, "error creating vendor: "+err.Error())
	}

	err = s.repo.SetInviteToUsed(ctx, invite.ID)
	if err != nil {
		return 0, models.NewHTTPError(http.StatusInternalServerError, "error setting invite to used: "+err.Error())
	}

	return vendor.ID, nil
}

func (s *VendorService) DeleteVendor(ctx context.Context, vendorID uint) (httpErr *models.HTTPError) {
	vendor, err := s.repo.GetVendorByID(ctx, vendorID)
	if err != nil {
		return models.NewHTTPError(http.StatusInternalServerError, "error retrieving vendor: "+err.Error())
	}

	if vendor == nil {
		return models.NewHTTPError(http.StatusNotFound, "vendor not found")
	}

	if vendor.Balance != 0 {
		return models.NewHTTPError(http.StatusBadRequest, "vendor balance must be 0 to delete vendor")
	}

	err = s.repo.DeleteAllPosForVendor(ctx, vendorID)
	if err != nil {
		return models.NewHTTPError(http.StatusInternalServerError, "error deleting POS for vendor: "+err.Error())
	}

	err = s.repo.DeleteAllTransactionsForVendor(ctx, vendorID)
	if err != nil {
		return models.NewHTTPError(http.StatusInternalServerError, "error deleting transactions for vendor: "+err.Error())
	}

	err = s.repo.DeleteVendor(ctx, vendorID)
	if err != nil {
		return models.NewHTTPError(http.StatusInternalServerError, "error deleting vendor: "+err.Error())
	}

	return nil
}

func (s *VendorService) CreatePos(ctx context.Context, name string, password string, vendorID uint) (httpErr *models.HTTPError) {

	if len(name) < 3 || len(name) > 50 {
		return models.NewHTTPError(http.StatusBadRequest, "name must be at least 3 characters and no more than 50 characters")
	}

	if len(password) < 8 || len(password) > 50 {
		return models.NewHTTPError(http.StatusBadRequest, "password must be at least 8 characters and no more than 50 characters")
	}

	nameTaken, err := s.repo.PosByNameExistsForVendor(ctx, name, vendorID)

	if err != nil {
		return models.NewHTTPError(http.StatusBadRequest, "error checking if POS name exists: "+err.Error())
	}

	if nameTaken {
		return models.NewHTTPError(http.StatusBadRequest, "POS name already taken")
	}

	// check to see if vendor still exists. This is to prevent POS creation on deleted vendor, but probably needs to be done in a better way
	vendor, err := s.repo.GetVendorByID(ctx, vendorID)
	if err != nil {
		return models.NewHTTPError(http.StatusBadRequest, "error retrieving vendor: "+err.Error())
	}

	if vendor == nil {
		return models.NewHTTPError(http.StatusBadRequest, "vendor not found")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return models.NewHTTPError(http.StatusInternalServerError, "error hashing password: "+err.Error())
	}

	pos := &models.Pos{
		Name:         name,
		PasswordHash: string(hashedPassword),
		VendorID:     vendorID,
	}

	err = s.repo.CreatePos(ctx, pos)
	if err != nil {
		return models.NewHTTPError(http.StatusInternalServerError, "error creating POS: "+err.Error())
	}

	return nil
}

func (s *VendorService) GetBalance(ctx context.Context, _ uint) (*WalletBalance, *models.HTTPError) {
	if s.rpcClient == nil {
		return nil, models.NewHTTPError(http.StatusInternalServerError, "wallet RPC client not configured")
	}

	if ctx == nil {
		ctx = context.Background()
	}

	var resp struct {
		Balance         uint64 `json:"balance"`
		UnlockedBalance uint64 `json:"unlocked_balance"`
	}

	params := map[string]any{"account_index": 0}
	if err := s.rpcClient.Call(ctx, "get_balance", params, &resp); err != nil {
		return nil, models.NewHTTPError(http.StatusInternalServerError, "error retrieving wallet balance: "+err.Error())
	}

	locked := uint64(0)
	if resp.UnlockedBalance <= resp.Balance {
		locked = resp.Balance - resp.UnlockedBalance
	}

	return &WalletBalance{
		Total:    resp.Balance,
		Unlocked: resp.UnlockedBalance,
		Locked:   locked,
	}, nil
}

func (s *VendorService) GetVendorAccountBalance(ctx context.Context, vendorID uint) (int64, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	return s.repo.GetBalance(ctx, vendorID)
}

// GetWalletAddress returns the primary wallet address for account 0.
func (s *VendorService) GetWalletAddress(ctx context.Context) (string, *models.HTTPError) {
	if s.rpcClient == nil {
		return "", models.NewHTTPError(http.StatusInternalServerError, "wallet RPC client not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	var resp struct {
		Address      string `json:"address"`
		AddressIndex int    `json:"address_index"`
		Used         bool   `json:"used"`
	}
	params := map[string]any{"account_index": 0, "address_index": 0}
	if err := s.rpcClient.Call(ctx, "get_address", params, &resp); err != nil {
		return "", models.NewHTTPError(http.StatusInternalServerError, "error retrieving wallet address: "+err.Error())
	}
	return resp.Address, nil
}

// GetWalletInfo returns the primary address, wallet name, and network type.
func (s *VendorService) GetWalletInfo(ctx context.Context) (map[string]interface{}, *models.HTTPError) {
	if s.rpcClient == nil {
		return nil, models.NewHTTPError(http.StatusInternalServerError, "wallet RPC client not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	var addrResp struct {
		Address      string `json:"address"`
		AddressIndex int    `json:"address_index"`
	}
	addrParams := map[string]any{"account_index": 0, "address_index": 0}
	if err := s.rpcClient.Call(ctx, "get_address", addrParams, &addrResp); err != nil {
		return nil, models.NewHTTPError(http.StatusInternalServerError, "error retrieving wallet address: "+err.Error())
	}

	var netResp struct {
		Nettype string `json:"nettype"`
	}
	if err := s.rpcClient.Call(ctx, "query_key", map[string]any{"key_type": "mnemonic"}, &netResp); err != nil {
		// query_key doesn't always exist; ignore error
	}

	return map[string]interface{}{
		"address":     addrResp.Address,
		"wallet_name": s.config.WalletName,
		"network":     netResp.Nettype,
	}, nil
}

// SetupWallet creates a new wallet file via wallet-rpc.
func (s *VendorService) SetupWallet(ctx context.Context) *models.HTTPError {
	if s.rpcClient == nil {
		return models.NewHTTPError(http.StatusInternalServerError, "wallet RPC client not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	// If wallet is already open, just return success
	var addrResp struct {
		Address string `json:"address"`
	}
	if err := s.rpcClient.Call(ctx, "get_address", map[string]any{"account_index": 0}, &addrResp); err == nil && addrResp.Address != "" {
		return nil
	}

	// Close any currently open wallet first (ignore error if none open)
	_ = s.rpcClient.Call(ctx, "close_wallet", nil, nil)

	type createParams struct {
		Filename string `json:"filename"`
		Password string `json:"password,omitempty"`
		Language string `json:"language"`
	}
	req := createParams{
		Filename: s.config.WalletName,
		Language: "English",
	}
	if s.config.WalletPassword != "" {
		req.Password = s.config.WalletPassword
	}

	// Retry create_wallet with backoff — wallet-rpc may need a moment
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			log.Printf("Retrying create_wallet (attempt %d/3) after 3s...", attempt+1)
			time.Sleep(3 * time.Second)
		}
		createCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
		lastErr = s.rpcClient.Call(createCtx, "create_wallet", req, nil)
		cancel()
		if lastErr == nil {
			break
		}
	}
	if lastErr != nil {
		return models.NewHTTPError(http.StatusInternalServerError, "failed to create wallet: "+lastErr.Error())
	}

	// Open the newly created wallet with retry
	var openErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(2 * time.Second)
		}
		openCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		type openParams struct {
			Filename string `json:"filename"`
			Password string `json:"password,omitempty"`
		}
		openReq := openParams{Filename: s.config.WalletName}
		if s.config.WalletPassword != "" {
			openReq.Password = s.config.WalletPassword
		}
		openErr = s.rpcClient.Call(openCtx, "open_wallet", openReq, nil)
		cancel()
		if openErr == nil {
			break
		}
	}
	if openErr != nil {
		return models.NewHTTPError(http.StatusInternalServerError, "failed to open created wallet: "+openErr.Error())
	}

	return nil
}

// RestoreWallet restores a wallet from mnemonic seed via wallet-rpc.
func (s *VendorService) RestoreWallet(ctx context.Context, seed string, restoreHeight uint64) *models.HTTPError {
	if s.rpcClient == nil {
		return models.NewHTTPError(http.StatusInternalServerError, "wallet RPC client not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	seed = strings.TrimSpace(seed)
	if seed == "" {
		return models.NewHTTPError(http.StatusBadRequest, "mnemonic seed is required")
	}

	// Close any currently open wallet first
	_ = s.rpcClient.Call(ctx, "close_wallet", nil, nil)

	type restoreParams struct {
		Filename       string `json:"filename"`
		Password       string `json:"password,omitempty"`
		Seed           string `json:"seed"`
		SeedOffset     string `json:"seed_offset,omitempty"`
		RestoreHeight  uint64 `json:"restore_height,omitempty"`
		Language       string `json:"language"`
	}
	req := restoreParams{
		Filename:      s.config.WalletName,
		Seed:          seed,
		RestoreHeight: restoreHeight,
		Language:      "English",
	}
	if s.config.WalletPassword != "" {
		req.Password = s.config.WalletPassword
	}

	// Retry restore with backoff
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			log.Printf("Retrying restore_deterministic_wallet (attempt %d/3) after 5s...", attempt+1)
			time.Sleep(5 * time.Second)
		}
		restoreCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
		lastErr = s.rpcClient.Call(restoreCtx, "restore_deterministic_wallet", req, nil)
		cancel()
		if lastErr == nil {
			break
		}
	}
	if lastErr != nil {
		return models.NewHTTPError(http.StatusInternalServerError, "failed to restore wallet: "+lastErr.Error())
	}

	// Open the restored wallet with retry
	var openErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(2 * time.Second)
		}
		openCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		type openParams struct {
			Filename string `json:"filename"`
			Password string `json:"password,omitempty"`
		}
		openReq := openParams{Filename: s.config.WalletName}
		if s.config.WalletPassword != "" {
			openReq.Password = s.config.WalletPassword
		}
		openErr = s.rpcClient.Call(openCtx, "open_wallet", openReq, nil)
		cancel()
		if openErr == nil {
			break
		}
	}
	if openErr != nil {
		return models.NewHTTPError(http.StatusInternalServerError, "failed to open restored wallet: "+openErr.Error())
	}

	return nil
}

// GetWalletSeed returns the mnemonic seed for the currently open wallet.
func (s *VendorService) GetWalletSeed(ctx context.Context) (string, *models.HTTPError) {
	if s.rpcClient == nil {
		return "", models.NewHTTPError(http.StatusInternalServerError, "wallet RPC client not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	var resp struct {
		Key string `json:"key"`
	}
	if err := s.rpcClient.Call(ctx, "query_key", map[string]any{"key_type": "mnemonic"}, &resp); err != nil {
		return "", models.NewHTTPError(http.StatusInternalServerError, "failed to retrieve seed: "+err.Error())
	}
	return resp.Key, nil
}

func (s *VendorService) CreateTransfer(ctx context.Context, vendorID uint) *models.HTTPError {
	s.mu.Lock()
	defer s.mu.Unlock()

	vendor, err := s.repo.GetVendorByID(ctx, vendorID)
	if err != nil {
		return models.NewHTTPError(http.StatusInternalServerError, "DB error: "+err.Error())
	}
	if vendor == nil {
		return models.NewHTTPError(http.StatusBadRequest, "Vendor not found")
	}
	address := strings.TrimSpace(vendor.MoneroSubaddress)
	if address == "" {
		return models.NewHTTPError(http.StatusBadRequest, "Vendor is missing a Monero subaddress")
	}
	if !moneroSubaddressRegex.MatchString(address) {
		return models.NewHTTPError(http.StatusBadRequest, "Stored vendor subaddress is invalid")
	}

	// Check if vendor already has a transfer in progress
	transfer, err := s.repo.GetActiveTransferByVendorID(ctx, vendorID)
	if err != nil {
		return models.NewHTTPError(http.StatusInternalServerError, "DB error: "+err.Error())
	}
	if transfer != nil {
		return models.NewHTTPError(http.StatusBadRequest, "Transfer already in progress for this vendor")
	}

	transactions, err := s.repo.GetAllTransferableTransactions(ctx, vendorID)

	if err != nil {
		return models.NewHTTPError(http.StatusInternalServerError, "DB error: "+err.Error())
	}

	if len(transactions) == 0 {
		return models.NewHTTPError(http.StatusBadRequest, "No transferable transactions found for this vendor")
	}

	totalAmount := int64(0)

	for _, tx := range transactions {
		totalAmount += tx.Amount
	}

	// Do not allow withdrawals of less than 0.003 XMR as the fee is too high
	if totalAmount < 3000000 {
		return models.NewHTTPError(http.StatusBadRequest, "Minimum transfer amount is 0.003 XMR")
	}

	// Create a new transfer record
	newTransfer := &models.Transfer{
		VendorID:     vendorID,
		Amount:       totalAmount,
		Address:      address,
		Transactions: transactions,
	}

	err = s.repo.CreateTransfer(ctx, newTransfer)
	if err != nil {
		return models.NewHTTPError(http.StatusInternalServerError, "DB error: "+err.Error())
	}

	return nil
}

func (s *VendorService) ListPosDevices(ctx context.Context, vendorID uint) ([]*models.Pos, *models.HTTPError) {
	if ctx == nil {
		ctx = context.Background()
	}

	devices, err := s.repo.GetPosDevicesByVendorID(ctx, vendorID)
	if err != nil {
		return nil, models.NewHTTPError(http.StatusInternalServerError, "DB error: "+err.Error())
	}

	return devices, nil
}

type VendorTransactionSummary struct {
	ID          uint    `json:"id"`
	PosID       uint    `json:"pos_id"`
	PosName     string  `json:"pos_name"`
	Amount      int64   `json:"amount"`
	Description *string `json:"description"`
	Accepted    bool    `json:"accepted"`
	Confirmed   bool    `json:"confirmed"`
	Transferred bool    `json:"transferred"`
	CreatedAt   string  `json:"created_at"`
	TxHash      string  `json:"tx_hash,omitempty"`
}

type VendorListTransactionsResult struct {
	Confirmed []VendorTransactionSummary `json:"confirmed_transactions"`
	Pending   []VendorTransactionSummary `json:"pending_transactions"`
}

func (s *VendorService) ListTransactionsByVendor(ctx context.Context, vendorID uint) (*VendorListTransactionsResult, *models.HTTPError) {
	if ctx == nil {
		ctx = context.Background()
	}

	transactions, err := s.repo.FindTransactionsByVendorID(ctx, vendorID)
	if err != nil {
		return nil, models.NewHTTPError(http.StatusInternalServerError, "DB error: "+err.Error())
	}

	result := &VendorListTransactionsResult{
		Confirmed: make([]VendorTransactionSummary, 0),
		Pending:   make([]VendorTransactionSummary, 0),
	}

	for _, tx := range transactions {
		posName := ""
		if tx.Pos.Name != "" {
			posName = tx.Pos.Name
		}

		summary := VendorTransactionSummary{
			ID:          tx.ID,
			PosID:       tx.PosID,
			PosName:     posName,
			Amount:      tx.Amount,
			Description: tx.Description,
			Accepted:    tx.Accepted,
			Confirmed:   tx.Confirmed,
			Transferred: tx.Transferred,
			CreatedAt:   tx.CreatedAt.Format(time.RFC3339),
		}

		if tx.Confirmed && len(tx.SubTransactions) > 0 {
			summary.TxHash = tx.SubTransactions[0].TxHash
			result.Confirmed = append(result.Confirmed, summary)
		} else {
			result.Pending = append(result.Pending, summary)
		}
	}

	return result, nil
}

const moneroAtomicUnitsPerXMR int64 = 1_000_000_000_000

func (s *VendorService) ExportTransactionsByVendor(ctx context.Context, vendorID uint) (string, *models.HTTPError) {
	if ctx == nil {
		ctx = context.Background()
	}

	transactions, err := s.repo.FindTransactionsByVendorID(ctx, vendorID)
	if err != nil {
		return "", models.NewHTTPError(http.StatusInternalServerError, "DB error: "+err.Error())
	}

	var builder strings.Builder
	builder.WriteString("Koinly Date,Amount,Currency,Label,TxHash")

	rows := 0
	for _, transaction := range transactions {
		if !transaction.Confirmed {
			continue
		}

		for _, sub := range transaction.SubTransactions {
			builder.WriteByte('\n')
			date := sub.Timestamp.UTC()
			dateStr := fmt.Sprintf("%04d-%02d-%02d 00:00 UTC", date.Year(), date.Month(), date.Day())
			amount := formatAtomicAmount(sub.Amount)
			builder.WriteString(fmt.Sprintf("%s,%s,XMR,income,%s", dateStr, amount, sub.TxHash))
			rows++
		}
	}

	if rows == 0 {
		return "", models.NewHTTPError(http.StatusNotFound, "No confirmed transactions to export")
	}

	return builder.String(), nil
}

func formatAtomicAmount(amount int64) string {
	integer := amount / moneroAtomicUnitsPerXMR
	remainder := amount % moneroAtomicUnitsPerXMR

	if remainder < 0 {
		remainder += moneroAtomicUnitsPerXMR
		integer--
	}

	centsDivisor := moneroAtomicUnitsPerXMR / 100
	roundingOffset := centsDivisor / 2

	decimals := (remainder + roundingOffset) / centsDivisor
	if decimals == 100 {
		integer++
		decimals = 0
	}

	return fmt.Sprintf("%d.%02d", integer, decimals)
}
