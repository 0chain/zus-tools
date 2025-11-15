# 🚀 ZS3 Monitoring - Complete Deployment Guide

## 📋 **Quick Start - Complete Deployment**

The `build.sh` script now provides **one-command deployment** that handles everything automatically:

```bash
# Complete deployment (DEFAULT)
./build.sh

# Or explicitly
./build.sh deploy
./build.sh start
```

## 🔄 **What the Complete Deployment Does**

The deployment process runs **7 steps automatically**:

### **Step 1: Build Binary**
```bash
make build
```
- ✅ Compiles the Go binary
- ✅ Creates `build/zs3_monitoring`

### **Step 2: Run Unit Tests**
```bash
make test-go
```
- ✅ Runs all unit tests (`main_test.go`)
- ✅ Ensures code quality and logic correctness
- ✅ **Stops deployment if tests fail**

### **Step 3: Install Binary**
```bash
make install
```
- ✅ Copies binary to `/usr/local/bin/zs3_monitoring`
- ✅ Sets executable permissions

### **Step 4: Create Systemd Service**
```bash
make install-service
```
- ✅ Creates `/etc/systemd/system/zs3-monitoring.service`
- ✅ Configures service to run as `ubuntu` user
- ✅ Sets up logging to systemd journal

### **Step 5: Create Cron Job**
```bash
make install-cron
```
- ✅ Adds cron job: `0 2 * * 0 /usr/local/bin/zs3_monitoring`
- ✅ Runs **weekly on Sundays at 2 AM**

### **Step 6: Start Service**
```bash
sudo systemctl start zs3-monitoring.service
```
- ✅ **Starts the monitoring service immediately**
- ✅ Service runs as a one-shot task

### **Step 7: Check Status**
```bash
sudo systemctl status zs3-monitoring.service
```
- ✅ Verifies service started successfully
- ✅ Shows service status and logs

## 🎯 **Usage Examples**

### **Complete Deployment (Recommended)**
```bash
# Default - runs complete deployment
./build.sh

# Explicit deployment
./build.sh deploy
./build.sh start
```

### **Individual Steps (If Needed)**
```bash
# Build only
./build.sh build

# Test only
./build.sh test

# Install only
./build.sh install

# Create service only
./build.sh service

# Create cron only
./build.sh cron
```

## 🔧 **Service Management**

After deployment, you can manage the service:

```bash
# Check service status
sudo systemctl status zs3-monitoring.service

# View service logs
sudo journalctl -u zs3-monitoring.service -f

# Stop service
sudo systemctl stop zs3-monitoring.service

# Restart service
sudo systemctl restart zs3-monitoring.service

# Disable service (prevent auto-start)
sudo systemctl disable zs3-monitoring.service

# Enable service (allow auto-start)
sudo systemctl enable zs3-monitoring.service
```

## 📅 **Cron Job Management**

The cron job runs **weekly on Sundays at 2 AM**:

```bash
# View current cron jobs
crontab -l

# Edit cron jobs
crontab -e

# Remove the monitoring cron job
crontab -e
# Delete the line: 0 2 * * 0 /usr/local/bin/zs3_monitoring
```

## 🛠️ **Troubleshooting**

### **Service Won't Start**
```bash
# Check service status
sudo systemctl status zs3-monitoring.service

# Check logs
sudo journalctl -u zs3-monitoring.service --no-pager -l

# Check if binary exists
ls -la /usr/local/bin/zs3_monitoring

# Test binary manually
/usr/local/bin/zs3_monitoring
```

### **Permission Issues**
```bash
# Check binary permissions
ls -la /usr/local/bin/zs3_monitoring

# Fix permissions if needed
sudo chmod +x /usr/local/bin/zs3_monitoring

# Check service file permissions
ls -la /etc/systemd/system/zs3-monitoring.service
```

### **Configuration Issues**
```bash
# Check if metadata file exists
ls -la /var/lib/zs3/metadata.env

# Check if wallet file exists
ls -la /home/ubuntu/.zcn/wallet.json

# Check if required directories exist
ls -la /var/lib/zs3/
ls -la /home/ubuntu/.zcn/
```

## 📊 **Monitoring the Service**

### **Real-time Logs**
```bash
# Follow logs in real-time
sudo journalctl -u zs3-monitoring.service -f

# View recent logs
sudo journalctl -u zs3-monitoring.service --since "1 hour ago"
```

### **Service Health Check**
```bash
# Check if service is running
sudo systemctl is-active zs3-monitoring.service

# Check if service is enabled
sudo systemctl is-enabled zs3-monitoring.service

# Check service status
sudo systemctl status zs3-monitoring.service
```

## 🎉 **Success Indicators**

After successful deployment, you should see:

1. ✅ **Binary built**: `build/zs3_monitoring` exists
2. ✅ **Tests passed**: Unit tests complete successfully
3. ✅ **Binary installed**: `/usr/local/bin/zs3_monitoring` exists
4. ✅ **Service created**: `/etc/systemd/system/zs3-monitoring.service` exists
5. ✅ **Cron job added**: `crontab -l` shows the monitoring job
6. ✅ **Service started**: `systemctl status` shows active
7. ✅ **Service status**: Shows successful execution

## 🔄 **Re-deployment**

To update the service:

```bash
# Complete re-deployment
./build.sh deploy

# Or step by step
./build.sh build
sudo systemctl stop zs3-monitoring.service
./build.sh install
sudo systemctl start zs3-monitoring.service
```

## 📝 **Notes**

- **Service Type**: `oneshot` - runs once and exits
- **User**: Runs as `ubuntu` user
- **Logging**: Outputs to systemd journal
- **Schedule**: Weekly via cron (Sundays at 2 AM)
- **Dependencies**: Requires `zbox` and `zwallet` CLI tools
- **Configuration**: Uses `/var/lib/zs3/metadata.env` and `/home/ubuntu/.zcn/wallet.json`

