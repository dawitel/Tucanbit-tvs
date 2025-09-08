package verificationservice

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/tuncanbit/tvs/internal/domain"
	"github.com/tuncanbit/tvs/internal/infrastructure/clients"
	"github.com/tuncanbit/tvs/internal/infrastructure/rpc"
	"github.com/tuncanbit/tvs/internal/repositories/balancerepo"
	"github.com/tuncanbit/tvs/internal/repositories/sessionrepo"
	"github.com/tuncanbit/tvs/internal/repositories/transactionrepo"
	"github.com/tuncanbit/tvs/internal/repositories/withdrawalrepo"
	"github.com/tuncanbit/tvs/internal/server/websocket"
	"github.com/tuncanbit/tvs/pkg/config"
	"github.com/tuncanbit/tvs/pkg/currency"
)

type verificationService struct {
	sessionRepo       sessionrepo.ISessionRepository
	transactionRepo   transactionrepo.ITransactionRepository
	balanceRepo       balancerepo.IBalanceRepository
	withdrawalRepo    withdrawalrepo.IWithdrawalRepository
	config            config.VerificationConfig
	logger            zerolog.Logger
	heliusClient      *rpc.HeliusClient
	exchangeAPIClient *clients.ExchangeAPIClient
	currencyUtils     *currency.CurrencyUtils
	wsHub             *websocket.WsHub
}

func New(
	sessionRepo sessionrepo.ISessionRepository,
	transactionRepo transactionrepo.ITransactionRepository,
	balanceRepo balancerepo.IBalanceRepository,
	withdrawalRepo withdrawalrepo.IWithdrawalRepository,
	cfg config.VerificationConfig,
	logger zerolog.Logger,
	heliusClient *rpc.HeliusClient,
	exchangeAPIClient *clients.ExchangeAPIClient,
	wsHub *websocket.WsHub,
) IVerificationService {
	return &verificationService{
		sessionRepo:       sessionRepo,
		transactionRepo:   transactionRepo,
		balanceRepo:       balanceRepo,
		withdrawalRepo:    withdrawalRepo,
		config:            cfg,
		logger:            logger,
		heliusClient:      heliusClient,
		exchangeAPIClient: exchangeAPIClient,
		currencyUtils:     currency.NewCurrencyUtils(),
		wsHub:             wsHub,
	}
}

func (s *verificationService) StartTransactionVerification(ctx context.Context) error {
	s.logger.Info().Msg("Starting transaction verification service")

	ticker := time.NewTicker(time.Duration(s.config.PollingInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info().Msg("Transaction verification service stopped")
			return ctx.Err()
		case <-ticker.C:
			if err := s.processPendingSessions(ctx); err != nil {
				s.logger.Error().Err(err).Msg("Failed to process pending deposit sessions")
			}
			if err := s.processPendingWithdrawals(ctx); err != nil {
				s.logger.Error().Err(err).Msg("Failed to process pending withdrawals")
			}
		}
	}
}

func (s *verificationService) processPendingSessions(ctx context.Context) error {
	const limit = 100
	offset := 0
	chainSessions := make(map[string][]domain.DepositSession)

	for {
		sessions, err := s.sessionRepo.LoadPendingDepositSessions(ctx, limit, offset)
		if err != nil {
			return fmt.Errorf("failed to load pending deposit sessions: %w", err)
		}

		if len(sessions) == 0 {
			break
		}

		for _, session := range sessions {
			if isPDMChain(session.ChainID) {
				s.logger.Info().
					Str("session_id", session.SessionID).
					Str("chain_id", session.ChainID).
					Msg("Skipping verification for PDM chain")
				continue
			}

			if time.Since(session.CreatedAt) > time.Duration(s.config.SessionTimeoutHours)*time.Hour {
				session.Status = domain.SessionStatusExpired
				if err := s.sessionRepo.UpdateDepositSessionStatus(ctx, session.SessionID, string(domain.SessionStatusExpired), "Session expired"); err != nil {
					s.logger.Error().
						Str("session_id", session.SessionID).
						Err(err).
						Msg("Failed to mark session as expired")
				}
				s.wsHub.BroadcastDepositSession(session)
				continue
			}
			chainSessions[session.ChainID] = append(chainSessions[session.ChainID], session)
		}

		offset += limit
	}

	for chainID, sessions := range chainSessions {
		switch chainID {
		case "sol-mainnet", "sol-testnet":
			go s.processSolanaSessions(ctx, sessions)
		case "eth-mainnet", "eth-testnet":
			go s.processEthereumSessions(ctx, sessions)
		case "tron-mainnet":
			go s.processTronSessions(ctx, sessions)
		default:
			s.logger.Warn().
				Str("chain_id", chainID).
				Msg("No processor available for chain")
		}
	}

	return nil
}

