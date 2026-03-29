# Assessly API Documentation

Complete API documentation for Assessly Backend System.

## 📚 Documentation Formats

We provide API documentation in multiple formats:

### 1. **OpenAPI/Swagger** (Recommended)
- **File**: `docs/openapi.yaml`
- **View Online**: https://editor.swagger.io/
  - Copy content of `openapi.yaml` and paste into Swagger Editor
- **Local Swagger UI**: (Coming soon - add swagger middleware)

### 2. **Postman Collection**
- **File**: `docs/postman-collection.json`
- **Import**: Open Postman → Import → Choose file
- **Environment Variables**: Set in Postman:
  ```
  base_url: https://assessly-be-production.up.railway.app
  creator_token: <your-JWT-token>
  test_id: <uuid>
  access_token: <participant-access-token>
  ```

### 3. **cURL Examples**
- **Script**: `tests/curl/test-api-flow.sh`
- **Run**: `bash tests/curl/test-api-flow.sh`

---

## 🚀 Quick Start

### Base URLs

**Local Development**:
```
http://localhost:8080/api/v1
```

**Production (Railway)**:
```
https://assessly-be-production.up.railway.app/api/v1
```

---

## 🔐 Authentication

### User Authentication (JWT)

**For Creators & Reviewers**:

1. **Register**:
   ```bash
   POST /api/v1/auth/register
   {
     "name": "John Doe",
     "email": "john@example.com",
     "password": "SecurePass123!",
     "role": "creator"
   }
   ```

2. **Login**:
   ```bash
   POST /api/v1/auth/login
   {
     "email": "john@example.com",
     "password": "SecurePass123!  "
   }
   
   Response:
   {
     "token": "eyJhbGci...",
     "user": {...}
   }
   ```

3. **Use Token**:
   ```bash
   Authorization: Bearer eyJhbGci...
   ```

### Participant Access Tokens

**For Test Submissions**:

1. **Generate Access Token** (Creator only):
   ```bash
   POST /api/v1/tests/{testID}/access-token
   Authorization: Bearer <creator-jwt>
   {
     "email": "participant@example.com",
     "expiry_hours": 24
   }
   
   Response:
   {
     "access_token": "eyJhbGci...",
     "test_id": "uuid",
     "email": "participant@example.com",
     "expiry_hours": 24
   }
   ```

2. **Use Access Token**:
   ```bash
   # Option 1: Header
   X-Access-Token: eyJhbGci...
   
   # Option 2: Request body
   {
     "access_token": "eyJhbGci...",
     "answers": [...]
   }
   ```

---

## 📋 API Endpoints

### Health Check

```bash
GET /health

Response:
{
  "status": "healthy",
  "database": "connected",
  "redis": "connected"
}
```

---

### Authentication

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| POST | `/auth/register` | None | Register new user |
| POST | `/auth/login` | None | Login and get JWT token |
| POST | `/auth/request-reset` | None | Request password reset |
| PUT | `/auth/reset-password` | None | Reset password with token |

---

### Tests (Creator Only)

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| POST | `/tests` | JWT (creator) | Create new test |
| POST | `/tests/{testID}/publish` | JWT (creator) | Publish test |
| POST | `/tests/{testID}/questions` | JWT (creator) | Add question to test |
| **POST** | **`/tests/{testID}/access-token`** | **JWT (creator)** | **Generate participant access token** |

---

### Submissions (Participants)

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| POST | `/submissions/access` | None | Request access (sends email) |
| POST | `/submissions` | Access Token | Submit test answers |
| GET | `/submissions/{id}` | Access Token or JWT | Get submission with scores |

---

### Reviews (Reviewers)

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| GET | `/tests/{testID}/submissions` | JWT (reviewer) | List submissions for test |
| PUT | `/reviews/{answerID}` | JWT (reviewer) | Add/update manual review |
| GET | `/reviews/{answerID}` | JWT (reviewer) | Get review details |

---

## 💡 Common Use Cases

### Use Case 1: Create and Publish Test

```bash
# 1. Register Creator
POST /auth/register
{
  "name": "Teacher",
  "email": "teacher@school.com",
  "password": "SecurePass123!",
  "role": "creator"
}

# 2. Login
POST /auth/login
{
  "email": "teacher@school.com",
  "password": "SecurePass123!"
}
# Save token from response

# 3. Create Test
POST /tests
Authorization: Bearer <token>
{
  "title": "JavaScript Basics",
  "description": "Test your JS knowledge",
  "allow_retakes": false
}
# Save test_id from response

# 4. Add Questions
POST /tests/{test_id}/questions
Authorization: Bearer <token>
{
  "text": "What is a closure?",
  "expected_answer": "A closure is a function that retains access...",
  "order_num": 1
}

# 5. Publish Test
POST /tests/{test_id}/publish
Authorization: Bearer <token>
```

---

### Use Case 2: Participant Takes Test

