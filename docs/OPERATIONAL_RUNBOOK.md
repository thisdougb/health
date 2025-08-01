# Health Package Operational Runbook

## Purpose

This runbook provides step-by-step procedures for junior developers and operations teams to manage the health package in production environments. It covers backup operations, data recovery, troubleshooting, and monitoring.

## Quick Reference

### Emergency Contacts
- **Development Team**: [Your team contact info]
- **Operations Team**: [Ops team contact info]
- **On-Call Engineer**: [On-call rotation info]

### Critical File Locations
- **Database**: `/data/health.db` (configurable via `HEALTH_DB_PATH`)
- **Backups**: `/data/backups/health/` (configurable via `HEALTH_BACKUP_DIR`)
- **Logs**: Application logs contain health package messages
- **Configuration**: Environment variables (see Configuration section)

---

## Backup Operations

### Understanding Health Package Backups

**Important for Junior Developers**: Health package backups are **event-driven**, not scheduled. This means:
- ✅ Backups happen when your application shuts down gracefully
- ✅ Backups can be triggered manually via API
- ❌ Backups do NOT happen automatically on a schedule
- ❌ Backups are NOT created during normal operation

### Backup File Format
```
/data/backups/health/
├── health_20250801.db    ← Today's backup
├── health_20250731.db    ← Yesterday's backup
├── health_20250730.db    ← 2 days ago
└── ...
```

**Naming Convention**: `health_YYYYMMDD.db`
- One backup file per day maximum
- Multiple shutdowns on same day overwrite the daily backup
- Automatic cleanup after retention period (default: 30 days)

### Manual Backup Creation

#### Method 1: Graceful Application Shutdown (Recommended)
```bash
# Send SIGTERM to application (triggers graceful shutdown and backup)
kill -TERM $(pgrep myapp)

# Or using systemctl
sudo systemctl stop myapp

# Or using Docker
docker stop myapp-container

# Or using Kubernetes
kubectl delete pod myapp-pod-name --grace-period=30
```

**What Happens During Graceful Shutdown**:
1. Application receives shutdown signal
2. Health package `Close()` method is called
3. Backup is created automatically (if `HEALTH_BACKUP_ENABLED=true`)
4. Application exits cleanly

#### Method 2: Manual Backup API (If Implemented)
```bash
# Example curl command to trigger manual backup
# (This requires your application to expose a backup endpoint)
curl -X POST http://localhost:8080/admin/backup \
  -H "Authorization: Bearer YOUR_API_TOKEN"
```

#### Method 3: Direct Database Backup (Emergency Only)
```bash
# CAUTION: Only use when application is stopped
# This creates a backup without the health package

# Stop application first
sudo systemctl stop myapp

# Create manual backup
cp /data/health.db /data/backups/health/health_manual_$(date +%Y%m%d_%H%M%S).db

# Restart application
sudo systemctl start myapp
```

### Verifying Backup Creation

#### Check Backup Directory
```bash
# List recent backups
ls -la /data/backups/health/

# Expected output:
# -rw-r--r-- 1 app app 1234567 Aug  1 10:30 health_20250801.db
# -rw-r--r-- 1 app app 1234321 Jul 31 10:25 health_20250731.db

# Check backup file is not empty
du -h /data/backups/health/health_20250801.db
# Should show file size > 0
```

#### Verify Backup File Integrity
```bash
# Check if backup file is a valid SQLite database
sqlite3 /data/backups/health/health_20250801.db "SELECT COUNT(*) FROM metrics;"

# Should return a number (count of metrics in backup)
# If error "file is not a database", backup is corrupted
```

#### Check Application Logs
```bash
# Look for backup success messages
tail -n 100 /var/log/myapp.log | grep -i backup

# Expected log entries:
# 2025-08-01 10:30:15 INFO: Health backup created: /data/backups/health/health_20250801.db
# 2025-08-01 10:30:15 INFO: Backup cleanup completed, retained 30 days
```

---

## Data Recovery Procedures

### When to Restore from Backup

**Common Scenarios**:
- Database file is corrupted or missing
- Data was accidentally deleted
- Application crashes caused data loss
- Migration or upgrade went wrong
- Need to analyze historical data

### Pre-Recovery Checklist

1. **Stop the Application** (Critical!)
```bash
sudo systemctl stop myapp
# Or: docker stop myapp-container
# Or: kubectl scale deployment myapp --replicas=0
```

