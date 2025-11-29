package ebs

import (
	"bufio"
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	ZBOX_DIR = "/opt/0chain/zboxcli"
	ZCN_DIR  = "/home/n7/.zcn"
	ZWALLET  = "/opt/0chain/zwalletcli/zwallet"
	ZBOX     = "/opt/0chain/zboxcli/zbox"
	METADATA = "/var/lib/zs3/metadata.env"
)

// metadata keys expected (script sources whatever is present in the file)
var metadata = map[string]string{}

func handle_ebs_increase(oldEbsStr string, newEbsStr string) {

	oldEbs, err := strconv.ParseFloat(oldEbsStr, 64)
	if err != nil {
		exitError(fmt.Sprintf("invalid OLD_EBS_SIZE_GB: %v", err))
	}
	newEbs, err := strconv.ParseFloat(newEbsStr, 64)
	if err != nil {
		exitError(fmt.Sprintf("invalid NEW_EBS_SIZE_GB: %v", err))
	}

	// calculate the increase in size
	increaseSizeGB := newEbs - oldEbs
	if !(increaseSizeGB > 0) {
		fmt.Println("New EBS size must be greater than old EBS size.")
		os.Exit(1)
	}

	// per blobber increase (script uses 3 blobbers)
	increaseSizePerBlobberGB := increaseSizeGB / 3.0

	// load metadata
	if _, err := os.Stat(METADATA); err == nil {
		if err := loadMetadata(METADATA); err != nil {
			exitError(fmt.Sprintf("failed to load metadata: %v", err))
		}
	} else {
		fmt.Printf("Warning: Metadata file %s not found!\n", METADATA)
		os.Exit(1)
	}

	// get BLOBBER_ID_LIST from metadata
	blobberList := metadata["BLOBBER_ID_LIST"]
	if blobberList == "" {
		exitError("BLOBBER_ID_LIST not found in metadata")
	}
	// split by comma -> array
	blobberIDs := splitCommaList(blobberList)

	fmt.Println("[INFO] Staking additional storage on blobbers...")

	// function stake_on_blobber equivalent (we'll call below)
	// For each blobber in array:
	for _, blobberID := range blobberIDs {
		blobberID = strings.TrimSpace(blobberID)
		if blobberID == "" {
			continue
		}
		fmt.Printf("[INFO] Staking on blobber %s for additional %.6f GB...\n", blobberID, increaseSizePerBlobberGB)

		// STAKE_FLOAT = INCREASE_SIZE_PER_BLOBBER_GB * 1024 * MIN_WRITE_PRICE
		minWritePriceStr := metadata["MIN_WRITE_PRICE"]
		if minWritePriceStr == "" {
			exitError("MIN_WRITE_PRICE not found in metadata")
		}
		minWritePrice, err := strconv.ParseFloat(minWritePriceStr, 64)
		if err != nil {
			exitError(fmt.Sprintf("invalid MIN_WRITE_PRICE in metadata: %v", err))
		}

		stakeFloat := increaseSizePerBlobberGB * minWritePrice
		if math.IsNaN(stakeFloat) || math.IsInf(stakeFloat, 0) {
			stakeFloat = 1.0
		}

		log.Printf("stakeFloat: %f", stakeFloat)
		log.Printf("increaseSizePerBlobberGB: %f", increaseSizePerBlobberGB)
		log.Printf("minWritePrice: %f", minWritePrice)

		// STAKE_AMOUNT = awk ceil-ish: if v<1 print 1 else int(v+0.999999)
		stakeAmount := ceilWithMin(stakeFloat, 1.0)
		fmt.Printf("[INFO] Calculated stake amount: %f tokens\n", stakeAmount)

		if !stakeOnBlobber(blobberID, stakeAmount) {
			fmt.Printf("[ERROR] Staking failed for blobber %s. Exiting.\n", blobberID)
			os.Exit(1)
		}
	}

	fmt.Println("[INFO] Successfully staked additional storage on all blobbers.")

	// ----------------------- EXTEND ALLOCATION -----------------------
	fmt.Println("[INFO] Extending allocation size...")

	// LOCK_FLOAT = 3 * MIN_WRITE_PRICE * INCREASE_SIZE_GB * 1024
	minWritePriceStr := metadata["MIN_WRITE_PRICE"]
	if minWritePriceStr == "" {
		exitError("MIN_WRITE_PRICE not found in metadata")
	}
	minWritePrice, err := strconv.ParseFloat(minWritePriceStr, 64)
	if err != nil {
		exitError(fmt.Sprintf("invalid MIN_WRITE_PRICE in metadata: %v", err))
	}
	lockFloat := 3.0 * minWritePrice * increaseSizeGB
	if math.IsNaN(lockFloat) || math.IsInf(lockFloat, 0) {
		lockFloat = 1.0
	}
	lockTokens := ceilWithMin(lockFloat, 1.0)
	fmt.Printf("[INFO] Calculated lock tokens for allocation update: %d tokens\n", lockTokens)

	log.Printf("lockFloat: %f", lockFloat)
	log.Printf("minWritePrice: %f", minWritePrice)
	log.Printf("increaseSizeGB: %f", increaseSizeGB)

	// call update_allocation_size with same retry behavior
	if !updateAllocationSize(increaseSizeGB, lockTokens) {
		fmt.Println("[ERROR] Allocation size update failed. Exiting.")
		os.Exit(1)
	}

	fmt.Printf("[INFO] Successfully extended allocation size to %.0f GB.\n", newEbs)
}

