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
        
        print(f"✅ Conexión exitosa!")
        print(f"   Versión Clickhouse: {version}")
        print()
        
        # Test query simple
        print("🔍 Ejecutando query de prueba...")
        result = client.query('SELECT currentDatabase()')
        database = result.result_rows[0][0] if result and result.result_rows else "Unknown"
        print(f"✅ Database actual: {database}")
        print()
        
        # Listar tablas
        print("🔍 Listando tablas disponibles...")
        result = client.query('SHOW TABLES')
        tables = [row[0] for row in result.result_rows]
        
        print(f"✅ Encontradas {len(tables)} tablas:")
        for table in tables[:10]:
            print(f"   - {table}")
        if len(tables) > 10:
            print(f"   ... y {len(tables) - 10} más")
        print()
        
        # Verificar tablas críticas
        print("🔍 Verificando tablas críticas para EL rewards...")
        critical_tables = [
            't_validator_rewards_summary',
            't_block_rewards', 
            't_block_metrics'
        ]
        
        for table in critical_tables:
            if table in tables:
                print(f"   ✅ {table}")
            else:
                print(f"   ❌ {table} - NO ENCONTRADA")
        print()
        
        return True
            
    except Exception as e:
        import traceback
        print(f"❌ Error detallado: {e}")
        print(f"Tipo: {type(e).__name__}")
        traceback.print_exc()
        
        error_msg = str(e).lower()
        if 'connection' in error_msg or 'refused' in error_msg:
            print("❌ Error: No se pudo conectar a localhost:8123")
            print()
            print("🔧 ¿Estableciste el túnel SSH?")
            print("   Comando: ssh -L 8123:localhost:8123 zyrav21@57.129.148.45")
            print()
            print("   Luego ejecuta este script nuevamente.")
            return False
        else:
            print(f"❌ Error inesperado: {e}")
            return False

def main():
    print("="*80)
    print("TEST DE CONEXION A CLICKHOUSE - Via SSH Tunnel")
    print("="*80)
    
    if test_connection():
        print("="*80)
        print("✅ CONEXIÓN EXITOSA")
        print("="*80)
        print()
        print("Puedes proceder con:")
        print("  python verify_el_rewards_issue.py http://zyrav21@localhost:8123/default")
        print()
    else:
        print("="*80)
        print("❌ CONEXIÓN FALLIDA")
        print("="*80)
        print()
        print("Pasos para conectar:")
        print("  1. Abre una terminal y ejecuta:")
        print("     ssh -L 8123:localhost:8123 zyrav21@57.129.148.45")
        print()
        print("  2. Deja esa terminal abierta (el túnel debe estar activo)")
        print()
        print("  3. En otra terminal ejecuta:")
        print("     python test_connection.py")
        print()

if __name__ == "__main__":
    main()
