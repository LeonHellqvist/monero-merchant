package server

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/monero-merchant/monero-merchant/backend/internal/core/config"
	"github.com/monero-merchant/monero-merchant/backend/internal/core/rpc"
	localMiddleware "github.com/monero-merchant/monero-merchant/backend/internal/core/server/middleware"
	"github.com/monero-merchant/monero-merchant/backend/internal/features/admin"
	"github.com/monero-merchant/monero-merchant/backend/internal/features/auth"
	"github.com/monero-merchant/monero-merchant/backend/internal/features/callback"
	"github.com/monero-merchant/monero-merchant/backend/internal/features/misc"
	"github.com/monero-merchant/monero-merchant/backend/internal/features/pos"
	"github.com/monero-merchant/monero-merchant/backend/internal/features/vendor"
	"github.com/monero-merchant/monero-merchant/backend/internal/thirdparty/moneropay"

	"gorm.io/gorm"
)

// Accept a context tied to server lifecycle to stop background workers on shutdown
func NewRouter(ctx context.Context, cfg *config.Config, db *gorm.DB, walletRPC, daemonRPC *rpc.Client, moneroPayClient *moneropay.MoneroPayAPIClient) *chi.Mux {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	if moneroPayClient == nil {
		moneroPayClient = moneropay.NewMoneroPayAPIClient()
		if cfg.MoneroPayBaseURL != "" {
			moneroPayClient.BaseURL = cfg.MoneroPayBaseURL
		}
	}

	if walletRPC == nil {
		walletRPC = rpc.NewClient(
			cfg.MoneroWalletRPCEndpoint,
			cfg.MoneroWalletRPCUsername,
			cfg.MoneroWalletRPCPassword,
		)
	}

	// Initialize repositories
	adminRepository := admin.NewAdminRepository(db)
	authRepository := auth.NewAuthRepository(db)
	vendorRepository := vendor.NewVendorRepository(db)
	posRepository := pos.NewPosRepository(db)
	callbackRepository := callback.NewCallbackRepository(db)
	miscRepository := misc.NewMiscRepository(db)

	// Initialize services
	vendorService := vendor.NewVendorService(vendorRepository, db, cfg, walletRPC, moneroPayClient)
	vendorService.StartTransferCompleter(ctx, 30*time.Second) // Check every 30 seconds
	adminService := admin.NewAdminService(adminRepository, cfg, vendorService)
	authService := auth.NewAuthService(authRepository, cfg)
	posService := pos.NewPosService(posRepository, cfg, moneroPayClient)
	posService.StartPendingCleanup(ctx, 15*time.Minute, 2*time.Hour)
	callbackService := callback.NewCallbackService(callbackRepository, cfg, moneroPayClient)
	callbackService.StartConfirmationChecker(ctx, 2*time.Second) // Check for confirmations every 2 seconds
	miscService := misc.NewMiscService(miscRepository, cfg, moneroPayClient)

	// Initialize handlers
	adminHandler := admin.NewAdminHandler(adminService, vendorService)
	authHandler := auth.NewAuthHandler(authService)
	vendorHandler := vendor.NewVendorHandler(vendorService)
	posHandler := pos.NewPosHandler(posService)
	callbackHandler := callback.NewCallbackHandler(callbackService)
	miscHandler := misc.NewMiscHandler(miscService)

	// Static dashboard files
	r.Get("/", serveDashboard)
	r.Get("/dashboard", serveDashboard)
	r.Get("/dashboard/*", serveDashboard)

	// Vendor self-service portal (signup, POS devices, payouts)
	r.Get("/store", serveStore)
	r.Get("/store/*", serveStore)

	// Public routes
	r.Group(func(r chi.Router) {
		// Auth routes
		r.Get("/auth/admin-status", authHandler.GetAdminStatus)
		r.Post("/auth/login-admin", authHandler.LoginAdmin)
		r.Post("/auth/login-vendor", authHandler.LoginVendor)
		r.Post("/auth/login-pos", authHandler.LoginPos)
		r.Post("/auth/refresh", authHandler.RefreshToken)

		// Vendor routes
		r.Post("/vendor/create", vendorHandler.CreateVendor)

		// Callback routes
		r.Post("/callback/receive/{jwt}", callbackHandler.ReceiveTransaction)
		r.Post("/receive/{jwt}", callbackHandler.ReceiveTransaction)
		r.Post("/callback/lws-hook/{jwt}", callbackHandler.LwsHook)

		// Miscellaneous routes
		r.Get("/misc/health", miscHandler.GetHealth)

		// Dashboard API — public status endpoint (no auth required)
		r.Get("/api/status", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"password_set": cfg.AdminPassword != "",
				"version":      "2.1.0",
			})
		})
		// Daemon sync status (no auth required)
		r.Get("/api/daemon-status", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			resp := map[string]interface{}{
				"synced":        false,
				"height":        0,
				"target_height": 0,
			}
			if daemonRPC != nil {
				var info struct {
					Height              uint64 `json:"height"`
					TargetHeight        uint64 `json:"target_height"`
					Synchronized        bool   `json:"synchronized"`
					Nettype             string `json:"nettype"`
					IncomingConnections uint64 `json:"incoming_connections_count"`
					OutgoingConnections uint64 `json:"outgoing_connections_count"`
				}
				ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
				defer cancel()
				if err := daemonRPC.Call(ctx, "get_info", nil, &info); err == nil {
					resp["synced"] = info.Synchronized
					resp["height"] = info.Height
					resp["target_height"] = info.TargetHeight
					resp["nettype"] = info.Nettype
					resp["incoming_connections"] = info.IncomingConnections
					resp["outgoing_connections"] = info.OutgoingConnections
				}
			}
			_ = json.NewEncoder(w).Encode(resp)
		})
		r.Get("/misc/health", miscHandler.GetHealth)
	})

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(localMiddleware.AuthMiddleware(cfg, authRepository))

		// Auth routes
		r.Post("/auth/update-password", authHandler.UpdatePassword)

		// Admin routes
		r.Post("/admin/invite", adminHandler.CreateInvite)
		r.Get("/admin/vendors", adminHandler.ListVendors)
		r.Get("/admin/balance", adminHandler.GetWalletBalance)
		r.Post("/admin/transfer-balance", adminHandler.TransferBalance)
		r.Post("/admin/delete", adminHandler.DeleteVendor)
		r.Get("/admin/invites", adminHandler.ListInvites)
		r.Get("/admin/transactions", adminHandler.ListTransactions)
		r.Get("/admin/pos-devices", adminHandler.ListPosDevices)
		r.Get("/admin/wallet-info", adminHandler.GetWalletInfo)
		r.Get("/admin/connection-info", adminHandler.GetConnectionInfo)
		r.Post("/admin/setup-wallet", adminHandler.SetupWallet)
		r.Post("/admin/restore-wallet", adminHandler.RestoreWallet)
		r.Get("/admin/wallet-seed", adminHandler.GetWalletSeed)

		// Vendor routes
		r.Post("/vendor/delete", vendorHandler.DeleteVendor)
		r.Post("/vendor/create-pos", vendorHandler.CreatePos)
		r.Get("/vendor/balance", vendorHandler.GetAccountBalance)
		r.Post("/vendor/transfer-balance", vendorHandler.TransferBalance)
		r.Get("/vendor/pos-list", vendorHandler.ListPosDevices)
		r.Get("/vendor/transactions", vendorHandler.ListTransactions)
		r.Get("/vendor/export", vendorHandler.ExportTransactions)

		// POS routes
		r.Post("/pos/create-transaction", posHandler.CreateTransaction)
		r.Get("/pos/transaction/{id}", posHandler.GetTransaction)
		r.Get("/pos/transactions", posHandler.ListTransactions)
		r.Get("/pos/balance", posHandler.GetPosBalance)
		r.Get("/pos/export", posHandler.ExportTransactions)
		r.HandleFunc("/pos/ws/transaction", posHandler.TransactionWS)
	})

	return r
}

