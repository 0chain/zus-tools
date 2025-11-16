package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Configuration constants - can be overridden by environment variables
var (
	ZCN_DIR                   = getEnvOrDefault("ZCN_DIR", "/home/ubuntu/.zcn")
	ZWALLET_PATH              = getEnvOrDefault("ZWALLET_PATH", "/opt/0chain/zwalletcli/zwallet")
	ZBOX_PATH                 = getEnvOrDefault("ZBOX_PATH", "/opt/0chain/zboxcli/zbox")
	METADATA_PATH             = getEnvOrDefault("METADATA_PATH", "/var/lib/zs3/metadata.env")
	BASELINE_FILE             = getEnvOrDefault("BASELINE_FILE", "/var/lib/zs3/initial_balances.json")
	LOG_FILE                  = getEnvOrDefault("LOG_FILE", "/var/log/zs3_monitoring.log")
	BALANCE_THRESHOLD_PERCENT = 50
	THRESHOLD_DAYS            = 30
)

// getEnvOrDefault returns environment variable value or default if not set
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// WalletBalance represents a wallet balance entry
type WalletBalance struct {
	Balance int `json:"balance"`
}

// BalancesMap represents the balances structure
type BalancesMap map[string]WalletBalance

// Allocation represents an allocation entry
type Allocation struct {
	ID             string `json:"id"`
	ExpirationDate int64  `json:"expiration_date"`
	Expiration     int64  `json:"expiration"`
	Size           int64  `json:"size"`
}

// Metadata represents the metadata configuration
type Metadata struct {
	BlobberIDList   string `json:"BLOBBER_ID_LIST"`
	ClusterID       string `json:"CLUSTER_ID"`
	UserID          string `json:"USER_ID"`
	ClientID        string `json:"CLIENT_ID"`
	AuthTicketsFile string `json:"AUTH_TICKETS_FILE"`
}

// ZS3Monitor represents the monitoring service
type ZS3Monitor struct {
	metadata    *Metadata
	zs3ClientID string
	blobberIDs  []string
	logger      *log.Logger
}

func (m *ZS3Monitor) logCommand(cmd *exec.Cmd) {
	if m == nil || m.logger == nil || cmd == nil {
		return
	}
	path := cmd.Path
	if path == "" && len(cmd.Args) > 0 {
		path = cmd.Args[0]
	}
	m.logger.Printf("→ Executing command: %s %s", path, strings.Join(cmd.Args[1:], " "))
	if cmd.Dir != "" {
		m.logger.Printf("   • Working directory: %s", cmd.Dir)
	}
	if len(cmd.Env) > 0 {
		m.logger.Printf("   • Custom environment entries: %d", len(cmd.Env))
	}
}

func (m *ZS3Monitor) logCommandOutput(label string, output []byte) {
	if m == nil || m.logger == nil || len(output) == 0 {
		return
	}
	m.logger.Printf("%s\n%s", label, strings.TrimSpace(string(output)))
}

func (m *ZS3Monitor) logStep(message string) {
	if m == nil || m.logger == nil {
		return
	}
	m.logger.Println(message)
}

// NewZS3Monitor creates a new monitoring instance
func NewZS3Monitor() (*ZS3Monitor, error) {
	// Setup logging
	logFile, err := os.OpenFile(LOG_FILE, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %v", err)
	}

	logger := log.New(logFile, "", 0)
	logger.SetFlags(0) // Remove timestamp prefix since we add our own

	monitor := &ZS3Monitor{
		logger: logger,
	}

	// Load metadata
	if err := monitor.loadMetadata(); err != nil {
		monitor.logger.Printf("Warning: Failed to load metadata: %v", err)
		monitor.logger.Println("Continuing with available data...")
	}

	// Determine wallet IDs
	monitor.determineWalletIDs()

	return monitor, nil
}

