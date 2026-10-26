package vendor

import (
	"context"
	"errors"
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
	mu        sync.Mutex
}

func NewVendorService(repo VendorRepository, db *gorm.DB, cfg *config.Config, rpcClient *rpc.Client) *VendorService {
	return &VendorService{repo: repo, db: db, config: cfg, rpcClient: rpcClient}
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

	// A new tx could spend the same outputs as an unrelayed one
	if !s.relayPendingTransfers(ctx) {
		return
	}

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

			txHash, metadata, amounts, err := s.buildTransfer(ctx, destinations)
			if err != nil {
				// Possibly too big for one tx, try a smaller batch
				log.Printf("Building transfer of %d payouts failed: %v", len(transfers), err)
				_ = dbTx.Rollback()
				return
			}

			for index, transfer := range transfers {
				amountTransferred := transfer.Amount
				if len(amounts) > index && amounts[index] != 0 {
					amountTransferred = amounts[index]
				}
				if err := s.repo.SetTransferTx(ctx, dbTx, transfer.ID, amountTransferred, txHash, metadata); err != nil {
					log.Println("Error saving transfer tx:", err)
					batchErr = err
					_ = dbTx.Rollback()
					return
				}
			}
			// Save the tx before relaying it, so a failed relay is retried with the same tx instead of paying again
			if err := dbTx.Commit().Error; err != nil {
				log.Println("Error committing transaction:", err)
				batchErr = err
				return
			}
			if err := s.relayTransfer(ctx, txHash, metadata); err != nil {
				log.Printf("Relaying transfer %s failed: %v", txHash, err)
				batchErr = err
				return
			}
			log.Println("Transfer completed successfully")
		}() // end per-iteration scope
		if batchErr != nil {
			break
		}
	}

}

// buildTransfer creates the tx without relaying it
func (s *VendorService) buildTransfer(ctx context.Context, destinations []moneropay.Destination) (string, string, []int64, error) {
	type rpcDestination struct {
		Amount  int64  `json:"amount"`
		Address string `json:"address"`
	}

	type transferParams struct {
		Destinations           []rpcDestination `json:"destinations"`
		SubtractFeeFromOutputs []uint           `json:"subtract_fee_from_outputs,omitempty"`
		DoNotRelay             bool             `json:"do_not_relay"`
		GetTxMetadata          bool             `json:"get_tx_metadata"`
	}

	params := transferParams{
		Destinations:           make([]rpcDestination, len(destinations)),
		SubtractFeeFromOutputs: make([]uint, 0, len(destinations)),
		DoNotRelay:             true,
		GetTxMetadata:          true,
	}

	for i, dest := range destinations {
		params.Destinations[i] = rpcDestination{
			Amount:  dest.Amount,
			Address: dest.Address,
		}
		params.SubtractFeeFromOutputs = append(params.SubtractFeeFromOutputs, uint(i))
	}

	var result struct {
		AmountsByDest struct {
			Amounts []int64 `json:"amounts"`
		} `json:"amounts_by_dest"`
		TxHash     string `json:"tx_hash"`
		TxMetadata string `json:"tx_metadata"`
	}

	callCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := s.rpcClient.Call(callCtx, "transfer", params, &result); err != nil {
		return "", "", nil, err
	}

	return result.TxHash, result.TxMetadata, result.AmountsByDest.Amounts, nil
}

func (s *VendorService) relayTransfer(ctx context.Context, txHash string, metadata string) error {
	callCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if err := s.rpcClient.Call(callCtx, "relay_tx", map[string]any{"hex": metadata}, nil); err != nil {
		return err
	}
	return s.repo.MarkTransfersCompleted(ctx, txHash)
}

// relayPendingTransfers retries txs whose relay failed, and reports whether none are left
func (s *VendorService) relayPendingTransfers(ctx context.Context) bool {
	transfers, err := s.repo.GetUnrelayedTransfers(ctx)
	if err != nil {
		log.Println("Error fetching unrelayed transfers:", err)
		return false
	}

	ok := true
	done := map[string]bool{}
	for _, transfer := range transfers {
		txHash := *transfer.TxHash
		if done[txHash] {
			continue
		}
		done[txHash] = true

		// The wallet knows the tx if an earlier relay went through, unless the daemon dropped it
		var known struct {
			Transfer struct {
				Type string `json:"type"`
			} `json:"transfer"`
		}
		err := s.rpcClient.Call(ctx, "get_transfer_by_txid", map[string]any{"txid": txHash}, &known)
		if err == nil && known.Transfer.Type != "failed" {
			err = s.repo.MarkTransfersCompleted(ctx, txHash)
		} else {
			err = s.relayTransfer(ctx, txHash, *transfer.TxMetadata)
		}
		if err != nil {
			log.Printf("Relaying transfer %s failed: %v", txHash, err)
			ok = false
		}
	}
	return ok
}

func (s *VendorService) CreateVendor(ctx context.Context, name string, email string, password string, inviteCode string, moneroSubaddress string) (id uint, httpErr *models.HTTPError) {

	if len(name) < 3 || len(name) > 50 {
		return 0, models.NewHTTPError(http.StatusBadRequest, "name must be at least 3 characters and no more than 50 characters")
	}

	if len(password) < 8 || len(password) > 72 {
		return 0, models.NewHTTPError(http.StatusBadRequest, "password must be at least 8 characters and no more than 72 characters")
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

	if time.Now().After(invite.ValidUntil) {
		return 0, models.NewHTTPError(http.StatusBadRequest, "invite code expired")
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

	err = s.repo.CreateVendorWithInvite(ctx, vendor, invite.ID)
	if errors.Is(err, ErrInviteUsed) {
		return 0, models.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err != nil {
		return 0, models.NewHTTPError(http.StatusInternalServerError, "error creating vendor: "+err.Error())
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

	balance, err := s.repo.GetBalance(ctx, vendorID)
	if err != nil {
		return models.NewHTTPError(http.StatusInternalServerError, "error retrieving balance: "+err.Error())
	}

	if balance != 0 {
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

	if len(password) < 8 || len(password) > 72 {
		return models.NewHTTPError(http.StatusBadRequest, "password must be at least 8 characters and no more than 72 characters")
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

func (s *VendorService) GetVendorAccountBalance(ctx context.Context, vendorID uint) (int64, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	return s.repo.GetBalance(ctx, vendorID)
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