2. **Backup Current State** (Even if corrupted)
```bash
# Backup current database (even if broken)
cp /data/health.db /data/health.db.broken.$(date +%Y%m%d_%H%M%S)
```

3. **Identify Recovery Point**
```bash
# List available backups
ls -la /data/backups/health/

# Choose the backup file you want to restore from
# Usually the most recent: health_YYYYMMDD.db
```

### Recovery Procedure

#### Step 1: Stop All Applications Using the Database
```bash
# Stop primary application
sudo systemctl stop myapp

# Stop any monitoring or analytics tools
sudo systemctl stop myapp-analytics

# Verify no processes are using the database
lsof /data/health.db
# Should return empty (no processes using file)
```

#### Step 2: Remove Corrupted Database
```bash
# Move corrupted database to backup location
mv /data/health.db /data/health.db.corrupted.$(date +%Y%m%d_%H%M%S)

# Verify database file is gone
ls -la /data/health.db
# Should show "No such file or directory"
```

#### Step 3: Restore from Backup
```bash
# Copy backup file to active database location
cp /data/backups/health/health_20250801.db /data/health.db

# Verify file was copied correctly
ls -la /data/health.db
# Should show the restored file with recent timestamp

# Verify file integrity
sqlite3 /data/health.db "SELECT COUNT(*) FROM metrics;"
# Should return a count of metrics
```

#### Step 4: Set Correct Permissions
```bash
# Set ownership (replace 'app' with your application user)
chown app:app /data/health.db

# Set permissions (read/write for owner, read for group)
chmod 664 /data/health.db

# Verify permissions
ls -la /data/health.db
# Should show: -rw-rw-r-- 1 app app
```

#### Step 5: Restart Application
```bash
# Start application
sudo systemctl start myapp

# Verify application started successfully
sudo systemctl status myapp

# Check application logs for health package initialization
tail -f /var/log/myapp.log | grep -i health
# Should show successful database connection
```

#### Step 6: Verify Recovery
```bash
# Test health endpoint
curl http://localhost:8080/health/

# Should return valid JSON with metrics
# {"Identity":"myapp","Started":1234567890,"Metrics":{...}}

# Check that new metrics can be added
# (This will happen automatically as application runs)
```

### Partial Recovery (Advanced)

If you need to recover specific data or merge backups:

#### Extract Specific Component Data
```bash
# Connect to backup database
sqlite3 /data/backups/health/health_20250801.db

# List available components
SELECT DISTINCT component FROM metrics;

# Export specific component data
.mode csv
.output webserver_metrics.csv
SELECT * FROM metrics WHERE component = 'webserver';
.quit
```

#### Merge Multiple Backups
```bash
# This is advanced - only for experienced operators
# Creates a new database with data from multiple backups

sqlite3 merged_recovery.db << EOF
-- Attach backup files
ATTACH '/data/backups/health/health_20250801.db' AS backup1;
ATTACH '/data/backups/health/health_20250731.db' AS backup2;

-- Create metrics table
CREATE TABLE metrics AS SELECT * FROM backup1.metrics;

-- Insert non-duplicate data from second backup
INSERT OR IGNORE INTO metrics 
SELECT * FROM backup2.metrics 
WHERE timestamp < (SELECT MIN(timestamp) FROM backup1.metrics);

-- Detach databases
DETACH backup1;
DETACH backup2;
EOF

# Use merged database as recovery source
cp merged_recovery.db /data/health.db
```

---

## Monitoring and Alerts

### Key Metrics to Monitor

#### Disk Space
```bash
# Monitor backup directory disk usage
df -h /data/backups/health/

# Alert if backup directory > 80% full
# Set up monitoring: df /data/backups | awk 'NR==2 {print $5}' | sed 's/%//'
```

#### Backup Creation
```bash
# Check when last backup was created
ls -la /data/backups/health/ | head -2

# Alert if no backup in last 48 hours
find /data/backups/health/ -name "health_*.db" -mtime -2 | wc -l
# Should return > 0
```

#### Database Health
```bash
# Check database file size growth (should be reasonable)
du -h /data/health.db

# Check database integrity
sqlite3 /data/health.db "PRAGMA integrity_check;"
# Should return "ok"
```

### Automated Monitoring Scripts