func (s *verificationService) processPendingWithdrawals(ctx context.Context) error {
	const limit = 100
	offset := 0
	chainWithdrawals := make(map[string][]domain.Withdrawal)

	for {
		withdrawals, err := s.withdrawalRepo.LoadPendingWithdrawals(ctx, limit, offset)
		if err != nil {
			return fmt.Errorf("failed to load pending withdrawals: %w", err)
		}

		if len(withdrawals) == 0 {
			break
		}

		for _, withdrawal := range withdrawals {
			if isPDMChain(withdrawal.ChainID) {
				s.logger.Info().
					Str("withdrawal_id", withdrawal.WithdrawalID).
					Str("chain_id", withdrawal.ChainID).
					Msg("Skipping verification for PDM chain")
				continue
			}
			// Skip withdrawals without TxHash if they're too new
			if withdrawal.TxHash == "" && time.Since(withdrawal.CreatedAt) < time.Duration(s.config.PollingInterval)*2*time.Second {
				s.logger.Info().
					Str("withdrawal_id", withdrawal.WithdrawalID).
					Msg("No transaction hash for withdrawal, too early to verify")
				continue
			}
			if withdrawal.TxHash == "" {
				s.logger.Warn().
					Str("withdrawal_id", withdrawal.WithdrawalID).
					Msg("No transaction hash for withdrawal, marking as failed")
				withdrawal.Status = domain.WithdrawalStatusFailed
				if err := s.withdrawalRepo.UpdateWithdrawal(ctx, withdrawal); err != nil {
					s.logger.Error().
						Str("withdrawal_id", withdrawal.WithdrawalID).
						Err(err).
						Msg("Failed to mark withdrawal as failed")
				}
				continue
			}
			chainWithdrawals[withdrawal.ChainID] = append(chainWithdrawals[withdrawal.ChainID], withdrawal)
		}

		offset += limit
	}

	for chainID, withdrawals := range chainWithdrawals {
		switch chainID {
		case "sol-mainnet", "sol-testnet":
			go s.processSolanaWithdrawals(ctx, withdrawals)
		case "eth-mainnet", "eth-testnet":
			go s.processEthereumWithdrawals(ctx, withdrawals)
		case "tron-mainnet":
			go s.processTronWithdrawals(ctx, withdrawals)
		default:
			s.logger.Warn().
				Str("chain_id", chainID).
				Msg("No processor available for chain")
		}
	}

	return nil
}

func (s *verificationService) processSolanaSessions(ctx context.Context, sessions []domain.DepositSession) {
	const maxWorkers = 10
	semaphore := make(chan struct{}, maxWorkers)
	for _, session := range sessions {
		semaphore <- struct{}{}
		go func(session domain.DepositSession) {
			defer func() { <-semaphore }()
			if err := s.verifySolanaSession(ctx, session); err != nil {
				s.logger.Error().
					Str("session_id", session.SessionID).
					Err(err).
					Msg("Failed to verify Solana session")
			}
		}(session)
	}

	for i := 0; i < cap(semaphore); i++ {
		semaphore <- struct{}{}
	}
}

