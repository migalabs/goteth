#!/usr/bin/env python3
"""
List all available Clickhouse databases and search for goteth tables.

Usage:
    1. Establish SSH tunnel: ssh -i ~/.ssh/key -L 8123:localhost:8123 user@host
    2. Update credentials in this script
    3. Run: python list_databases.py
"""

import clickhouse_connect

# Update these values to match your environment
DB_CONFIG = {
    'host': 'localhost',
    'port': 8123,
    'username': 'your_username',
    'password': 'your_password'
}

client = clickhouse_connect.get_client(**DB_CONFIG)

print("Listing all databases:")
print("="*60)

result = client.query('SHOW DATABASES')
databases = [row[0] for row in result.result_rows]

for db in databases:
    print(f"- {db}")
    
print("\n" + "="*60)
print(f"Total: {len(databases)} databases")

# Search for goteth tables in each database
print("\n\nSearching for goteth tables...")
print("="*60)

target_tables = ['t_validator_rewards_summary', 't_block_rewards', 't_block_metrics']

for db in databases:
    if db in ['system', 'information_schema', 'INFORMATION_SCHEMA']:
        continue
    
    try:
        result = client.query(f'SHOW TABLES FROM {db}')
        tables = [row[0] for row in result.result_rows]
        
        found = [t for t in target_tables if t in tables]
        
        if found:
            print(f"\nFound in database: {db}")
            print(f"   Tables: {', '.join(found)}")
            print(f"   Total tables: {len(tables)}")
    except Exception as e:
        print(f"\nCould not access {db}: {e}")

print("\n" + "="*60)
