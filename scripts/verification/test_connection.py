#!/usr/bin/env python3
"""
Test Clickhouse database connectivity through SSH tunnel.

Usage:
    1. Establish SSH tunnel: ssh -i ~/.ssh/key -L 8123:localhost:8123 user@host
    2. Update credentials in this script
    3. Run: python test_connection.py
"""

import sys
import clickhouse_connect

def test_connection():
    """Test connection to Clickhouse database."""
    
    print("Testing Clickhouse connection...")
    print("Host: localhost (via SSH tunnel)")
    print("HTTP Port: 8123")
    print()
    
    # Update these values to match your environment
    DB_HOST = 'localhost'
    DB_PORT = 8123
    DB_USER = 'your_username'
    DB_PASSWORD = 'your_password'
    
    try:
        client = clickhouse_connect.get_client(
            host=DB_HOST,
            port=DB_PORT,
            username=DB_USER,
            password=DB_PASSWORD
        )
        
        print("Testing connection...")
        result = client.query('SELECT version()')
        
        # Verificar si hay resultados
        if result and result.result_rows:
            version = result.result_rows[0][0]
        else:
            version = "Unknown"
        
        print(f"Connection successful!")
        print(f"   Clickhouse version: {version}")
        print()
        
        # Test query
        print("Executing test query...")
        result = client.query('SELECT currentDatabase()')
        database = result.result_rows[0][0] if result and result.result_rows else "Unknown"
        print(f"Current database: {database}")
        print()
        
        # List tables
        print("Listing available tables...")
        result = client.query('SHOW TABLES')
        tables = [row[0] for row in result.result_rows]
        
        print(f"Found {len(tables)} tables:")
        for table in tables[:10]:
            print(f"   - {table}")
        if len(tables) > 10:
            print(f"   ... and {len(tables) - 10} more")
        print()
        
        # Verify critical tables
        print("Verifying critical tables for EL rewards...")
        critical_tables = [
            't_validator_rewards_summary',
            't_block_rewards', 
            't_block_metrics'
        ]
        
        for table in critical_tables:
            if table in tables:
                print(f"   [OK] {table}")
            else:
                print(f"   [MISSING] {table}")
        print()
        
        return True
            
    except Exception as e:
        import traceback
        print(f"Detailed error: {e}")
        print(f"Type: {type(e).__name__}")
        traceback.print_exc()
        
        error_msg = str(e).lower()
        if 'connection' in error_msg or 'refused' in error_msg:
            print("Error: Cannot connect to localhost:8123")
            print()
            print("Did you establish the SSH tunnel?")
            print("   Command: ssh -L 8123:localhost:8123 user@host")
            print()
            print("   Then run this script again.")
            return False
        else:
            print(f"Unexpected error: {e}")
            return False

def main():
    print("="*80)
    print("CONNECTION TEST TO CLICKHOUSE - Via SSH Tunnel")
    print("="*80)
    
    if test_connection():
        print("="*80)
        print("CONNECTION SUCCESSFUL")
        print("="*80)
        print()
        print("You can proceed with:")
        print("  python verify_apr_calculation.py")
        print()
    else:
        print("="*80)
        print("CONNECTION FAILED")
        print("="*80)
        print()
        print("Connection steps:")
        print("  1. Open a terminal and run:")
        print("     ssh -L 8123:localhost:8123 user@host")
        print()
        print("  2. Keep that terminal open (tunnel must stay active)")
        print()
        print("  3. In another terminal run:")
        print("     python test_connection.py")
        print()

if __name__ == "__main__":
    main()
