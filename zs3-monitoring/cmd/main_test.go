package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Test helper functions
func createTempDir(t *testing.T) string {
	dir, err := ioutil.TempDir("", "zs3_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	return dir
}

func createTempFile(t *testing.T, dir, filename, content string) string {
	filepath := filepath.Join(dir, filename)
	err := ioutil.WriteFile(filepath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file %s: %v", filepath, err)
	}
	return filepath
}

func cleanupTempDir(t *testing.T, dir string) {
	err := os.RemoveAll(dir)
	if err != nil {
		t.Logf("Failed to cleanup temp dir %s: %v", dir, err)
	}
}

// Test getEnvOrDefault
func TestGetEnvOrDefault(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		expected     string
	}{
		{
			name:         "Environment variable set",
			key:          "TEST_VAR",
			defaultValue: "default",
			envValue:     "env_value",
			expected:     "env_value",
		},
		{
			name:         "Environment variable not set",
			key:          "NONEXISTENT_VAR",
			defaultValue: "default",
			envValue:     "",
			expected:     "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable if needed
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}

			result := getEnvOrDefault(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getEnvOrDefault(%s, %s) = %s, expected %s", tt.key, tt.defaultValue, result, tt.expected)
			}
		})
	}
}

// Test Metadata struct and parsing
func TestLoadMetadata(t *testing.T) {
	tempDir := createTempDir(t)
	defer cleanupTempDir(t, tempDir)

	tests := []struct {
		name            string
		metadataContent string
		expected        *Metadata
		expectError     bool
	}{
		{
			name: "Valid metadata file",
			metadataContent: `BLOBBER_ID_LIST="blobber1,blobber2,blobber3"
CLUSTER_ID="test-cluster"
USER_ID="test-user"
CLIENT_ID="test-client-id"
AUTH_TICKETS_FILE="/path/to/auth.json"`,
			expected: &Metadata{
				BlobberIDList:   "blobber1,blobber2,blobber3",
				ClusterID:       "test-cluster",
				UserID:          "test-user",
				ClientID:        "test-client-id",
				AuthTicketsFile: "/path/to/auth.json",
			},
			expectError: false,
		},
		{
			name: "Metadata file with comments and empty lines",
			metadataContent: `# This is a comment
BLOBBER_ID_LIST="blobber1,blobber2"

# Another comment
CLUSTER_ID="test-cluster"

USER_ID="test-user"`,
			expected: &Metadata{
				BlobberIDList: "blobber1,blobber2",
				ClusterID:     "test-cluster",
				UserID:        "test-user",
			},
			expectError: false,
		},
		{
			name:            "Empty metadata file",
			metadataContent: ``,
			expected:        &Metadata{},
			expectError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create metadata file
			metadataFile := createTempFile(t, tempDir, "metadata.env", tt.metadataContent)

			// Create monitor with custom metadata path
			monitor := &ZS3Monitor{
				logger: log.New(os.Stdout, "", 0),
			}

			// Override METADATA_PATH for this test
			originalPath := METADATA_PATH
			METADATA_PATH = metadataFile
			defer func() { METADATA_PATH = originalPath }()

			err := monitor.loadMetadata()

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.expectError {
				if monitor.metadata == nil {
					t.Errorf("Expected metadata to be loaded")
					return
				}

				if monitor.metadata.BlobberIDList != tt.expected.BlobberIDList {
					t.Errorf("BlobberIDList = %s, expected %s", monitor.metadata.BlobberIDList, tt.expected.BlobberIDList)
				}
				if monitor.metadata.ClusterID != tt.expected.ClusterID {
					t.Errorf("ClusterID = %s, expected %s", monitor.metadata.ClusterID, tt.expected.ClusterID)
				}
				if monitor.metadata.UserID != tt.expected.UserID {
					t.Errorf("UserID = %s, expected %s", monitor.metadata.UserID, tt.expected.UserID)
				}
				if monitor.metadata.ClientID != tt.expected.ClientID {
					t.Errorf("ClientID = %s, expected %s", monitor.metadata.ClientID, tt.expected.ClientID)
				}
				if monitor.metadata.AuthTicketsFile != tt.expected.AuthTicketsFile {
					t.Errorf("AuthTicketsFile = %s, expected %s", monitor.metadata.AuthTicketsFile, tt.expected.AuthTicketsFile)
				}
			}
		})
	}
}