// ---------------- helper functions ----------------

func splitCommaList(s string) []string {
	// mimic IFS=',' read -r -a array <<< "$BLOBBER_ID_LIST"
	parts := strings.Split(s, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func ceilWithMin(v float64, minVal float64) float64 {
	if v < float64(minVal) {
		return minVal
	}
	// exact replicate: int(v + 0.999999)
	return math.Floor(v + 0.999999)
}

func stakeOnBlobber(blobberID string, tokens float64) bool {
	// stake_on_blobber() { ... tries 5 times; uses ./zbox sp-lock --wallet "$ZCN_DIR/wallet.json" --blobber_id "$blobber_id" --tokens 9; }
	// NOTE: The original script's stake_on_blobber had tokens param but used hardcoded 9.
	// To keep logic EXACTLY the same, use tokens "9" as the script does.
	for attempt := 1; attempt <= 5; attempt++ {
		fmt.Printf("[INFO] Staking attempt %d for blobber %s...\n", attempt, blobberID)

		// run ./zbox sp-lock --wallet "$ZCN_DIR/wallet.json" --blobber_id "$blobber_id" --tokens 9
		cmd := exec.Command("./zbox", "sp-lock",
			"--wallet", filepath.Join(ZCN_DIR, "wallet.json"),
			"--blobber_id", blobberID,
			"--tokens", fmt.Sprintf("%f", tokens),
		)
		cmd.Dir = ZBOX_DIR
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err == nil {
			fmt.Printf("[INFO] Successfully staked via zbox sp-lock on %s (attempt %d)\n", blobberID, attempt)
			return true
		}

		if attempt < 5 {
			waitTime := attempt * 10
			fmt.Printf("[WARN] Staking attempt %d failed for %s, waiting %ds before retry...\n", attempt, blobberID, waitTime)
			time.Sleep(time.Duration(waitTime) * time.Second)
		}
	}

	fmt.Printf("[ERROR] All staking attempts failed for blobber %s\n", blobberID)
	return false
}

func updateAllocationSize(newSizeGB float64, lockTokens float64) bool {
	// update_allocation_size() { ... tries up to 5 times; runs zbox updateallocation --wallet "$ZCN_DIR/wallet.json" --allocation "$ALLOCATION_ID" --size "$((new_size_gb * 1024 * 1024 * 1024))" --lock "$lock_tokens"; }
	allocID := metadata["ALLOCATION_ID"]
	if allocID == "" {
		exitError("ALLOCATION_ID not found in metadata")
	}

	// size in bytes (script used integer arithmetic). compute via floats then cast to integer bytes.
	sizeBytesFloat := newSizeGB * 1024.0 * 1024.0 * 1024.0
	// Ensure non-NaN and non-infinite
	if math.IsNaN(sizeBytesFloat) || math.IsInf(sizeBytesFloat, 0) {
		sizeBytesFloat = 0.0
	}
	// convert to integer string, matching shell integer arithmetic effect
	sizeBytesInt := int64(sizeBytesFloat)

	for attempt := 1; attempt <= 5; attempt++ {
		fmt.Printf("[INFO] Update allocation attempt %d...\n", attempt)
		cmd := exec.Command("./zbox", "updateallocation",
			"--wallet", filepath.Join(ZCN_DIR, "wallet.json"),
			"--allocation", allocID,
			"--size", fmt.Sprintf("%d", sizeBytesInt),
			"--lock", fmt.Sprintf("%f", lockTokens),
		)
		cmd.Dir = ZBOX_DIR
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err == nil {
			fmt.Printf("[INFO] Successfully updated allocation size to %.6f GB (attempt %d)\n", newSizeGB, attempt)
			return true
		}

		if attempt < 5 {
			waitTime := attempt * 10
			fmt.Printf("[WARN] Update allocation attempt %d failed, waiting %ds before retry...\n", attempt, waitTime)
			time.Sleep(time.Duration(waitTime) * time.Second)
		}
	}

	fmt.Println("[ERROR] All update allocation attempts failed")
	return false
}

func loadMetadata(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// split at first '='
		idx := strings.Index(line, "=")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		// remove surrounding quotes if present
		if len(val) >= 2 && ((val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'')) {
			val = val[1 : len(val)-1]
		}
		metadata[key] = val
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

func exitError(msg string) {
	fmt.Println("[ERROR]", msg)
	os.Exit(1)
}
