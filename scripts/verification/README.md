# Verification Scripts

This directory contains scripts to verify and demonstrate the missing EL rewards display and underestimated APR issue.

## Prerequisites

```bash
pip install -r requirements.txt
```

## Connection Setup

If connecting to a remote Clickhouse server, establish an SSH tunnel first:

```bash
ssh -i ~/.ssh/your_key -L 8123:localhost:8123 user@host
```

## Scripts

### verify_apr_calculation.py

Main verification script demonstrating both issues: missing EL rewards display and underestimated APR.

**Purpose:** 
- Shows EL rewards (fees + MEV) that are not displayed in dashboards
- Compares APR calculated with CL only vs CL+EL rewards
- Quantifies both the missing rewards percentage and APR underestimation

**Usage:**
```bash
python verify_apr_calculation.py
```

**Configuration:**
Edit `DB_CONFIG` in the script to match your environment:
- `host`: Clickhouse host (localhost if using SSH tunnel)
- `port`: Clickhouse HTTP port (default 8123)
- `username`: Database username
- `password`: Database password
- `database`: Database name (typically `goteth_mainnet`)

**Expected Output:**
- Total staked ETH and validator counts
- Complete rewards breakdown (CL + EL components)
- EL rewards percentage relative to total
- APR comparison (current vs correct)
- Quantification of both issues

### test_connection.py

Basic connectivity test for Clickhouse database.

**Usage:**
```bash
python test_connection.py
```

### list_databases.py

Lists all available databases and their table counts.

**Usage:**
```bash
python list_databases.py
```

### quick_verification.py

Lightweight verification script for quick checks on smaller epoch ranges.

**Usage:**
```bash
python quick_verification.py
```

## Notes

- All scripts assume database credentials are configured in the script files
- For production use, consider using environment variables for credentials
- Ensure sufficient Clickhouse query timeout for large epoch ranges