// loadMetadata loads the metadata configuration
func (m *ZS3Monitor) loadMetadata() error {
	if _, err := os.Stat(METADATA_PATH); os.IsNotExist(err) {
		if m.logger != nil {
			m.logger.Printf("Metadata file %s not found on disk", METADATA_PATH)
		}
		return fmt.Errorf("metadata file %s not found", METADATA_PATH)
	}

	if m.logger != nil {
		m.logger.Printf("Loading metadata from %s", METADATA_PATH)
	}

	content, err := ioutil.ReadFile(METADATA_PATH)
	if err != nil {
		return fmt.Errorf("failed to read metadata file: %v", err)
	}

	// Parse environment file format
	lines := strings.Split(string(content), "\n")
	metadata := &Metadata{}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(strings.Trim(parts[1], "\""))

		switch key {
		case "BLOBBER_ID_LIST":
			if m.logger != nil {
				m.logger.Printf("Metadata BLOBBER_ID_LIST=%s", value)
			}
			metadata.BlobberIDList = value
		case "CLUSTER_ID":
			if m.logger != nil {
				m.logger.Printf("Metadata CLUSTER_ID=%s", value)
			}
			metadata.ClusterID = value
		case "USER_ID":
			if m.logger != nil {
				m.logger.Printf("Metadata USER_ID=%s", value)
			}
			metadata.UserID = value
		case "CLIENT_ID":
			if m.logger != nil {
				m.logger.Printf("Metadata CLIENT_ID=%s", value)
			}
			metadata.ClientID = value
		case "AUTH_TICKETS_FILE":
			if m.logger != nil {
				m.logger.Printf("Metadata AUTH_TICKETS_FILE=%s", value)
			}
			metadata.AuthTicketsFile = value
		}
	}

	m.metadata = metadata
	return nil
}

// determineWalletIDs determines the ZS3 client ID and blobber IDs
func (m *ZS3Monitor) determineWalletIDs() {
	// Get ZS3 client ID
	zs3WalletPath := filepath.Join(ZCN_DIR, "wallet.json")
	if _, err := os.Stat(zs3WalletPath); err == nil {
		if m.logger != nil {
			m.logger.Printf("Reading ZS3 wallet from %s", zs3WalletPath)
		}
		content, err := ioutil.ReadFile(zs3WalletPath)
		if err == nil {
			var wallet map[string]interface{}
			if json.Unmarshal(content, &wallet) == nil {
				if clientID, ok := wallet["client_id"].(string); ok {
					m.zs3ClientID = clientID
					if m.logger != nil {
						m.logger.Printf("Extracted ZS3 client ID from wallet.json: %s", m.zs3ClientID)
					}
				}
			}
		}
	}

	// Fallback to metadata client ID
	if m.zs3ClientID == "" && m.metadata != nil {
		m.zs3ClientID = m.metadata.ClientID
		if m.logger != nil && m.zs3ClientID != "" {
			m.logger.Printf("Using ZS3 client ID from metadata: %s", m.zs3ClientID)
		}
	}

	// Get blobber IDs from metadata or auth tickets file
	var blobberIDList string
	if m.metadata != nil && m.metadata.BlobberIDList != "" {
		if m.logger != nil {
			m.logger.Printf("Using blobber IDs from metadata list")
		}
		blobberIDList = m.metadata.BlobberIDList
	} else if m.metadata != nil && m.metadata.AuthTicketsFile != "" {
		if _, err := os.Stat(m.metadata.AuthTicketsFile); err == nil {
			if m.logger != nil {
				m.logger.Printf("Reading blobber IDs from auth tickets file %s", m.metadata.AuthTicketsFile)
			}
			content, err := ioutil.ReadFile(m.metadata.AuthTicketsFile)
			if err == nil {
				var authTickets map[string]interface{}
				if json.Unmarshal(content, &authTickets) == nil {
					var keys []string
					for key := range authTickets {
						keys = append(keys, key)
					}
					blobberIDList = strings.Join(keys, ",")
				}
			}
		}
	}

	if blobberIDList != "" {
		m.blobberIDs = strings.Split(blobberIDList, ",")
		// Clean up blobber IDs
		var cleanIDs []string
		for _, id := range m.blobberIDs {
			id = strings.TrimSpace(id)
			if id != "" {
				cleanIDs = append(cleanIDs, id)
			}
		}
		m.blobberIDs = cleanIDs
		if m.logger != nil {
			m.logger.Printf("Resolved %d blobber ID(s)", len(m.blobberIDs))
		}
	} else {
		m.logger.Println("Warning: No blobber IDs found")
		m.blobberIDs = []string{}
	}

	m.logger.Printf("Monitoring: ZS3=%s, Blobbers=%d", m.zs3ClientID, len(m.blobberIDs))
}