// Test determineWalletIDs
func TestDetermineWalletIDs(t *testing.T) {
	tempDir := createTempDir(t)
	defer cleanupTempDir(t, tempDir)

	tests := []struct {
		name             string
		walletContent    string
		metadata         *Metadata
		expectedZS3ID    string
		expectedBlobbers []string
	}{
		{
			name: "ZS3 wallet file exists",
			walletContent: `{
				"client_id": "zs3-client-123",
				"client_key": "test-key",
				"keys": []
			}`,
			metadata: &Metadata{
				BlobberIDList: "blobber1,blobber2",
			},
			expectedZS3ID:    "zs3-client-123",
			expectedBlobbers: []string{"blobber1", "blobber2"},
		},
		{
			name: "Fallback to metadata client ID",
			walletContent: `{
				"invalid": "json"
			}`,
			metadata: &Metadata{
				ClientID:      "metadata-client-456",
				BlobberIDList: "blobber3,blobber4",
			},
			expectedZS3ID:    "metadata-client-456",
			expectedBlobbers: []string{"blobber3", "blobber4"},
		},
		{
			name: "Blobber IDs from auth tickets file",
			walletContent: `{
				"client_id": "zs3-client-789"
			}`,
			metadata: &Metadata{
				AuthTicketsFile: createTempFile(t, tempDir, "auth.json", `{
					"blobber5": {"ticket": "ticket1"},
					"blobber6": {"ticket": "ticket2"}
				}`),
			},
			expectedZS3ID:    "zs3-client-789",
			expectedBlobbers: []string{"blobber5", "blobber6"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create wallet file
			createTempFile(t, tempDir, "wallet.json", tt.walletContent)

			// Create monitor
			var logBuffer bytes.Buffer
			monitor := &ZS3Monitor{
				metadata: tt.metadata,
				logger:   log.New(&logBuffer, "", 0),
			}

			// Override ZCN_DIR for this test
			originalZCN := ZCN_DIR
			ZCN_DIR = tempDir
			defer func() { ZCN_DIR = originalZCN }()

			monitor.determineWalletIDs()

			if monitor.zs3ClientID != tt.expectedZS3ID {
				t.Errorf("ZS3 Client ID = %s, expected %s", monitor.zs3ClientID, tt.expectedZS3ID)
			}

			if len(monitor.blobberIDs) != len(tt.expectedBlobbers) {
				t.Errorf("Blobber IDs length = %d, expected %d", len(monitor.blobberIDs), len(tt.expectedBlobbers))
			}

			for i, expectedID := range tt.expectedBlobbers {
				if i < len(monitor.blobberIDs) && monitor.blobberIDs[i] != expectedID {
					t.Errorf("Blobber ID[%d] = %s, expected %s", i, monitor.blobberIDs[i], expectedID)
				}
			}
		})
	}
}

// Test calcTopupNeeded
func TestCalcTopupNeeded(t *testing.T) {
	monitor := &ZS3Monitor{}

	tests := []struct {
		name            string
		currentBalance  int
		baselineBalance int
		expected        int
	}{
		{
			name:            "No topup needed - current above threshold",
			currentBalance:  60,
			baselineBalance: 100,
			expected:        0,
		},
		{
			name:            "Topup needed - current below threshold",
			currentBalance:  30,
			baselineBalance: 100,
			expected:        20, // 50% of 100 = 50, 50 - 30 = 20
		},
		{
			name:            "No baseline - no topup",
			currentBalance:  50,
			baselineBalance: 0,
			expected:        0,
		},
		{
			name:            "Exact threshold - no topup",
			currentBalance:  50,
			baselineBalance: 100,
			expected:        0,
		},
		{
			name:            "Negative baseline - no topup",
			currentBalance:  50,
			baselineBalance: -10,
			expected:        0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := monitor.calcTopupNeeded(tt.currentBalance, tt.baselineBalance)
			if result != tt.expected {
				t.Errorf("calcTopupNeeded(%d, %d) = %d, expected %d", tt.currentBalance, tt.baselineBalance, result, tt.expected)
			}
		})
	}
}

