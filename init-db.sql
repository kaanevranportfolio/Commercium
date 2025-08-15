#!/bin/bash
set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    -- Create development database (already created as POSTGRES_DB)
    \echo 'Development database created: $POSTGRES_DB'
    
    -- Create test database for integration tests
    CREATE DATABASE commercium_test_db;
    GRANT ALL PRIVILEGES ON DATABASE commercium_test_db TO $POSTGRES_USER;
    \echo 'Test database created: commercium_test_db'
    
    -- Connect to test database and set up basic permissions
    \c commercium_test_db
    GRANT ALL ON SCHEMA public TO $POSTGRES_USER;
    
    -- Connect back to main database and set up permissions  
    \c $POSTGRES_DB
    GRANT ALL ON SCHEMA public TO $POSTGRES_USER;
    
    \echo 'Database setup completed successfully'
EOSQL