// readBalance reads the balance for a given client ID
func (m *ZS3Monitor) readBalance(clientID string) (int, error) {
	if m.logger != nil {
		m.logger.Printf("Preparing balance check for client %s", clientID)
	}
	if _, err := os.Stat(ZWALLET_PATH); os.IsNotExist(err) {
		m.logger.Printf("Warning: zwallet not found, using 0 for %s", clientID)
		return 0, nil
	}

	// Create temporary wallet file
	tempWallet, err := os.CreateTemp("/home/ubuntu/.zcn/", "temp_wallet_*.json")
	if err != nil {
		return 0, fmt.Errorf("failed to create temp wallet: %v", err)
	}
	defer os.Remove(tempWallet.Name())
	if m.logger != nil {
		m.logger.Printf("Created temporary wallet at %s", tempWallet.Name())
	}

	walletData := map[string]interface{}{
		"client_id":  clientID,
		"client_key": "temp",
		"keys":       []interface{}{},
	}

	walletJSON, err := json.Marshal(walletData)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal wallet data: %v", err)
	}

	if _, err := tempWallet.Write(walletJSON); err != nil {
		return 0, fmt.Errorf("failed to write temp wallet: %v", err)
	}
	tempWallet.Close()
	if m.logger != nil {
		m.logger.Printf("Temporary wallet populated for %s", clientID)
	}

	fileName := filepath.Base(tempWallet.Name())

	// Execute zwallet getbalance command
	cmd := exec.Command(ZWALLET_PATH, "getbalance", "--wallet", fileName)
	cmd.Dir = "/opt/0chain/zwalletcli"
	m.logCommand(cmd)
	output, err := cmd.Output()
	if err != nil {
		m.logCommandOutput("Command stdout before error (may be empty):", output)
		m.logger.Printf("Warning: Failed to get balance for %s: %v", clientID, err)
		return 0, nil
	}
	m.logCommandOutput("Command stdout (zwallet getbalance):", output)

	// Parse balance from output (format: "Balance: 7.2807487746 ZCN")
	outputStr := string(output)
	lines := strings.Split(outputStr, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Balance:") {
			parts := strings.Split(line, "Balance:")
			if len(parts) > 1 {
				balanceStr := strings.TrimSpace(parts[1])
				balanceStr = strings.Split(balanceStr, " ")[0] // Remove "ZCN" part
				if balance, err := strconv.ParseFloat(balanceStr, 64); err == nil {
					if m.logger != nil {
						m.logger.Printf("Parsed balance %.4f for client %s", balance, clientID)
					}
					return int(balance), nil
				}
			}
		}
	}

	if m.logger != nil {
		m.logger.Printf("Could not parse balance for client %s; defaulting to 0", clientID)
	}
	return 0, nil
}

// fundFromZS3 funds a blobber wallet from the ZS3 wallet
func (m *ZS3Monitor) fundFromZS3(toClientID string, neededTokens int) error {
	if neededTokens <= 0 {
		return nil
	}

	if _, err := os.Stat(ZWALLET_PATH); os.IsNotExist(err) {
		m.logger.Printf("Warning: zwallet not found, cannot fund %s", toClientID)
		return fmt.Errorf("zwallet not found")
	}

	zs3WalletPath := filepath.Join(ZCN_DIR, "wallet.json")
	m.logger.Printf("→ Funding %d tokens from ZS3 to %s", neededTokens, toClientID)

	cmd := exec.Command(ZWALLET_PATH, "send",
		"--wallet", zs3WalletPath,
		"--to_client_id", toClientID,
		"--tokens", strconv.Itoa(neededTokens),
		"--desc", "auto-fund-monitoring")
	cmd.Dir = "/opt/0chain/zwalletcli"

	m.logCommand(cmd)

	output, err := cmd.CombinedOutput()
	if err != nil {
		m.logger.Printf("Warning: Failed to fund %s: %v", toClientID, err)
		m.logger.Printf("Command output: %s", string(output))
		return err
	}

	m.logCommandOutput("Funding command output:", output)
	m.logger.Printf("Successfully funded %d tokens to %s", neededTokens, toClientID)
	return nil
}