// Test loadBaselineBalances and saveBaselineBalances
func TestBaselineBalances(t *testing.T) {
	tempDir := createTempDir(t)
	defer cleanupTempDir(t, tempDir)

	// Override BASELINE_FILE for this test
	originalBaseline := BASELINE_FILE
	BASELINE_FILE = filepath.Join(tempDir, "baseline.json")
	defer func() { BASELINE_FILE = originalBaseline }()

	monitor := &ZS3Monitor{
		logger: log.New(os.Stdout, "", 0),
	}

	// Test saving baseline balances
	testBalances := BalancesMap{
		"client1": {Balance: 100},
		"client2": {Balance: 200},
		"client3": {Balance: 300},
	}

	err := monitor.saveBaselineBalances(testBalances)
	if err != nil {
		t.Fatalf("Failed to save baseline balances: %v", err)
	}

	// Test loading baseline balances
	loadedBalances, err := monitor.loadBaselineBalances()
	if err != nil {
		t.Fatalf("Failed to load baseline balances: %v", err)
	}

	if len(loadedBalances) != len(testBalances) {
		t.Errorf("Loaded balances length = %d, expected %d", len(loadedBalances), len(testBalances))
	}

	for clientID, expectedBalance := range testBalances {
		if loadedBalance, exists := loadedBalances[clientID]; !exists {
			t.Errorf("Client %s not found in loaded balances", clientID)
		} else if loadedBalance.Balance != expectedBalance.Balance {
			t.Errorf("Client %s balance = %d, expected %d", clientID, loadedBalance.Balance, expectedBalance.Balance)
		}
	}

	// Test loading non-existent file
	BASELINE_FILE = filepath.Join(tempDir, "nonexistent.json")
	emptyBalances, err := monitor.loadBaselineBalances()
	if err != nil {
		t.Errorf("Expected no error for non-existent file, got: %v", err)
	}
	if len(emptyBalances) != 0 {
		t.Errorf("Expected empty balances for non-existent file, got %d entries", len(emptyBalances))
	}
}

// Test Allocation struct and JSON parsing
func TestAllocationParsing(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		expected Allocation
	}{
		{
			name: "Allocation with expiration_date",
			jsonData: `{
				"id": "alloc123",
				"expiration_date": 1640995200,
				"size": 1000000000
			}`,
			expected: Allocation{
				ID:             "alloc123",
				ExpirationDate: 1640995200,
				Expiration:     0,
				Size:           1000000000,
			},
		},
		{
			name: "Allocation with expiration field",
			jsonData: `{
				"id": "alloc456",
				"expiration": 1640995200,
				"size": 2000000000
			}`,
			expected: Allocation{
				ID:             "alloc456",
				ExpirationDate: 0,
				Expiration:     1640995200,
				Size:           2000000000,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var allocation Allocation
			err := json.Unmarshal([]byte(tt.jsonData), &allocation)
			if err != nil {
				t.Fatalf("Failed to unmarshal allocation: %v", err)
			}

			if allocation.ID != tt.expected.ID {
				t.Errorf("ID = %s, expected %s", allocation.ID, tt.expected.ID)
			}
			if allocation.ExpirationDate != tt.expected.ExpirationDate {
				t.Errorf("ExpirationDate = %d, expected %d", allocation.ExpirationDate, tt.expected.ExpirationDate)
			}
			if allocation.Expiration != tt.expected.Expiration {
				t.Errorf("Expiration = %d, expected %d", allocation.Expiration, tt.expected.Expiration)
			}
			if allocation.Size != tt.expected.Size {
				t.Errorf("Size = %d, expected %d", allocation.Size, tt.expected.Size)
			}
		})
	}
}