func (s *verificationService) processSolanaWithdrawals(ctx context.Context, withdrawals []domain.Withdrawal) {
	const maxWorkers = 10
	semaphore := make(chan struct{}, maxWorkers)
	for _, withdrawal := range withdrawals {
		semaphore <- struct{}{}
		go func(withdrawal domain.Withdrawal) {
			defer func() { <-semaphore }()
			if err := s.verifySolanaWithdrawal(ctx, withdrawal); err != nil {
				s.logger.Error().
					Str("withdrawal_id", withdrawal.WithdrawalID).
					Err(err).
					Msg("Failed to verify Solana withdrawal")
			}
		}(withdrawal)
	}

	for i := 0; i < cap(semaphore); i++ {
		semaphore <- struct{}{}
	}
}

func (s *verificationService) verifySolanaSession(ctx context.Context, session domain.DepositSession) error {
	tokenType, err := mapCryptoToSPLTokenType(session.CryptoCurrency)
	if err != nil {
		s.logger.Info().
			Str("session_id", session.SessionID).
			Str("crypto_currency", session.CryptoCurrency).
			Msg("Skipping verification for unsupported crypto currency")
		return nil
	}

	clusterType, err := mapChainIDToClusterType(session.ChainID)
	if err != nil {
		return fmt.Errorf("invalid chain_id %s: %w", session.ChainID, err)
	}

	decimals, err := s.heliusClient.GetDecimals(clusterType, tokenType)
	if err != nil {
		return fmt.Errorf("failed to get decimals for %s on %s: %w", tokenType, clusterType, err)
	}

	exchangeRate, err := s.exchangeAPIClient.GetExchangeRate(ctx, session.CryptoCurrency, "USD")
	if err != nil {
		return fmt.Errorf("Failed to get exchange rate, using default rate, %v", err)
	}

	requiredAmount := session.Amount // Already in decimal form (e.g., 3.0 USDC)

	params := rpc.VerifyDepositParams{
		Address:        session.WalletAddress,
		RequiredAmount: int64(requiredAmount * math.Pow(10, float64(decimals))), // Convert to lamports/units
		TokenType:      tokenType,
		ClusterType:    clusterType,
	}

	const maxRetries = 3
	var backoffBase = time.Duration(s.config.PollingInterval) * time.Second
	for attempt := 0; attempt <= maxRetries; attempt++ {
		isVerified, transactions, err := s.heliusClient.VerifyDeposit(ctx, params)
		if err == nil {
			if isVerified {
				return s.processVerifiedDeposit(ctx, session, transactions, tokenType, requiredAmount, clusterType, exchangeRate.Rate, decimals)
			}
			s.logger.Info().
				Str("session_id", session.SessionID).
				Int("attempt", attempt).
				Msg("No matching transaction found")
			if attempt == maxRetries {
				return nil
			}
		} else {
			if isTransientError(err) {
				s.logger.Warn().
					Str("session_id", session.SessionID).
					Int("attempt", attempt).
					Err(err).
					Msg("Transient error during verification, retrying")
				if attempt < maxRetries {
					time.Sleep(backoffBase * time.Duration(math.Pow(2, float64(attempt))))
					continue
				}
			}
			return fmt.Errorf("failed to verify deposit for session %s after %d attempts: %w", session.SessionID, maxRetries+1, err)
		}
	}
	return nil
}