#### Backup Health Check Script
```bash
#!/bin/bash
# File: /usr/local/bin/health-backup-check.sh

BACKUP_DIR="/data/backups/health"
ALERT_EMAIL="ops@yourcompany.com"
DAYS_OLD=2

# Check if backup directory exists
if [ ! -d "$BACKUP_DIR" ]; then
    echo "ERROR: Backup directory $BACKUP_DIR does not exist" | mail -s "Health Backup Alert" $ALERT_EMAIL
    exit 1
fi

# Check for recent backups
RECENT_BACKUPS=$(find $BACKUP_DIR -name "health_*.db" -mtime -$DAYS_OLD | wc -l)

if [ $RECENT_BACKUPS -eq 0 ]; then
    echo "WARNING: No health backups found in last $DAYS_OLD days" | mail -s "Health Backup Alert" $ALERT_EMAIL
    exit 1
fi

# Check disk space
DISK_USAGE=$(df $BACKUP_DIR | awk 'NR==2 {print $5}' | sed 's/%//')

if [ $DISK_USAGE -gt 80 ]; then
    echo "WARNING: Backup directory is $DISK_USAGE% full" | mail -s "Health Backup Disk Space" $ALERT_EMAIL
fi

echo "Health backup check passed - $RECENT_BACKUPS recent backups found, ${DISK_USAGE}% disk usage"
```

#### Database Health Check Script
```bash
#!/bin/bash
# File: /usr/local/bin/health-db-check.sh

DB_PATH="/data/health.db"
ALERT_EMAIL="ops@yourcompany.com"

# Check if database file exists
if [ ! -f "$DB_PATH" ]; then
    echo "ERROR: Health database $DB_PATH does not exist" | mail -s "Health DB Alert" $ALERT_EMAIL
    exit 1
fi

# Check database integrity
INTEGRITY_CHECK=$(sqlite3 $DB_PATH "PRAGMA integrity_check;" 2>&1)

if [ "$INTEGRITY_CHECK" != "ok" ]; then
    echo "ERROR: Health database integrity check failed: $INTEGRITY_CHECK" | mail -s "Health DB Corruption" $ALERT_EMAIL
    exit 1
fi

# Check database size (alert if > 1GB)
DB_SIZE_MB=$(du -m $DB_PATH | cut -f1)

if [ $DB_SIZE_MB -gt 1024 ]; then
    echo "WARNING: Health database is ${DB_SIZE_MB}MB (>1GB)" | mail -s "Health DB Size Warning" $ALERT_EMAIL
fi

echo "Health database check passed - integrity ok, size ${DB_SIZE_MB}MB"
```

#### Cron Schedule
```bash
# Add to crontab: crontab -e

# Check backups every 6 hours
0 */6 * * * /usr/local/bin/health-backup-check.sh

# Check database integrity daily at 2 AM
0 2 * * * /usr/local/bin/health-db-check.sh
```

---

## Troubleshooting Guide

### Common Issues and Solutions

#### Issue 1: No Backups Being Created

**Symptoms**:
- Backup directory is empty
- No backup files after application restarts

**Diagnosis**:
```bash
# Check environment variables
env | grep HEALTH

# Should show:
# HEALTH_BACKUP_ENABLED=true
# HEALTH_BACKUP_DIR=/data/backups/health
```

**Solutions**:
1. **Enable Backups**:
```bash
export HEALTH_BACKUP_ENABLED=true
systemctl restart myapp
```

2. **Check Directory Permissions**:
```bash
# Ensure backup directory exists and is writable
mkdir -p /data/backups/health
chown app:app /data/backups/health
chmod 755 /data/backups/health
```

3. **Check Application Logs**:
```bash
tail -f /var/log/myapp.log | grep -i backup
# Look for error messages about backup creation
```

#### Issue 2: Database Corruption

**Symptoms**:
- Application won't start
- SQLite errors in logs
- Backup integrity check fails

**Immediate Actions**:
```bash
# 1. Stop application to prevent further corruption
sudo systemctl stop myapp

# 2. Check database integrity
sqlite3 /data/health.db "PRAGMA integrity_check;"

# 3. If corrupted, restore from backup (see Recovery Procedures above)
```

**Prevention**:
- Ensure proper application shutdown procedures
- Monitor disk space to prevent partial writes
- Use graceful shutdown signals (SIGTERM, not SIGKILL)

#### Issue 3: Backup Directory Full

**Symptoms**:
- Disk space alerts
- New backups failing to create
- Application performance issues

**Immediate Actions**:
```bash
# 1. Check disk usage
df -h /data/backups/health/

# 2. List backup files by age
ls -la /data/backups/health/ | sort -k6,7

# 3. Remove old backups manually (if needed)
find /data/backups/health/ -name "health_*.db" -mtime +60 -delete

# 4. Verify space is freed
df -h /data/backups/health/
```