// Test monitorAllocations logic
func TestMonitorAllocationsLogic(t *testing.T) {
	now := time.Now().Unix()

	tests := []struct {
		name        string
		allocations []Allocation
		expectRenew bool
	}{
		{
			name: "Allocation expiring soon - should renew",
			allocations: []Allocation{
				{
					ID:             "alloc1",
					ExpirationDate: now + 15*24*3600, // 15 days from now
				},
			},
			expectRenew: true,
		},
		{
			name: "Allocation expiring far - should not renew",
			allocations: []Allocation{
				{
					ID:             "alloc2",
					ExpirationDate: now + 45*24*3600, // 45 days from now
				},
			},
			expectRenew: false,
		},
		{
			name: "Allocation with expiration field",
			allocations: []Allocation{
				{
					ID:         "alloc3",
					Expiration: now + 20*24*3600, // 20 days from now
				},
			},
			expectRenew: true,
		},
		{
			name: "Invalid allocation - should skip",
			allocations: []Allocation{
				{
					ID:             "",
					ExpirationDate: now + 15*24*3600,
				},
				{
					ID:             "null",
					ExpirationDate: now + 15*24*3600,
				},
				{
					ID:             "alloc4",
					ExpirationDate: 0,
				},
			},
			expectRenew: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test focuses on the logic, not the actual command execution
			// We'll test the expiration calculation logic

			cutoff := now + int64(THRESHOLD_DAYS*24*3600)

			for _, allocation := range tt.allocations {
				if allocation.ID == "" || allocation.ID == "null" {
					continue
				}

				expiration := allocation.ExpirationDate
				if expiration == 0 {
					expiration = allocation.Expiration
				}

				if expiration <= 0 {
					continue
				}

				shouldRenew := expiration < cutoff
				if shouldRenew != tt.expectRenew {
					t.Errorf("Allocation %s shouldRenew = %v, expected %v", allocation.ID, shouldRenew, tt.expectRenew)
				}
			}
		})
	}
}

// Test NewZS3Monitor with mock environment
func TestNewZS3Monitor(t *testing.T) {
	tempDir := createTempDir(t)
	defer cleanupTempDir(t, tempDir)

	// Create test files
	metadataFile := createTempFile(t, tempDir, "metadata.env", `BLOBBER_ID_LIST="blobber1,blobber2"
CLUSTER_ID="test-cluster"
USER_ID="test-user"
CLIENT_ID="test-client-id"`)

	createTempFile(t, tempDir, "wallet.json", `{
		"client_id": "zs3-client-123",
		"client_key": "test-key",
		"keys": []
	}`)

	logFile := filepath.Join(tempDir, "test.log")

	// Override global variables for this test
	originalMetadata := METADATA_PATH
	originalZCN := ZCN_DIR
	originalLog := LOG_FILE

	METADATA_PATH = metadataFile
	ZCN_DIR = tempDir
	LOG_FILE = logFile

	defer func() {
		METADATA_PATH = originalMetadata
		ZCN_DIR = originalZCN
		LOG_FILE = originalLog
	}()

	monitor, err := NewZS3Monitor()
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}

	if monitor == nil {
		t.Fatal("Monitor is nil")
	}

	if monitor.metadata == nil {
		t.Error("Metadata should be loaded")
	}

	if monitor.zs3ClientID != "zs3-client-123" {
		t.Errorf("ZS3 Client ID = %s, expected zs3-client-123", monitor.zs3ClientID)
	}

	if len(monitor.blobberIDs) != 2 {
		t.Errorf("Expected 2 blobber IDs, got %d", len(monitor.blobberIDs))
	}

	expectedBlobbers := []string{"blobber1", "blobber2"}
	for i, expectedID := range expectedBlobbers {
		if i < len(monitor.blobberIDs) && monitor.blobberIDs[i] != expectedID {
			t.Errorf("Blobber ID[%d] = %s, expected %s", i, monitor.blobberIDs[i], expectedID)
		}
	}

	// Verify log file was created
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Error("Log file should be created")
	}
}

// Test error handling in NewZS3Monitor
func TestNewZS3MonitorErrorHandling(t *testing.T) {
	tempDir := createTempDir(t)
	defer cleanupTempDir(t, tempDir)

	// Test with invalid log file path (read-only directory)
	readOnlyDir := filepath.Join(tempDir, "readonly")
	err := os.Mkdir(readOnlyDir, 0444) // Read-only permissions
	if err != nil {
		t.Fatalf("Failed to create read-only dir: %v", err)
	}

	originalLog := LOG_FILE
	LOG_FILE = filepath.Join(readOnlyDir, "test.log")
	defer func() { LOG_FILE = originalLog }()

	_, err = NewZS3Monitor()
	if err == nil {
		t.Error("Expected error for invalid log file path")
	}

	if !strings.Contains(err.Error(), "failed to open log file") {
		t.Errorf("Expected log file error, got: %v", err)
	}
}

