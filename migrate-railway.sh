#!/bin/bash

# Migration script for Railway PostgreSQL
# Run all migrations in order

set -e

# Railway PostgreSQL connection details
# Update these with your Railway database credentials
DB_HOST="${RAILWAY_DB_HOST:-your-db-host.railway.app}"
DB_PORT="${RAILWAY_DB_PORT:-5432}"
DB_USER="${RAILWAY_DB_USER:-postgres}"
DB_PASSWORD="${RAILWAY_DB_PASSWORD:-your-password}"
DB_NAME="${RAILWAY_DB_NAME:-railway}"

echo "🚀 Running migrations to Railway PostgreSQL"
echo "Host: $DB_HOST"
echo "Database: $DB_NAME"
echo "=========================================="

# Check if psql is installed
if ! command -v psql &> /dev/null; then
    echo "❌ psql is not installed. Installing..."
    echo "Run: sudo apt-get install postgresql-client"
    exit 1
fi

# Export password for psql
export PGPASSWORD="$DB_PASSWORD"

# Function to run a migration file
run_migration() {
    local file=$1
    echo "Running: $file"
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f "$file"
}

# Run migrations in order
echo ""
echo "📝 Running migrations..."
echo ""

run_migration "migrations/000001_create_users.up.sql"
run_migration "migrations/000002_create_tests.up.sql"
run_migration "migrations/000003_create_questions.up.sql"
run_migration "migrations/000004_create_submissions.up.sql"
run_migration "migrations/000005_create_answers.up.sql"
run_migration "migrations/000006_create_reviews.up.sql"

echo ""
echo "✅ All migrations completed successfully!"
echo ""

# Verify tables were created
echo "📊 Verifying tables..."
psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "\dt"

# Cleanup
unset PGPASSWORD

echo ""
echo "🎉 Migration complete!"
