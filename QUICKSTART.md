# Quick Start - Docker Deployment

Panduan singkat untuk menjalankan Assessly dengan Docker.

## Prerequisites

Pastikan sudah running di host machine:
- PostgreSQL 14+ di `localhost:5432`
- Redis 7+ di `localhost:6379`

## Setup Database

```bash
# 1. Create database
sudo -u postgres psql -c "CREATE USER assessly WITH PASSWORD 'your_password';"
sudo -u postgres psql -c "CREATE DATABASE assessly OWNER assessly;"

# 2. Run migrations
cd migrations
migrate -path . -database "postgres://assessly:your_password@localhost:5432/assessly?sslmode=disable" up
```

## Run dengan Docker Compose

```bash
# 1. Clone repo
git clone <repo-url>
cd assessly-be

# 2. Copy dan edit .env
cp .env.example .env
nano .env  # Edit DB_PASSWORD, GROQ_API_KEY, dll

# 3. Build dan run
docker-compose up -d

# 4. Check status
docker-compose ps
curl http://localhost:8080/health
```

## Environment Variables Penting

Di file `.env`, pastikan set:

```bash
# Database connection ke host machine
DB_HOST=host.docker.internal  # Mac/Windows
# DB_HOST=172.17.0.1           # Linux alternative
DB_PASSWORD=your_password

# Redis connection ke host machine  
REDIS_HOST=host.docker.internal  # Mac/Windows
# REDIS_HOST=172.17.0.1           # Linux alternative

# JWT secret (min 64 chars)
JWT_SECRET=generate-random-string-min-64-characters

# Groq API untuk AI scoring
GROQ_API_KEY=your-groq-api-key
```

## Linux Users

Jika menggunakan Linux, ganti `host.docker.internal` dengan `172.17.0.1` di `.env`:

```bash
DB_HOST=172.17.0.1
REDIS_HOST=172.17.0.1
```

Atau tambahkan di `docker-compose.yml` extra_hosts sudah tersedia:
```yaml
extra_hosts:
  - "host.docker.internal:host-gateway"
```

## Common Commands

```bash
# Start
docker-compose up -d

# Stop
docker-compose down

# Logs
docker-compose logs -f api
docker-compose logs -f worker

# Restart
docker-compose restart

# Rebuild
docker-compose up -d --build
```

## Troubleshooting PostgreSQL Connection

Jika API tidak bisa connect ke PostgreSQL:

```bash
# 1. Edit postgresql.conf
sudo nano /etc/postgresql/14/main/postgresql.conf
# Set: listen_addresses = '*'

# 2. Edit pg_hba.conf
sudo nano /etc/postgresql/14/main/pg_hba.conf
# Add: host all all 172.17.0.0/16 md5

# 3. Restart PostgreSQL
sudo systemctl restart postgresql

# 4. Test from container
docker-compose exec api sh
nc -zv host.docker.internal 5432
```

## Troubleshooting Redis Connection

Jika Worker tidak bisa connect ke Redis:

```bash
# 1. Edit redis.conf
sudo nano /etc/redis/redis.conf
# Set: bind 0.0.0.0

# 2. Restart Redis
sudo systemctl restart redis-server

# 3. Test from container
docker-compose exec worker sh
redis-cli -h host.docker.internal ping
```

## Verify Everything Works

```bash
# 1. Check API health
curl http://localhost:8080/health

# 2. Register user (test API)
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password123",
    "role": "creator"
  }'

# 3. Check worker logs
docker-compose logs worker
# Should see: "Worker started successfully"

# 4. Check Redis queue
redis-cli
> XINFO GROUPS assessly:scoring
```

## Production Notes

Untuk production:
1. Set `SERVER_ENV=production` di `.env`
2. Generate strong `JWT_SECRET` (min 64 karakter random)
3. Enable PostgreSQL SSL: `DB_SSL_MODE=require`
4. Set Redis password
5. Configure proper firewall rules
6. Setup reverse proxy (nginx) dengan SSL/TLS
7. Regular backups database

Lihat [DEPLOYMENT.md](DEPLOYMENT.md) untuk panduan lengkap.