// Integration test for the complete flow
func TestZS3MonitorIntegration(t *testing.T) {
	tempDir := createTempDir(t)
	defer cleanupTempDir(t, tempDir)

	// Create test environment
	metadataFile := createTempFile(t, tempDir, "metadata.env", `BLOBBER_ID_LIST="blobber1,blobber2"
CLUSTER_ID="test-cluster"
USER_ID="test-user"
CLIENT_ID="test-client-id"`)

	createTempFile(t, tempDir, "wallet.json", `{
		"client_id": "zs3-client-123",
		"client_key": "test-key",
		"keys": []
	}`)

	logFile := filepath.Join(tempDir, "integration.log")

	// Override global variables
	originalMetadata := METADATA_PATH
	originalZCN := ZCN_DIR
	originalLog := LOG_FILE
	originalBaseline := BASELINE_FILE

	METADATA_PATH = metadataFile
	ZCN_DIR = tempDir
	LOG_FILE = logFile
	BASELINE_FILE = filepath.Join(tempDir, "baseline.json")

	defer func() {
		METADATA_PATH = originalMetadata
		ZCN_DIR = originalZCN
		LOG_FILE = originalLog
		BASELINE_FILE = originalBaseline
	}()

	// Create monitor
	monitor, err := NewZS3Monitor()
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}

	// Test that monitor is properly initialized
	if monitor.metadata == nil {
		t.Error("Metadata should be loaded")
	}

	if monitor.zs3ClientID == "" {
		t.Error("ZS3 Client ID should be set")
	}

	if monitor.logger == nil {
		t.Error("Logger should be initialized")
	}

	// Test baseline balance operations
	testBalances := BalancesMap{
		monitor.zs3ClientID: {Balance: 1000},
		"blobber1":          {Balance: 500},
		"blobber2":          {Balance: 300},
	}

	err = monitor.saveBaselineBalances(testBalances)
	if err != nil {
		t.Fatalf("Failed to save baseline balances: %v", err)
	}

	loadedBalances, err := monitor.loadBaselineBalances()
	if err != nil {
		t.Fatalf("Failed to load baseline balances: %v", err)
	}

	if len(loadedBalances) != len(testBalances) {
		t.Errorf("Loaded balances length = %d, expected %d", len(loadedBalances), len(testBalances))
	}

	// Test topup calculation
	topupNeeded := monitor.calcTopupNeeded(200, 500) // 200 current, 500 baseline
	expectedTopup := 50                              // 50% of 500 = 250, 250 - 200 = 50
	if topupNeeded != expectedTopup {
		t.Errorf("Topup needed = %d, expected %d", topupNeeded, expectedTopup)
	}

	// Verify log file content
	logContent, err := ioutil.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logStr := string(logContent)
	if !strings.Contains(logStr, "Monitoring: ZS3=") {
		t.Error("Log should contain monitoring information")
	}
}

// Benchmark tests
func BenchmarkCalcTopupNeeded(b *testing.B) {
	monitor := &ZS3Monitor{}

	for i := 0; i < b.N; i++ {
		monitor.calcTopupNeeded(200, 500)
	}
}

func BenchmarkLoadMetadata(b *testing.B) {
	tempDir := createTempDir(&testing.T{})
	defer cleanupTempDir(&testing.T{}, tempDir)

	metadataFile := createTempFile(&testing.T{}, tempDir, "metadata.env", `BLOBBER_ID_LIST="blobber1,blobber2"
	CLUSTER_ID="test-cluster"
	USER_ID="test-user"
	CLIENT_ID="test-client-id"`)

	originalMetadata := METADATA_PATH
	METADATA_PATH = metadataFile
	defer func() { METADATA_PATH = originalMetadata }()

	monitor := &ZS3Monitor{
		logger: log.New(os.Stdout, "", 0),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		monitor.loadMetadata()
	}
}