**Long-term Solutions**:
- Reduce `HEALTH_BACKUP_RETENTION_DAYS`
- Move backups to larger disk
- Implement backup compression
- Set up automated cleanup monitoring

#### Issue 4: Application Won't Start After Recovery

**Symptoms**:
- Application fails to start after database restore
- Health package initialization errors

**Diagnosis Steps**:
```bash
# 1. Check database file permissions
ls -la /data/health.db

# 2. Check database integrity
sqlite3 /data/health.db "PRAGMA integrity_check;"

# 3. Check application logs
tail -f /var/log/myapp.log | grep -i health

# 4. Test database connectivity
sqlite3 /data/health.db "SELECT COUNT(*) FROM sqlite_master;"
```

**Solutions**:
1. **Fix Permissions**:
```bash
chown app:app /data/health.db
chmod 664 /data/health.db
```

2. **Verify Database Schema**:
```bash
# Check table structure
sqlite3 /data/health.db ".schema metrics"

# Should show proper table structure
```

3. **Clear Cache and Restart**:
```bash
# Remove any temporary files
rm -f /tmp/health_*

# Restart with clean state
systemctl restart myapp
```

### Emergency Recovery Procedures

#### Complete Data Loss Scenario

If both primary database and all backups are lost:

1. **Accept Data Loss**:
```bash
# Remove corrupted database
rm -f /data/health.db

# Start application (will create new empty database)
systemctl start myapp
```

2. **Verify Clean State**:
```bash
# Check application starts successfully
curl http://localhost:8080/health/

# Should return JSON with empty or minimal metrics
```

3. **Document Incident**:
- Record what data was lost
- Update backup procedures
- Implement additional monitoring

#### Disaster Recovery

For complete system failure:

1. **Restore from System Backup**:
   - Restore entire `/data` directory from system backups
   - Follow normal recovery procedures

2. **Cross-Region Recovery**:
   - If using cloud storage, restore from different region
   - Verify backup integrity before restoration

3. **Contact Development Team**:
   - For complex recovery scenarios
   - For data migration assistance
   - For emergency patches or fixes

---

## Performance Optimization

### Database Maintenance

#### Regular Maintenance Tasks
```bash
# Weekly: Optimize database (run during low-traffic period)
sqlite3 /data/health.db "VACUUM;"

# Monthly: Analyze database statistics
sqlite3 /data/health.db "ANALYZE;"

# Check database size before/after optimization
du -h /data/health.db
```

#### Performance Monitoring
```bash
# Check database query performance
sqlite3 /data/health.db "EXPLAIN QUERY PLAN SELECT * FROM metrics WHERE component = 'webserver';"

# Monitor table sizes
sqlite3 /data/health.db "SELECT COUNT(*) as total_metrics FROM metrics;"
sqlite3 /data/health.db "SELECT component, COUNT(*) FROM metrics GROUP BY component;"
```

### Configuration Tuning

#### Production Optimization
```bash
# Optimize for production workloads
export HEALTH_FLUSH_INTERVAL=60s      # Balance performance vs data safety
export HEALTH_BATCH_SIZE=100          # Larger batches for better performance
export HEALTH_BACKUP_RETENTION_DAYS=30 # Reasonable retention period
```

#### High-Volume Environments
```bash
# For applications with very high metric volume
export HEALTH_FLUSH_INTERVAL=30s      # More frequent flushes
export HEALTH_BATCH_SIZE=500          # Larger batches
```

---

## Contact Information and Escalation

### Support Levels

1. **Level 1 - Junior Developer**:
   - Use this runbook for standard operations
   - Follow documented procedures exactly
   - Escalate if procedures don't resolve issue

2. **Level 2 - Senior Developer**:
   - Handle complex recovery scenarios
   - Modify procedures as needed
   - Make configuration changes

3. **Level 3 - Development Team**:
   - Handle code-related issues
   - Emergency patches
   - Architecture changes

### When to Escalate

**Immediate Escalation**:
- Data corruption with no working backups
- Security incidents involving health data
- Complete system failure
- Unknown errors not covered in this runbook

**Planned Escalation**:
- Performance degradation
- Capacity planning
- Configuration changes
- Process improvements

### Documentation Updates

After resolving any issue:
1. Update this runbook with new procedures
2. Share lessons learned with team
3. Update monitoring and alerting
4. Review backup and recovery procedures