```bash
# 1. Creator generates access token
POST /tests/{test_id}/access-token
Authorization: Bearer <creator-token>
{
  "email": "student@example.com",
  "expiry_hours": 24
}
# Save access_token from response

# 2. Participant submits answers
POST /submissions
X-Access-Token: <access-token>
{
  "answers": [
    {
      "question_id": "uuid",
      "text": "A closure is a function..."
    }
  ]
}
# Save submission_id

# 3. Wait for AI scoring (5-10 seconds)

# 4. Get results
GET /submissions/{submission_id}
X-Access-Token: <access-token>

Response:
{
  "submission": {
    "id": "uuid",
    "ai_total_score": 95,
    "submitted_at": "..."
  },
  "answers": [
    {
      "id": "uuid",
      "text": "A closure is...",
      "review": {
        "ai_score": 95,
        "ai_feedback": "Excellent answer...",
        "display_score": 95
      }
    }
  ]
}
```

---

### Use Case 3: Reviewer Adds Manual Score

```bash
# 1. Login as Reviewer
POST /auth/login
{
  "email": "reviewer@example.com",
  "password": "SecurePass123!"
}

# 2. List Submissions for Test
GET /tests/{test_id}/submissions
Authorization: Bearer <reviewer-token>

# 3. Add Manual Review
PUT /reviews/{answer_id}
Authorization: Bearer <reviewer-token>
{
  "manual_score": 98,
  "manual_feedback": "Great answer, but could mention lexical scope..."
}
```

---

## 🧪 Testing

### Automated API Flow Test

```bash
# Run complete API flow test (includes AI scoring)
bash tests/curl/test-api-flow.sh

# Expected output:
# ✅ Health check
# ✅ User registration
# ✅ Test creation
# ✅ Questions added
# ✅ Test published
# ✅ Access token generated
# ✅ Submission created
# 🤖 AI Scoring completed
# ✅ Results retrieved
```

### Manual Testing with cURL

See examples in `tests/curl/` directory for each endpoint.

---

## 📊 Response Codes

| Code | Meaning | When |
|------|---------|------|
| 200 | OK | Successful GET/PUT request |
| 201 | Created | Successful POST (resource created) |
| 400 | Bad Request | Invalid input data |
| 401 | Unauthorized | Missing or invalid token |
| 403 | Forbidden | Insufficient permissions |
| 404 | Not Found | Resource doesn't exist |
| 409 | Conflict | Resource already exists |
| 500 | Internal Server Error | Server error (check logs) |

---

## 🔧 Environment Setup

### Required Environment Variables

**For API Service**:
```bash
# Database
DB_HOST=<postgres-host>
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=<password>
DB_NAME=railway

# Redis
REDIS_HOST=<redis-host>
REDIS_PORT=<redis-port>
REDIS_PASSWORD=<redis-password>

# JWT
JWT_SECRET=<random-64-chars>
JWT_EXPIRY_HOURS=24

# Groq AI (for scoring)
GROQ_API_KEY=<your-groq-key>
GROQ_MODEL=llama-3.1-70b-versatile
```

**For Worker Service** (same as above, required for AI scoring).

---

## 🐛 Troubleshooting

### "Unauthorized" Error
- Check if JWT token is valid (not expired)
- Verify `Authorization: Bearer <token>` header format
- For submissions, use `X-Access-Token` header instead

### "AI scoring not completed"
- Verify Worker service is deployed and running
- Check `GROQ_API_KEY` is set in worker environment
- Verify `GROQ_MODEL=llama-3.1-70b-versatile`
- Check worker logs for errors

### "Database connection failed"
- Verify `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD` are correct
- Check if database migration has been run
- For Railway: Use internal database host URL

### "Redis connection failed"
- Verify `REDIS_HOST`, `REDIS_PORT`, `REDIS_PASSWORD`
- Check if Redis service is running
- Test connection: `redis-cli -h <host> -p <port> -a <password> PING`

---

## 📝 Notes

### Access Token vs JWT Token

- **JWT Token**: For authenticated users (creators, reviewers)
  - Long-lived (24 hours default)
  - Contains user ID, email, role
  - Used in `Authorization: Bearer <token>` header

- **Access Token**: For anonymous participants
  - Short-lived (24 hours default, configurable)
  - Contains test ID, participant email
  - Used in `X-Access-Token: <token>` header
  - Role is always "participant"

### AI Scoring Flow

1. Participant submits answers → Submission created
2. API enqueues scoring job to Redis
3. Worker picks up job from Redis
4. Worker calls Groq API for each answer
5. Worker saves AI scores to database
6. Participant retrieves submission with scores

Typical processing time: 3-10 seconds for 3 questions.

---

## 🔗 Additional Resources

- **OpenAPI Spec**: `docs/openapi.yaml`
- **Postman Collection**: `docs/postman-collection.json`
- **cURL Scripts**: `tests/curl/`
- **Contract Tests**: `tests/contract/`
- **Integration Tests**: `tests/integration/`

---

## 📮 Support

For issues or questions:
- Check Railway logs for deployment errors
- Review contract tests for expected behavior
- Run `bash tests/curl/test-api-flow.sh` to verify system health