// calcTopupNeeded calculates how many tokens are needed to reach threshold
func (m *ZS3Monitor) calcTopupNeeded(currentBalance, baselineBalance int) int {
	if baselineBalance <= 0 {
		return 0
	}

	threshold := baselineBalance * BALANCE_THRESHOLD_PERCENT / 100
	if currentBalance < threshold {
		return threshold - currentBalance
	}
	return 0
}

// getCurrentBalances gets current balances for all wallets
func (m *ZS3Monitor) getCurrentBalances() (BalancesMap, error) {
	balances := make(BalancesMap)

	// Get ZS3 balance
	if m.zs3ClientID != "" {
		balance, err := m.readBalance(m.zs3ClientID)
		if err != nil {
			m.logger.Printf("Warning: Failed to read ZS3 balance: %v", err)
		} else {
			balances[m.zs3ClientID] = WalletBalance{Balance: balance}
			m.logger.Printf("ZS3 balance: %d", balance)
		}
	}

	// Get blobber balances
	for _, blobberID := range m.blobberIDs {
		balance, err := m.readBalance(blobberID)
		if err != nil {
			m.logger.Printf("Warning: Failed to read balance for blobber %s: %v", blobberID, err)
		} else {
			balances[blobberID] = WalletBalance{Balance: balance}
			m.logger.Printf("Blobber %s balance: %d", blobberID, balance)
		}
	}

	return balances, nil
}

// loadBaselineBalances loads baseline balances from file
func (m *ZS3Monitor) loadBaselineBalances() (BalancesMap, error) {
	if _, err := os.Stat(BASELINE_FILE); os.IsNotExist(err) {
		if m.logger != nil {
			m.logger.Printf("Baseline file %s does not exist yet", BASELINE_FILE)
		}
		return make(BalancesMap), nil
	}

	content, err := ioutil.ReadFile(BASELINE_FILE)
	if err != nil {
		if m.logger != nil {
			m.logger.Printf("Failed reading baseline file %s: %v", BASELINE_FILE, err)
		}
		return nil, fmt.Errorf("failed to read baseline file: %v", err)
	}

	var balances BalancesMap
	if err := json.Unmarshal(content, &balances); err != nil {
		if m.logger != nil {
			m.logger.Printf("Baseline file parse error: %v", err)
		}
		return nil, fmt.Errorf("failed to parse baseline file: %v", err)
	}

	return balances, nil
}

// saveBaselineBalances saves current balances as baseline
func (m *ZS3Monitor) saveBaselineBalances(balances BalancesMap) error {
	// Ensure directory exists
	dir := filepath.Dir(BASELINE_FILE)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}
	if m.logger != nil {
		m.logger.Printf("Saving baseline balances for %d wallet(s) to %s", len(balances), BASELINE_FILE)
	}

	content, err := json.MarshalIndent(balances, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal balances: %v", err)
	}

	if err := ioutil.WriteFile(BASELINE_FILE, content, 0644); err != nil {
		return fmt.Errorf("failed to write baseline file: %v", err)
	}
	if m.logger != nil {
		m.logger.Printf("Baseline balances saved to %s", BASELINE_FILE)
	}

	return nil
}

