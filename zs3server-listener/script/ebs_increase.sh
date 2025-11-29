#!/bin/bash
set -euo pipefail
# write logs 
exec >> /var/log/ebs_increase.log 2>&1

OLD_EBS_SIZE_GB=$1
NEW_EBS_SIZE_GB=$2

# calculate the increase in size (float)
INCREASE_SIZE_GB=$(echo "$NEW_EBS_SIZE_GB - $OLD_EBS_SIZE_GB" | bc -l 2>/dev/null || echo "0")

# Check if increase is greater than 0 (float comparison)
if [ "$(echo "$INCREASE_SIZE_GB <= 0" | bc -l 2>/dev/null || echo "1")" = "1" ]; then
  echo "New EBS size must be greater than old EBS size."
  exit 1
fi

# we have 3 blobbers of 2 data shards, and 1 parity shard
# per blobber increase size calculation (float)
INCREASE_SIZE_PER_BLOBBER_GB=$(echo "scale=6; $INCREASE_SIZE_GB / 3.0" | bc -l 2>/dev/null || echo "0")


# ---------------------- SET VARIABLES -----------------------
ZBOX_DIR="/opt/0chain/zboxcli"
ZCN_DIR="/home/n7/.zcn"
ZWALLET="/opt/0chain/zwalletcli/zwallet"
ZBOX="/opt/0chain/zboxcli/zbox"
METADATA="/var/lib/zs3/metadata.env"

# ---------------------- LOAD METADATA -----------------------
# metadata file format example:
# echo "BLOBBER_ID_LIST=$BLOBBER_ID_LIST" > /var/lib/zs3/metadata.env
# echo "CLIENT_ID=$CLIENT_ID" >> /var/lib/zs3/metadata.env
# echo "AUTH_TICKETS_FILE=$AUTH_TICKETS_FILE" >> /var/lib/zs3/metadata.env
# echo "MIN_WRITE_PRICE=$MIN_WRITE_PRICE" >> /var/lib/zs3/metadata.env
# echo "ALLOCATION_ID=$ALLOCATION_ID" >> /var/lib/zs3/metadata.env

if [ -f "$METADATA" ]; then
  # shellcheck disable=SC1090
  source "$METADATA"
else
  echo "Warning: Metadata file $METADATA not found!"
  exit 1
fi

# ----------------------- STAKE ON BLOBBERS -----------------------
echo "[INFO] Staking additional storage on blobbers..."

stake_on_blobber() {
    local blobber_id="$1"
    local tokens="$2"
    
    cd "$ZBOX_DIR"
    
    for attempt in 1 2 3 4 5; do
        echo "[INFO] Staking attempt $attempt for blobber $blobber_id..."
        if ./zbox sp-lock --wallet "$ZCN_DIR/wallet.json" --blobber_id "$blobber_id" --tokens "$tokens"; then
            echo "[INFO] Successfully staked via zbox sp-lock on $blobber_id (attempt $attempt)"
            return 0
        fi
        
        if [ $attempt -lt 5 ]; then
            wait_time=$((attempt * 10))
            echo "[WARN] Staking attempt $attempt failed for $blobber_id, waiting $${wait_time}s before retry..."
            sleep $wait_time
        fi
    done
    
    echo "[ERROR] All staking attempts failed for blobber $blobber_id"
    return 1
}

#STAKE_FLOAT=$(echo "65536 * $MIN_WRITE_PRICE" | bc -l 2>/dev/null || echo "1")
#STAKE_AMOUNT=$(awk -v v="$STAKE_FLOAT" 'BEGIN{ if (v<1) print 1; else printf("%d", int(v+0.999999)); }')
#SENTINEL_STAKE="/var/log/zs3_staked_$blobber_id"

# we need to stake for each blobber with increased size times the min write price
IFS=',' read -r -a BLOBBER_IDS_ARRAY <<< "$BLOBBER_ID_LIST"
for BLOBBER_ID in "${BLOBBER_IDS_ARRAY[@]}"; do
    echo "[INFO] Staking on blobber $BLOBBER_ID for additional $INCREASE_SIZE_PER_BLOBBER_GB GB..."
    # calculate stake amount
    # mZCN ZCN
    echo "min write price: $MIN_WRITE_PRICE"
    echo "increase size per blobber: $INCREASE_SIZE_PER_BLOBBER_GB"
    STAKE_FLOAT=$(echo "$INCREASE_SIZE_PER_BLOBBER_GB * $MIN_WRITE_PRICE" | bc -l 2>/dev/null || echo "1")
    echo "[INFO] Calculated stake float: $STAKE_FLOAT"
    # Check for NaN or Inf (matching main.go logic)
    if [ "$(echo "$STAKE_FLOAT" | grep -E '^(nan|inf|NaN|Inf)' || echo '')" != "" ]; then
        STAKE_FLOAT="1.0"
    fi
    # Match main.go ceilWithMin: if v<1 print 1, else floor(v+0.999999) as float
    STAKE_AMOUNT=$(awk -v v="$STAKE_FLOAT" 'BEGIN{ if (v<1) print "1.0"; else printf("%.0f", int(v+0.999999)); }')
    echo "[INFO] Calculated stake amount: $STAKE_AMOUNT tokens"
    if ! stake_on_blobber "$BLOBBER_ID" "$STAKE_AMOUNT"; then
        echo "[ERROR] Staking failed for blobber $BLOBBER_ID. Exiting."
        exit 1
    fi