// serveStore serves the vendor self-service portal (single-page app).
func serveStore(w http.ResponseWriter, r *http.Request) {
	dir := os.Getenv("DASHBOARD_DIR")
	if dir == "" {
		dir = "./web/dashboard"
	}
	storePath := filepath.Join(dir, "store.html")
	if _, err := os.Stat(storePath); os.IsNotExist(err) {
		http.Error(w, "Vendor portal not found", http.StatusNotFound)
		return
	}
	http.ServeFile(w, r, storePath)
}

// serveDashboard serves the embedded admin dashboard SPA.
// It looks for a DASHBOARD_DIR env var (Umbrel sets this to /data/dashboard)
// and falls back to ./web/dashboard/ for local development.
func serveDashboard(w http.ResponseWriter, r *http.Request) {
	dir := os.Getenv("DASHBOARD_DIR")
	if dir == "" {
		dir = "./web/dashboard"
	}

	// Strip leading "/dashboard" to get the file path
	path := strings.TrimPrefix(r.URL.Path, "/dashboard")
	if path == "" || path == "/" {
		path = "/index.html"
	}

	// Sanitize: prevent directory traversal
	path = filepath.Clean("/" + path)
	fullPath := filepath.Join(dir, path)

	// Ensure it's still within the dashboard dir
	if !strings.HasPrefix(fullPath, filepath.Clean(dir)) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// If file exists, serve it; otherwise serve index.html (SPA fallback)
	if _, err := os.Stat(fullPath); err == nil {
		http.ServeFile(w, r, fullPath)
		return
	}

	indexPath := dir + "/index.html"
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		http.Error(w, "Dashboard not found", http.StatusNotFound)
		return
	}
	http.ServeFile(w, r, indexPath)
}