// monitorBalances monitors wallet balances and performs auto-funding
func (m *ZS3Monitor) monitorBalances() error {
	m.logStep("Starting balance collection step")
	// Get current balances
	currentBalances, err := m.getCurrentBalances()
	if err != nil {
		return fmt.Errorf("failed to get current balances: %v", err)
	}
	if m.logger != nil {
		m.logger.Printf("Current balance snapshot contains %d wallet(s)", len(currentBalances))
	}

	// Load baseline balances
	baselineBalances, err := m.loadBaselineBalances()
	if err != nil {
		return fmt.Errorf("failed to load baseline balances: %v", err)
	}
	if m.logger != nil {
		m.logger.Printf("Baseline file entries loaded: %d", len(baselineBalances))
	}

	// Initialize baseline if not present
	if len(baselineBalances) == 0 {
		if err := m.saveBaselineBalances(currentBalances); err != nil {
			m.logger.Printf("Warning: Failed to save baseline balances: %v", err)
		} else {
			m.logger.Println("Initialized baseline balances from current state")
		}
		baselineBalances = currentBalances
		if m.logger != nil {
			m.logger.Printf("Baseline initialized with %d wallet(s)", len(baselineBalances))
		}
	}

	// Check ZS3 wallet balance
	if m.zs3ClientID != "" {
		currentZS3 := currentBalances[m.zs3ClientID].Balance
		baselineZS3 := baselineBalances[m.zs3ClientID].Balance

		if baselineZS3 > 0 {
			topupNeeded := m.calcTopupNeeded(currentZS3, baselineZS3)
			if topupNeeded > 0 {
				m.logger.Printf("ZS3 wallet below %d%% baseline (%d tokens needed)", BALANCE_THRESHOLD_PERCENT, topupNeeded)
				m.logger.Println("Note: ZS3 wallet is the original funding source - no external funding available")
			} else {
				m.logger.Println("ZS3 wallet balance OK")
			}
		}
	}

	// Check and fund blobber wallets from ZS3
	for _, blobberID := range m.blobberIDs {
		currentBal := currentBalances[blobberID].Balance
		baselineBal := baselineBalances[blobberID].Balance
		if m.logger != nil {
			m.logger.Printf("Evaluating blobber %s: current=%d baseline=%d", blobberID, currentBal, baselineBal)
		}

		if baselineBal > 0 {
			topupNeeded := m.calcTopupNeeded(currentBal, baselineBal)
			if topupNeeded > 0 {
				m.logger.Printf("Blobber %s below %d%% baseline, funding %d tokens from ZS3", blobberID, BALANCE_THRESHOLD_PERCENT, topupNeeded)
				if err := m.fundFromZS3(blobberID, topupNeeded); err != nil {
					m.logger.Printf("Warning: Failed to fund blobber %s: %v", blobberID, err)
				}
			} else {
				m.logger.Printf("Blobber %s balance OK", blobberID)
			}
		}
	}

	return nil
}

// listAllocations lists all allocations
func (m *ZS3Monitor) listAllocations() ([]Allocation, error) {
	if _, err := os.Stat(ZBOX_PATH); os.IsNotExist(err) {
		return nil, fmt.Errorf("zbox CLI not found")
	}

	zs3WalletPath := filepath.Join(ZCN_DIR, "wallet.json")
	cmd := exec.Command(ZBOX_PATH, "listallocations", "config", "", "--wallet", zs3WalletPath, "--json")
	cmd.Dir = "/opt/0chain/zboxcli"
	m.logCommand(cmd)

	output, err := cmd.Output()
	if err != nil {
		m.logCommandOutput("listallocations output before error:", output)
		return nil, fmt.Errorf("failed to list allocations: %v", err)
	}
	m.logCommandOutput("listallocations stdout:", output)

	scanner := bufio.NewScanner(bytes.NewReader(output))
	var builder strings.Builder
	started := false
	for scanner.Scan() {
		line := scanner.Text()
		trim := strings.TrimSpace(line)
		if !started && (strings.HasPrefix(trim, "{") || strings.HasPrefix(trim, "[")) {
			started = true
		}
		if started {
			builder.WriteString(line)
			builder.WriteString("\n")
		}
	}
	jsonPart := strings.TrimSpace(builder.String())
	if jsonPart == "" {
		return nil, fmt.Errorf("failed to locate JSON payload in listallocations output")
	}
	payload := []byte(jsonPart)
	if len(payload) > 0 {
		switch payload[0] {
		case '[':
			if idx := bytes.LastIndexByte(payload, ']'); idx >= 0 {
				payload = payload[:idx+1]
			}
		case '{':
			if idx := bytes.LastIndexByte(payload, '}'); idx >= 0 {
				payload = payload[:idx+1]
			}
		}
	}
	trimmed := bytes.TrimSpace(payload)
	if m.logger != nil && !bytes.Equal(trimmed, output) {
		m.logger.Printf("Extracted JSON payload of length %d from listallocations output (raw length %d)", len(trimmed), len(output))
	}

	var allocations []Allocation
	if err := json.Unmarshal(trimmed, &allocations); err != nil {
		return nil, fmt.Errorf("failed to parse allocations: %v", err)
	}

	return allocations, nil
}

