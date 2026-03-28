# Contract Tests

Contract tests validate the HTTP API interface stability for Assessly Backend. These tests ensure that the API contracts (request/response schemas, status codes, headers) remain consistent and don't break frontend integration.

## Overview

Contract tests complement the test pyramid:
- **Unit Tests**: Validate business logic correctness
- **Integration Tests**: Validate end-to-end flows with real database
- **Contract Tests**: Validate API interface stability (HTTP layer)

## Test Structure

### Authentication Endpoints (`auth_contract_test.go`)

#### TestAuthRegisterContract - `POST /api/v1/auth/register`
Validates the registration endpoint contract:
- ✅ Returns 201 with correct response schema on success
- ✅ Returns 400 on invalid JSON body
- ✅ Returns 400 when role is invalid
- ✅ Returns 409 when user already exists
- ✅ Returns 400 when validation fails

**Request Schema:**
```json
{
  "email": "string",
  "password": "string",
  "role": "creator" | "reviewer"
}
```

**Response Schema (201 Created):**
```json
{
  "id": "uuid",
  "email": "string",
  "role": "creator" | "reviewer",
  "created_at": "ISO8601 timestamp"
}
```

#### TestAuthLoginContract - `POST /api/v1/auth/login`
Validates the login endpoint contract:
- ✅ Returns 200 with correct response schema on success
- ✅ Returns 400 on invalid JSON body
- ✅ Returns 401 on invalid credentials
- ✅ Returns 400 when validation fails

**Request Schema:**
```json
{
  "email": "string",
  "password": "string"
}
```

**Response Schema (200 OK):**
```json
{
  "token": "JWT token string",
  "user": {
    "id": "uuid",
    "email": "string",
    "role": "creator" | "reviewer"
  }
}
```

### Test Management Endpoints (`test_contract_test.go`)

#### TestCreateTestContract - `POST /api/v1/tests` (Protected)
Validates the create test endpoint contract:
- ✅ Returns 201 with correct response schema on success
- ✅ Returns 400 on invalid JSON body
- ✅ Returns 401 when user not authenticated
- ✅ Returns 400 when validation fails

**Headers Required:**
```
Authorization: Bearer {JWT token}
```

**Request Schema:**
```json
{
  "title": "string",
  "description": "string",
  "allow_retakes": boolean
}
```

**Response Schema (201 Created):**
```json
{
  "id": "uuid",
  "creator_id": "uuid",
  "title": "string",
  "description": "string",
  "allow_retakes": boolean,
  "is_published": boolean,
  "created_at": "ISO8601 timestamp",
  "updated_at": "ISO8601 timestamp"
}
```

#### TestAddQuestionContract - `POST /api/v1/tests/:id/questions` (Protected)
Validates the add question endpoint contract:
- ✅ Returns 201 with correct response schema on success
- ✅ Returns 400 on invalid test ID
- ✅ Returns 404 when test not found
- ✅ Returns 400 when test is already published

**Request Schema:**
```json
{
  "text": "string",
  "expected_answer": "string",
  "order_num": integer (optional)
}
```

**Response Schema (201 Created):**
```json
{
  "id": "uuid",
  "test_id": "uuid",
  "text": "string",
  "expected_answer": "string",
  "order_num": integer,
  "created_at": "ISO8601 timestamp"
}
```

#### TestPublishTestContract - `POST /api/v1/tests/:id/publish` (Protected)
Validates the publish test endpoint contract:
- ✅ Returns 200 with correct response schema on success
- ✅ Returns 400 on invalid test ID
- ✅ Returns 404 when test not found
- ✅ Returns 400 when test has no questions
- ✅ Returns 400 when test is already published

**Response Schema (200 OK):**
```json
{
  "id": "uuid",
  "creator_id": "uuid",
  "title": "string",
  "description": "string",
  "allow_retakes": boolean,
  "is_published": boolean,  // true after publishing
  "created_at": "ISO8601 timestamp",
  "updated_at": "ISO8601 timestamp"
}
```

## Error Response Format

All error responses follow this consistent format:
```json
{
  "error": "error message string"
}
```

### HTTP Status Codes
- `200 OK`: Successful request
- `201 Created`: Resource created successfully
- `400 Bad Request`: Validation error or malformed request
- `401 Unauthorized`: Authentication required or failed
- `404 Not Found`: Resource not found
- `409 Conflict`: Resource already exists (e.g., duplicate email)

## Running Contract Tests

```bash
# Run all contract tests
go test ./tests/contract/... -v

# Run specific test file
go test ./tests/contract/auth_contract_test.go -v
go test ./tests/contract/test_contract_test.go -v

# Run with coverage
go test ./tests/contract/... -cover
```

## Test Philosophy

Contract tests:
1. **Focus on HTTP layer**: Test request/response format, not business logic
2. **Use real handlers**: Test actual HTTP handlers with mocked dependencies
3. **Validate schema**: Ensure JSON structure matches documentation
4. **Check status codes**: Verify correct HTTP status codes for each scenario
5. **Test error cases**: Validate error response format and status codes

## Implementation Details

- Tests use `httptest.NewRecorder()` to capture HTTP responses
- Real use cases are initialized with mock repositories
- Mock functions allow precise control over test scenarios
- Tests validate both success and error paths
- Schema validation checks field existence, types, and values

## Coverage

**Total Contract Tests**: 5 test suites with 22 sub-tests

- Authentication: 9 sub-tests (Register + Login)
- Test Management: 13 sub-tests (Create + Add Questions + Publish)

All tests validate:
- ✅ Correct HTTP status codes
- ✅ Response Content-Type (application/json)
- ✅ JSON schema structure
- ✅ Required fields presence
- ✅ Field types (string, boolean, number)
- ✅ Field values correctness
- ✅ Sensitive data exclusion (e.g., password not in response)

## Future Enhancements

Potential additions:
- Submission endpoint contract tests
- Review endpoint contract tests
- More edge case scenarios
- Response header validation (CORS, Cache-Control)
- Request header validation (Content-Type requirements)