done

echo "[INFO] Successfully staked additional storage on all blobbers."

# ----------------------- EXTEND ALLOCATION -----------------------
# LOCK_FLOAT=$(echo "$blobber_count * $MIN_WRITE_PRICE * $ebs_volume_size" | bc -l 2>/dev/null || echo "`10")
# LOCK_AMOUNT=$(awk -v v="$LOCK_FLOAT" 'BEGIN{ if (v<1) print 1; else printf("%d", int(v+0.999999)); }')
# ALLOCATION_SIZE=$((increase_size_gb * 1024 * 1024 * 1024)) # in bytes
# update the allocation size using zboxcli 
echo "[INFO] Extending allocation size..."

# inputs new size in gb to convert into bytes and lock tokens
update_allocation_size() {
    local new_size_gb="$1"
    local lock_tokens="$2"
    
    cd "$ZBOX_DIR"
    
    # Calculate size in bytes from float GB (matching main.go: newSizeGB * 1024.0 * 1024.0 * 1024.0)
    # Then convert to integer bytes for --size parameter
    SIZE_BYTES_FLOAT=$(echo "$new_size_gb * 1024.0 * 1024.0 * 1024.0" | bc -l 2>/dev/null || echo "0")
    # Check for NaN or Inf (matching main.go logic)
    if [ "$(echo "$SIZE_BYTES_FLOAT" | grep -E '^(nan|inf|NaN|Inf)' || echo '')" != "" ]; then
        SIZE_BYTES_FLOAT="0"
    fi
    # Convert to integer bytes (matching main.go: int64(sizeBytesFloat))
    SIZE_BYTES=$(echo "$SIZE_BYTES_FLOAT" | awk '{printf("%d", $1)}')
    
    for attempt in 1 2 3 4 5; do
        echo "[INFO] Update allocation attempt $attempt..."
        if ./zbox updateallocation --wallet "$ZCN_DIR/wallet.json" --allocation "$ALLOCATION_ID" --size "$SIZE_BYTES" --lock "$lock_tokens"; then
            echo "[INFO] Successfully updated allocation size to $new_size_gb GB (attempt $attempt)"
            return 0
        fi
        
        if [ $attempt -lt 5 ]; then
            wait_time=$((attempt * 10))
            echo "[WARN] Update allocation attempt $attempt failed, waiting $${wait_time}s before retry..."
            sleep $wait_time
        fi
    done
    
    echo "[ERROR] All update allocation attempts failed"
    return 1
}

# calculate lock tokens for the increase 
# the allocation size in update should be INCREASE_SIZE_GB 
# number of blobber * min write price * increase size in gb (matching main.go: 3.0 * minWritePrice * increaseSizeGB)

LOCK_FLOAT=$(echo "3.0 * $MIN_WRITE_PRICE * $INCREASE_SIZE_PER_BLOBBER_GB" | bc -l 2>/dev/null || echo "1")
# Check for NaN or Inf (matching main.go logic)
if [ "$(echo "$LOCK_FLOAT" | grep -E '^(nan|inf|NaN|Inf)' || echo '')" != "" ]; then
    LOCK_FLOAT="1.0"
fi
# Match main.go ceilWithMin: if v<1 print 1, else floor(v+0.999999) as float
LOCK_TOKENS=$(awk -v v="$LOCK_FLOAT" 'BEGIN{ if (v<1) print "1.0"; else printf("%.0f", int(v+0.999999)); }')
echo "[INFO] Calculated lock tokens for allocation update: $LOCK_TOKENS tokens"

if ! update_allocation_size "$INCREASE_SIZE_GB" "$LOCK_TOKENS"; then
    echo "[ERROR] Allocation size update failed. Exiting."
    exit 1
fi

echo "[INFO] Successfully extended allocation size to $NEW_EBS_SIZE_GB GB."

# ---------------------- RESET MONITORING BASELINE -------------------
# This is critical. We just staked more tokens, so the old
# balance baseline is obsolete. We delete it here.
# The zs3_monitoring.sh script will automatically create a new,
# correct baseline on its next scheduled run.

BASELINE_FILE="/var/lib/zs3/initial_balances.json"

if [ -f "$BASELINE_FILE" ]; then
  echo "Resetting monitoring: Removing old balance baseline file..."
  rm -f "$BASELINE_FILE"
  echo "Baseline file removed. Monitoring will create a new one on its next run."
else
  echo "Monitoring baseline file not found, no reset needed."
fi

echo "=== [$(date -Is)] Increase process and monitor reset complete ==="