// updateAllocation updates an allocation
func (m *ZS3Monitor) updateAllocation(allocationID string, extendDays int, lockTokens int) error {
	if _, err := os.Stat(ZBOX_PATH); os.IsNotExist(err) {
		return fmt.Errorf("zbox CLI not found")
	}

	zs3WalletPath := filepath.Join(ZCN_DIR, "wallet.json")
	args := []string{
		"updateallocation",
		"--wallet", zs3WalletPath,
		"--allocation", allocationID,
		"--extend", fmt.Sprintf("%dd", extendDays),
		"--lock", strconv.Itoa(lockTokens),
	}

	cmd := exec.Command(ZBOX_PATH, args...)
	cmd.Dir = "/opt/0chain/zboxcli"
	m.logCommand(cmd)
	if m.logger != nil {
		m.logger.Printf("Update allocation parameters: extend=%dd lock=%d", extendDays, lockTokens)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		m.logger.Printf("Warning: Failed to extend allocation %s: %v", allocationID, err)
		m.logger.Printf("Command output: %s", string(output))
		return err
	}

	m.logCommandOutput("updateallocation output:", output)

	m.logger.Printf("Successfully extended allocation %s", allocationID)
	return nil
}

// monitorAllocations monitors allocations and renews expiring ones
func (m *ZS3Monitor) monitorAllocations() error {
	m.logger.Println("Checking allocation expiry...")
	m.logStep("Fetching allocation list via zbox CLI")

	allocations, err := m.listAllocations()
	if err != nil {
		m.logger.Printf("Warning: Failed to list allocations: %v", err)
		return err
	}

	m.logger.Printf("Found %d allocation(s)", len(allocations))

	if len(allocations) == 0 {
		m.logger.Println("No allocations found")
		return nil
	}

	now := time.Now().Unix()
	cutoff := now + int64(THRESHOLD_DAYS*24*3600)

	for _, allocation := range allocations {
		if allocation.ID == "" || allocation.ID == "null" {
			continue
		}
		if m.logger != nil {
			m.logger.Printf("Evaluating allocation %s (expiration_date=%d expiration=%d)", allocation.ID, allocation.ExpirationDate, allocation.Expiration)
		}

		// Use expiration_date or expiration field
		expiration := allocation.ExpirationDate
		if expiration == 0 {
			expiration = allocation.Expiration
		}

		if expiration <= 0 {
			if m.logger != nil {
				m.logger.Printf("Allocation %s has no valid expiration, skipping", allocation.ID)
			}
			continue
		}

		if expiration < cutoff {
			m.logger.Printf("→ Allocation %s expires within %d days, extending...", allocation.ID, THRESHOLD_DAYS)
			if err := m.updateAllocation(allocation.ID, THRESHOLD_DAYS, 1); err != nil {
				m.logger.Printf("Warning: Failed to extend allocation %s: %v", allocation.ID, err)
			}
		} else {
			daysLeft := (expiration - now) / 86400
			m.logger.Printf("→ Allocation %s expires in %d days (OK)", allocation.ID, daysLeft)
		}
	}

	return nil
}

// Run performs the complete monitoring run
func (m *ZS3Monitor) Run() error {
	m.logger.Printf("=== [%s] ZS3 monitoring run start ===", time.Now().Format(time.RFC3339))
	m.logStep("Step 1: Balance monitoring")

	// Monitor balances
	if err := m.monitorBalances(); err != nil {
		m.logger.Printf("Error during balance monitoring: %v", err)
	}

	m.logStep("Step 2: Allocation monitoring")

	// Monitor allocations
	if err := m.monitorAllocations(); err != nil {
		m.logger.Printf("Error during allocation monitoring: %v", err)
	}

	m.logger.Printf("=== [%s] ZS3 monitoring run completed ===", time.Now().Format(time.RFC3339))
	return nil
}

func main() {
	monitor, err := NewZS3Monitor()
	if err != nil {
		log.Fatalf("Failed to create monitor: %v", err)
	}

	if err := monitor.Run(); err != nil {
		log.Fatalf("Monitoring run failed: %v", err)
	}
}