func (s *verificationService) verifySolanaWithdrawal(ctx context.Context, withdrawal domain.Withdrawal) error {
	if withdrawal.TxHash == "" {
		s.logger.Warn().
			Str("withdrawal_id", withdrawal.WithdrawalID).
			Msg("No transaction hash for withdrawal, verification skipped")
		return nil
	}

	tokenType, err := mapCryptoToSPLTokenType(withdrawal.CryptoCurrency)
	if err != nil {
		s.logger.Info().
			Str("withdrawal_id", withdrawal.WithdrawalID).
			Str("crypto_currency", withdrawal.CryptoCurrency).
			Msg("Skipping verification for unsupported crypto currency")
		return nil
	}

	clusterType, err := mapChainIDToClusterType(withdrawal.ChainID)
	if err != nil {
		return fmt.Errorf("invalid chain_id %s: %w", withdrawal.ChainID, err)
	}

	amount, err := strconv.ParseFloat(withdrawal.CryptoAmount, 64)
	if err != nil {
		return fmt.Errorf("failed to parse withdrawal amount %s: %w", withdrawal.CryptoAmount, err)
	}

	params := rpc.VerifyWithdrawalParams{
		TxHash:      withdrawal.TxHash,
		ToAddress:   withdrawal.ToAddress,
		Amount:      amount, // Already in decimal form (e.g., 3.0 USDC)
		TokenType:   tokenType,
		ClusterType: clusterType,
	}

	const maxRetries = 3
	var backoffBase = time.Duration(s.config.PollingInterval) * time.Second
	for attempt := 0; attempt <= maxRetries; attempt++ {
		isVerified, transaction, err := s.heliusClient.VerifyWithdrawal(ctx, params)
		if err == nil {
			if isVerified {
				return s.processVerifiedWithdrawal(ctx, withdrawal, transaction)
			}
			s.logger.Info().
				Str("withdrawal_id", withdrawal.WithdrawalID).
				Int("attempt", attempt).
				Msg("No matching transaction found")
			if attempt == maxRetries {
				return s.markWithdrawalFailed(ctx, withdrawal, "No matching transaction found after retries")
			}
		} else {
			if isTransientError(err) {
				s.logger.Warn().
					Str("withdrawal_id", withdrawal.WithdrawalID).
					Int("attempt", attempt).
					Err(err).
					Msg("Transient error during withdrawal verification, retrying")
				if attempt < maxRetries {
					time.Sleep(backoffBase * time.Duration(math.Pow(2, float64(attempt))))
					continue
				}
			}
			return s.markWithdrawalFailed(ctx, withdrawal, fmt.Sprintf("Failed to verify after %d attempts: %v", maxRetries+1, err))
		}
	}
	return nil
}

func (s *verificationService) markWithdrawalFailed(ctx context.Context, withdrawal domain.Withdrawal, reason string) error {
	withdrawal.Status = domain.WithdrawalStatusFailed
	withdrawal.UpdatedAt = time.Now()
	if err := s.withdrawalRepo.UpdateWithdrawal(ctx, withdrawal); err != nil {
		s.logger.Error().
			Str("withdrawal_id", withdrawal.WithdrawalID).
			Err(err).
			Msg("Failed to update withdrawal status to failed")
		return fmt.Errorf("failed to update withdrawal %s status: %w", withdrawal.WithdrawalID, err)
	}

	if !withdrawal.ReservationReleased {
		if err := s.balanceRepo.ReleaseReservedBalance(ctx, withdrawal.UserID, "USD", withdrawal.AmountReservedCents); err != nil {
			s.logger.Error().
				Str("withdrawal_id", withdrawal.WithdrawalID).
				Err(err).
				Msg("Failed to release reserved balance")
		} else {
			releaseLog := &domain.BalanceLog{
				ID:           uuid.New().String(),
				UserID:       withdrawal.UserID,
				Component:    "withdrawal",
				CurrencyCode: "USD",
				ChangeCents:  withdrawal.AmountReservedCents,
				ChangeUnits:  float64(withdrawal.AmountReservedCents) / 100.0,
				Description:  fmt.Sprintf("Released reserved balance for failed withdrawal %s: %s", withdrawal.WithdrawalID, reason),
				Timestamp:    time.Now(),
			}
			if logErr := s.balanceRepo.LogBalanceChange(ctx, releaseLog); logErr != nil {
				s.logger.Err(logErr).Msg("Failed to log balance release")
			}
		}
		withdrawal.ReservationReleased = true
		withdrawal.ReservationReleasedAt = time.Now()
		if err := s.withdrawalRepo.UpdateWithdrawal(ctx, withdrawal); err != nil {
			s.logger.Error().
				Str("withdrawal_id", withdrawal.WithdrawalID).
				Err(err).
				Msg("Failed to update withdrawal after releasing balance")
		}
	}

	s.logger.Info().
		Str("withdrawal_id", withdrawal.WithdrawalID).
		Str("reason", reason).
		Msg("Withdrawal marked as failed")
	s.wsHub.BroadcastWithdrawal(withdrawal)
	return nil
}

