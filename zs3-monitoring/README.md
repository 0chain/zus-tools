# ZS3 Monitoring - Go Implementation

This directory contains the Go implementation of the ZS3 monitoring functionality, replacing the complex bash script with a more maintainable and robust Go binary.

## Quick Start

### Prerequisites
- Go 1.21 or later
- Access to `zbox` and `zwallet` CLI tools

### Build and Run

```bash
# Using the build script (recommended)
./build.sh build

# Or using make directly
export PATH=$PATH:/usr/local/go/bin
make build

# Run the binary
./build/zs3_monitoring
```

### Available Commands

```bash
# Build commands
./build.sh build          # Build for local system
./build.sh build linux    # Build for Linux
./build.sh all            # Run all checks (fmt, lint, test, build)

# Development commands
./build.sh fmt            # Format code
./build.sh lint           # Run linter
./build.sh test           # Run tests

# Installation commands
./build.sh install        # Install binary to /usr/local/bin
./build.sh service        # Create systemd service
./build.sh cron           # Create cron job for weekly execution

# Testing
./build.sh mock           # Test with mock data
```

## Features

- **Balance Monitoring**: Monitors ZS3 and blobber wallet balances
- **Auto-Funding**: Automatically funds blobber wallets from ZS3 when balances fall below 50% of baseline
- **Allocation Renewal**: Automatically extends allocations that expire within 30 days
- **Baseline Management**: Creates and maintains baseline balance records
- **Comprehensive Logging**: Detailed logging with timestamps
- **Error Handling**: Robust error handling and recovery

## Configuration

The monitoring service reads configuration from `/var/lib/zs3/metadata.env`:

```bash
BLOBBER_ID_LIST="blobber1,blobber2,blobber3"
CLUSTER_ID="your-cluster-id"
USER_ID="your-user-id"
CLIENT_ID="your-client-id"
AUTH_TICKETS_FILE="/path/to/auth/tickets.json"
```

## File Structure

```
zs3-monitoring/
├── main.go              # Main Go source file
├── go.mod               # Go module definition
├── Makefile             # Build configuration
├── build.sh             # Build script with Go environment setup
└── README.md            # This file
```

## Advantages Over Bash Script

1. **Maintainability**: Structured, type-safe code
2. **Reliability**: Robust error handling and data validation
3. **Performance**: Faster execution and better resource management
4. **Deployment**: Single binary deployment with no external dependencies
5. **Monitoring**: Better integration with monitoring and observability systems

## Migration from Bash Script

1. **Backup Current Script**: `cp /path/to/zs3_monitoring.sh /path/to/zs3_monitoring.sh.backup`
2. **Build Go Binary**: `./build.sh build linux`
3. **Install Binary**: `./build.sh install`
4. **Update Cron/Systemd**: Update your scheduling to use the new binary
5. **Test and Verify**: Run both scripts in parallel for validation

## Logging

The service logs all activities to `/var/log/zs3_monitoring.log` with timestamps:

```
=== [2024-01-15T02:00:00Z] ZS3 monitoring run start ===
Monitoring: ZS3=client123, Blobbers=3
ZS3 balance: 5000
Blobber blobber1 balance: 1000
Blobber blobber2 balance: 800
Blobber blobber3 balance: 1200
ZS3 wallet balance OK
Blobber blobber1 balance OK
Blobber blobber2 below 50% baseline, funding 200 tokens from ZS3
→ Funding 200 tokens from ZS3 to blobber2
Successfully funded 200 tokens to blobber2
Blobber blobber3 balance OK
Checking allocation expiry...
Found 2 allocation(s)
→ Allocation alloc123 expires within 30 days, extending...
Successfully extended allocation alloc123
→ Allocation alloc456 expires in 45 days (OK)
=== [2024-01-15T02:00:15Z] ZS3 monitoring run completed ===
```

## Troubleshooting

### Common Issues

1. **Go Not Found**: Run `export PATH=$PATH:/usr/local/go/bin`
2. **Permission Denied**: Ensure proper permissions on wallet and log files
3. **Missing Dependencies**: Verify `zbox` and `zwallet` are installed
4. **Configuration Issues**: Check metadata file format and paths

### Debug Mode

For debugging, you can run the binary directly and check the logs:

```bash
# Run with verbose output
./build/zs3_monitoring

# Check logs
tail -f /var/log/zs3_monitoring.log
```

## Support

For issues and questions:
1. Check the logs: `/var/log/zs3_monitoring.log`
2. Verify configuration: `/var/lib/zs3/metadata.env`
3. Test dependencies: `zbox --version`, `zwallet --version`
4. Check permissions on wallet and log files