func (s *verificationService) processVerifiedDeposit(ctx context.Context, session domain.DepositSession, transactions []domain.HeliusTransaction, tokenType domain.SPLTokenType, requiredAmount float64, clusterType domain.SolanaClusterType, exchangeRate float64, decimals int) error {
	var matchedTx domain.HeliusTransaction
	var txAmount float64
	for _, tx := range transactions {
		if tx.Type == "TRANSFER" {
			if tokenType == domain.SPLTokenTypeSOL {
				for _, transfer := range tx.NativeTransfers {
					adjustedAmount := float64(transfer.Amount) / math.Pow(10, float64(decimals))
					if transfer.ToUserAccount == session.WalletAddress && adjustedAmount >= requiredAmount {
						matchedTx = tx
						txAmount = adjustedAmount
						break
					}
				}
			} else {
				targetMint, err := s.heliusClient.GetMintAddress(clusterType, tokenType)
				if err != nil {
					return fmt.Errorf("failed to get mint address for %s: %w", tokenType, err)
				}
				for _, transfer := range tx.TokenTransfers {
					s.logger.Debug().
						Str("transaction", tx.Signature).
						Str("to", transfer.ToUserAccount).
						Str("mint", transfer.Mint).
						Float64("amount", transfer.TokenAmount).
						Msg("Inspecting token transfer in processVerifiedDeposit")

					if transfer.ToUserAccount == session.WalletAddress && transfer.Mint == targetMint && transfer.TokenAmount >= requiredAmount {
						matchedTx = tx
						txAmount = transfer.TokenAmount
						break
					}
				}
			}
			if matchedTx.Signature != "" {
				break
			}
		}
	}

	if matchedTx.Signature == "" {
		return fmt.Errorf("no matching transaction found despite verification for session %s", session.SessionID)
	}

	usdAmountCents := s.currencyUtils.CryptoToUSDCents(txAmount, exchangeRate)

	tx := domain.Transaction{
		ID:               uuid.New().String(),
		DepositSessionID: session.SessionID,
		ChainID:          session.ChainID,
		Network:          session.Network,
		CryptoCurrency:   session.CryptoCurrency,
		TxHash:           matchedTx.Signature,
		FromAddress:      getFromAddress(matchedTx, tokenType),
		ToAddress:        session.WalletAddress,
		Amount:           fmt.Sprintf("%.18f", txAmount),
		USDAmountCents:   usdAmountCents,
		ExchangeRate:     fmt.Sprintf("%.6f", exchangeRate),
		Fee:              fmt.Sprintf("%.18f", float64(matchedTx.Fee)/math.Pow(10, 9)),
		BlockNumber:      matchedTx.Slot,
		Status:           domain.StatusVerified,
		Confirmations:    1,
		Timestamp:        time.Unix(matchedTx.Timestamp, 0),
		VerifiedAt:       time.Now(),
		Processor:        domain.ProcessorInternal,
		TransactionType:  domain.TypeDeposit,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	metadata, err := json.Marshal(matchedTx)
	if err != nil {
		s.logger.Error().
			Err(err).
			Str("session_id", session.SessionID).
			Str("transaction_hash", matchedTx.Signature).
			Msg("Failed to marshal metadata")
		tx.Metadata = json.RawMessage("{}")
		session.Metadata = json.RawMessage("{}")
	} else {
		tx.Metadata = metadata
		session.Metadata = metadata
	}

	if err := s.transactionRepo.Create(ctx, tx); err != nil {
		return fmt.Errorf("failed to create transaction record for session %s: %w", session.SessionID, err)
	}

	balance, err := s.balanceRepo.GetBalance(ctx, session.UserID, "USD")
	if err != nil {
		return fmt.Errorf("failed to get balance for user %s: %w", session.UserID, err)
	}

	newAmountCents := balance.AmountCents + usdAmountCents
	currentAmountUnits, err := strconv.ParseFloat(balance.AmountUnits, 64)
	if err != nil {
		s.logger.Warn().
			Str("user_id", session.UserID).
			Str("amount_units", balance.AmountUnits).
			Msg("Invalid balance.AmountUnits, defaulting to 0")
		currentAmountUnits = 0
	}
	newAmountUnits := fmt.Sprintf("%.18f", currentAmountUnits+(txAmount*exchangeRate))

	if newAmountCents < 0 || currentAmountUnits+(txAmount*exchangeRate) < 0 {
		return fmt.Errorf("invalid balance update for session %s: negative balance", session.SessionID)
	}

	if err := s.balanceRepo.UpdateBalance(ctx, session.UserID, "USD", newAmountCents, newAmountUnits); err != nil {
		return fmt.Errorf("failed to update balance for user %s: %w", session.UserID, err)
	}

	updatedBalance := domain.Balance{
		ID:            balance.ID,
		UserID:        session.UserID,
		CurrencyCode:  "USD",
		AmountCents:   newAmountCents,
		AmountUnits:   newAmountUnits,
		ReservedCents: balance.ReservedCents,
		ReservedUnits: balance.ReservedUnits,
		UpdatedAt:     time.Now(),
	}

	session.Status = domain.SessionStatusCompleted
	session.UpdatedAt = time.Now()

	if err := s.sessionRepo.CompleteSession(ctx, session, ""); err != nil {
		return fmt.Errorf("failed to update session %s status: %w", session.SessionID, err)
	}

	s.logger.Info().
		Str("session_id", session.SessionID).
		Str("transaction_hash", matchedTx.Signature).
		Float64("amount", txAmount).
		Int64("usd_amount_cents", usdAmountCents).
		Msg("Transaction verified, session updated, and balance updated")

	s.wsHub.BroadcastDepositSession(session)
	s.wsHub.BroadcastBalance(updatedBalance)

	return nil
}

func (s *verificationService) processVerifiedWithdrawal(ctx context.Context, withdrawal domain.Withdrawal, transaction domain.HeliusTransaction) error {
	metadata, err := json.Marshal(transaction)
	if err != nil {
		s.logger.Error().
			Err(err).
			Str("withdrawal_id", withdrawal.WithdrawalID).
			Str("transaction_hash", transaction.Signature).
			Msg("Failed to marshal metadata")
		metadata = json.RawMessage("{}")
	}

	amount, err := strconv.ParseFloat(withdrawal.CryptoAmount, 64)
	if err != nil {
		return fmt.Errorf("failed to parse withdrawal amount %s: %w", withdrawal.CryptoAmount, err)
	}
	exchangeRate, err := strconv.ParseFloat(withdrawal.ExchangeRate, 64)
	if err != nil {
		return fmt.Errorf("failed to parse exchange rate %s: %w", withdrawal.ExchangeRate, err)
	}

	tx := domain.Transaction{
		ID:              uuid.New().String(),
		WithdrawalID:    withdrawal.WithdrawalID,
		ChainID:         withdrawal.ChainID,
		Network:         withdrawal.Network,
		CryptoCurrency:  withdrawal.CryptoCurrency,
		TxHash:          transaction.Signature,
		FromAddress:     withdrawal.SourceWalletAddress,
		ToAddress:       withdrawal.ToAddress,
		Amount:          withdrawal.CryptoAmount,
		USDAmountCents:  withdrawal.USDAmountCents,
		ExchangeRate:    withdrawal.ExchangeRate,
		Fee:             fmt.Sprintf("%.18f", float64(transaction.Fee)/math.Pow(10, 9)),
		BlockNumber:     transaction.Slot,
		Status:          domain.StatusVerified,
		Confirmations:   1,
		Timestamp:       time.Unix(transaction.Timestamp, 0),
		VerifiedAt:      time.Now(),
		Processor:       domain.ProcessorInternal,
		TransactionType: domain.TypeWithdrawal,
		Metadata:        metadata,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := s.transactionRepo.Create(ctx, tx); err != nil {
		return fmt.Errorf("failed to create transaction record for withdrawal %s: %w", withdrawal.WithdrawalID, err)
	}

	balance, err := s.balanceRepo.GetBalance(ctx, withdrawal.UserID, "USD")
	if err != nil {
		return fmt.Errorf("failed to get balance for user %s: %w", withdrawal.UserID, err)
	}
	newAmountCents := balance.AmountCents - withdrawal.USDAmountCents
	currentAmountUnits, err := strconv.ParseFloat(balance.AmountUnits, 64)
	if err != nil {
		s.logger.Warn().
			Str("user_id", withdrawal.UserID).
			Str("amount_units", balance.AmountUnits).
			Msg("Invalid balance.AmountUnits, defaulting to 0")
		currentAmountUnits = 0
	}
	newAmountUnits := fmt.Sprintf("%.18f", currentAmountUnits-(amount*exchangeRate))
	if newAmountCents < 0 || currentAmountUnits-(amount*exchangeRate) < 0 {
		return fmt.Errorf("insufficient balance for withdrawal %s", withdrawal.WithdrawalID)
	}

	if err := s.balanceRepo.UpdateBalance(ctx, withdrawal.UserID, "USD", newAmountCents, newAmountUnits); err != nil {
		return fmt.Errorf("failed to update balance for user %s: %w", withdrawal.UserID, err)
	}

	updatedBalance := domain.Balance{
		ID:            balance.ID,
		UserID:        withdrawal.UserID,
		CurrencyCode:  "USD",
		AmountCents:   newAmountCents,
		AmountUnits:   newAmountUnits,
		ReservedCents: balance.ReservedCents,
		ReservedUnits: balance.ReservedUnits,
		UpdatedAt:     time.Now(),
	}

	if !withdrawal.ReservationReleased {
		if err := s.balanceRepo.ReleaseReservedBalance(ctx, withdrawal.UserID, "USD", withdrawal.AmountReservedCents); err != nil {
			return fmt.Errorf("failed to release reserved balance for withdrawal %s: %w", withdrawal.WithdrawalID, err)
		}
		withdrawal.ReservationReleased = true
		withdrawal.ReservationReleasedAt = time.Now()
	}

	withdrawal.Status = domain.WithdrawalStatusCompleted
	withdrawal.UpdatedAt = time.Now()
	if err := s.withdrawalRepo.UpdateWithdrawal(ctx, withdrawal); err != nil {
		return fmt.Errorf("failed to update withdrawal %s status: %w", withdrawal.WithdrawalID, err)
	}

	s.logger.Info().
		Str("withdrawal_id", withdrawal.WithdrawalID).
		Str("transaction_hash", transaction.Signature).
		Float64("crypto_amount", amount).
		Int64("usd_amount_cents", withdrawal.USDAmountCents).
		Msg("Withdrawal verified, transaction recorded, and balance updated")

	s.wsHub.BroadcastWithdrawal(withdrawal)
	s.wsHub.BroadcastBalance(updatedBalance)

	return nil
}

func (s *verificationService) processEthereumSessions(ctx context.Context, sessions []domain.DepositSession) {
	for _, session := range sessions {
		s.logger.Warn().
			Str("session_id", session.SessionID).
			Str("chain_id", session.ChainID).
			Msg("Ethereum session verification not implemented yet")
	}
}

func (s *verificationService) processTronSessions(ctx context.Context, sessions []domain.DepositSession) {
	for _, session := range sessions {
		s.logger.Warn().
			Str("session_id", session.SessionID).
			Str("chain_id", session.ChainID).
			Msg("Tron session verification not implemented yet")
	}
}

func (s *verificationService) processEthereumWithdrawals(ctx context.Context, withdrawals []domain.Withdrawal) {
	for _, withdrawal := range withdrawals {
		s.logger.Warn().
			Str("withdrawal_id", withdrawal.WithdrawalID).
			Str("chain_id", withdrawal.ChainID).
			Msg("Ethereum withdrawal verification not implemented yet")
	}
}

func (s *verificationService) processTronWithdrawals(ctx context.Context, withdrawals []domain.Withdrawal) {
	for _, withdrawal := range withdrawals {
		s.logger.Warn().
			Str("withdrawal_id", withdrawal.WithdrawalID).
			Str("chain_id", withdrawal.ChainID).
			Msg("Tron withdrawal verification not implemented yet")
	}
}

func (s *verificationService) VerifyTransactionFromPDMWebhook(ctx context.Context, req domain.PDMWebhookRequest) error {
	return nil
}

func isTransientError(err error) bool {
	if netErr, ok := err.(net.Error); ok && (netErr.Timeout() || netErr.Temporary()) {
		return true
	}

	if strings.Contains(err.Error(), "API request failed with status") {
		if strings.Contains(err.Error(), "429 Too Many Requests") {
			return true
		}
		if strings.Contains(err.Error(), "5") {
			return true
		}
		if strings.Contains(err.Error(), "400 Bad Request") || strings.Contains(err.Error(), "401 Unauthorized") {
			return false
		}
	}

	if strings.Contains(err.Error(), "invalid chain_id") ||
		strings.Contains(err.Error(), "failed to parse JSON response") ||
		strings.Contains(err.Error(), "no mint address configured") ||
		strings.Contains(err.Error(), "no Helius base URL configured") {
		return false
	}

	return true
}

func isSolanaChain(chainID string) bool {
	return chainID == "sol-mainnet" || chainID == "sol-testnet"
}

func isPDMChain(chainID string) bool {
	return chainID == "btc-mainnet" || chainID == "btc-testnet"
}

func mapCryptoToSPLTokenType(crypto string) (domain.SPLTokenType, error) {
	switch crypto {
	case "SOL":
		return domain.SPLTokenTypeSOL, nil
	case "USDC":
		return domain.SPLTokenTypeUSDC, nil
	case "USDT":
		return domain.SPLTokenTypeUSDT, nil
	default:
		return "", fmt.Errorf("unsupported crypto currency: %s", crypto)
	}
}

func mapChainIDToClusterType(chainID string) (domain.SolanaClusterType, error) {
	switch chainID {
	case "sol-mainnet":
		return domain.SolanaClusterTypeMainnet, nil
	case "sol-testnet":
		return domain.SolanaClusterTypeTestnet, nil
	default:
		return "", fmt.Errorf("unsupported Solana chain: %s", chainID)
	}
}

func getFromAddress(tx domain.HeliusTransaction, tokenType domain.SPLTokenType) string {
	if tokenType == domain.SPLTokenTypeSOL {
		for _, transfer := range tx.NativeTransfers {
			if transfer.ToUserAccount != "" {
				return transfer.FromUserAccount
			}
		}
	} else {
		for _, transfer := range tx.TokenTransfers {
			if transfer.ToUserAccount != "" {
				return transfer.FromUserAccount
			}
		}
	}
	return ""
}

func parseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}
