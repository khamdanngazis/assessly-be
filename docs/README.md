# 📚 API Documentation Guide

Quick guide for using Assessly API documentation.

## Available Documentation

### 1. **OpenAPI/Swagger Specification** 
- **File**: [`openapi.yaml`](./openapi.yaml)
- **Format**: OpenAPI 3.0.3
- **Use for**: 
  - Auto-generate client SDKs
  - Test endpoints in Swagger UI
  - Share with frontend teams
  - Generate API docs websites

#### View in Swagger Editor

**Online (No install required)**:
1. Go to https://editor.swagger.io/
2. Click **File** → **Import file**
3. Upload `docs/openapi.yaml`
4. Explore endpoints, try them out!

**Or paste URL** (if public):
```
https://raw.githubusercontent.com/your-org/assessly-be/main/docs/openapi.yaml
```

#### View Locally with Swagger UI

```bash
# Using Docker (easiest)
docker run -p 8081:8080 \
  -e SWAGGER_JSON=/docs/openapi.yaml \
  -v $(pwd)/docs:/docs \
  swaggerapi/swagger-ui

# Open: http://localhost:8081
```

**Or install Swagger UI server**:
```bash
npm install -g swagger-ui-watcher
swagger-ui-watcher docs/openapi.yaml

# Open: http://localhost:8080
```

---

### 2. **Postman Collection**
- **File**: [`postman-collection.json`](./postman-collection.json)
- **Format**: Postman Collection v2.1
- **Use for**: Manual API testing, team collaboration

#### Import to Postman

1. **Open Postman** (download from https://www.postman.com/)
2. Click **Import** button (top left)
3. **Choose files** → Select `docs/postman-collection.json`
4. Collection imported! ✅

#### Set Environment Variables

Create environment in Postman:

| Variable | Value | Description |
|----------|-------|-------------|
| `base_url` | `https://assessly-be-production.up.railway.app/api/v1` | API base URL |
| `jwt_token` | _(auto-filled after login)_ | JWT token for creators/reviewers |
| `access_token` | _(auto-filled after generation)_ | Participant access token |
| `test_id` | _(auto-filled after test creation)_ | Current test UUID |
| `submission_id` | _(auto-filled)_ | Current submission UUID |
| `question_id` | _(auto-filled)_ | Current question UUID |
| `answer_id` | _(manual)_ | Answer UUID for reviews |

#### Usage Flow in Postman

**Sequential Testing** (auto-saves variables):

1. **Authentication** → Register Creator
2. **Authentication** → Login Creator _(saves jwt_token)_
3. **Tests** → Create Test _(saves test_id)_
4. **Tests** → Add Question _(saves question_id)_
5. **Tests** → Publish Test
6. **Tests** → Generate Access Token _(saves access_token)_
7. **Submissions** → Submit Test _(saves submission_id)_
8. **Submissions** → Get Submission with AI Scores

---

### 3. **Markdown Documentation**
- **File**: [`API_DOCUMENTATION.md`](./API_DOCUMENTATION.md)
- **Format**: Markdown
- **Use for**: Quick reference, onboarding, troubleshooting

#### View

- **GitHub**: Automatically rendered when browsing repo
- **VS Code**: Preview with `Ctrl+Shift+V` (or `Cmd+Shift+V` on Mac)
- **Terminal**: Use `mdcat`, `glow`, or `bat`:
  ```bash
  # Install glow: https://github.com/charmbracelet/glow
  glow API_DOCUMENTATION.md
  ```

---

## Quick Start Examples

### Example 1: Test API with cURL (from docs)

```bash
# 1. Health Check
curl https://assessly-be-production.up.railway.app/health

# 2. Register
curl -X POST https://assessly-be-production.up.railway.app/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "John Doe",
    "email": "john@example.com",
    "password": "SecurePass123!",
    "role": "creator"
  }'

# 3. Login
curl -X POST https://assessly-be-production.up.railway.app/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "john@example.com",
    "password": "SecurePass123!"
  }'

# Save token from response, then:

# 4. Create Test
curl -X POST https://assessly-be-production.up.railway.app/api/v1/tests \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "JavaScript Quiz",
    "description": "Test your JS skills",
    "allow_retakes": false
  }'
```

### Example 2: Run Automated Test Script

```bash
# From project root
cd /home/azis/p/assessly/assessly-be

# Run complete API flow test
bash tests/curl/test-api-flow.sh

# Expected: All endpoints tested, AI scoring verified ✅
```

---

## Documentation Best Practices

### For Developers

1. **Create tests** → Use Postman collection for each endpoint
2. **Update OpenAPI** → Keep `openapi.yaml` in sync with code
3. **Add examples** → Include request/response examples
4. **Document errors** → List all possible error codes
5. **Version docs** → Tag docs with API versions

### For API Consumers

1. **Start with Swagger** → Visual interface is easiest
2. **Use Postman** → For manual testing and debugging
3. **Read Markdown** → For understanding concepts
4. **Try cURL** → For automation/scripting

---

## Generating Docs from OpenAPI

### Generate HTML Docs

```bash
# Using Redoc (beautiful docs)
npx @redocly/cli build-docs docs/openapi.yaml \
  -o docs/api.html

# Open docs/api.html in browser
```

### Generate Markdown from OpenAPI

```bash
# Using widdershins
npm install -g widdershins
widdershins docs/openapi.yaml -o docs/API_REFERENCE_GENERATED.md
```

### Generate Client SDKs

```bash
# JavaScript/TypeScript client
npx @openapitools/openapi-generator-cli generate \
  -i docs/openapi.yaml \
  -g typescript-axios \
  -o clients/typescript/

# Python client
npx @openapitools/openapi-generator-cli generate \
  -i docs/openapi.yaml \
  -g python \
  -o clients/python/

# Go client
npx @openapitools/openapi-generator-cli generate \
  -i docs/openapi.yaml \
  -g go \
  -o clients/go/
```

---

## Updating Documentation

### When to Update

Update docs when you:
- ✅ Add new endpoint
- ✅ Change request/response format
- ✅ Add new query parameters or headers
- ✅ Change authentication method
- ✅ Update error codes
- ✅ Deprecate endpoints

### How to Update

1. **OpenAPI** (`openapi.yaml`):
   - Add endpoint under appropriate tag
   - Define request/response schemas
   - Add examples
   - Update version number

2. **Postman** (`postman-collection.json`):
   - Add request to appropriate folder
   - Set pre-request scripts (if needed)
   - Add test scripts to save variables
   - Test manually in Postman

3. **Markdown** (`API_DOCUMENTATION.md`):
   - Update Quick Start if flow changes
   - Add to Common Use Cases
   - Update troubleshooting section
   - Add new endpoints to table

### Validation

```bash
# Validate OpenAPI spec
npx @openapitools/openapi-generator-cli validate -i docs/openapi.yaml

# Or use online validator
# https://apitools.dev/swagger-parser/online/
```

---

## Testing Documentation

### Test OpenAPI Spec

```bash
# 1. Start local API server
make run-api

# 2. In another terminal, test with Swagger UI
docker run -p 8081:8080 \
  -e SWAGGER_JSON=/docs/openapi.yaml \
  -v $(pwd)/docs:/docs \
  swaggerapi/swagger-ui

# 3. Test each endpoint in Swagger UI
# http://localhost:8081
```

### Test Postman Collection

```bash
# Run collection with Newman (Postman CLI)
npm install -g newman

newman run docs/postman-collection.json \
  --environment postman-environment.json \
  --reporters cli,html \
  --reporter-html-export testrun-report.html
```

---

## Additional Resources

- **OpenAPI Spec**: https://spec.openapis.org/oas/v3.0.3
- **Swagger Editor**: https://editor.swagger.io/
- **Postman Learning**: https://learning.postman.com/
- **Redoc**: https://github.com/Redocly/redoc
- **API Design Guide**: https://apiguide.readthedocs.io/

---

## Support

For API documentation questions:
- Check `docs/API_DOCUMENTATION.md` for usage examples
- Test endpoints with `tests/curl/test-api-flow.sh`
- Validate OpenAPI spec before deploying
- Keep Postman collection in sync with OpenAPI

---

**Last Updated**: March 28, 2026  
**API Version**: 1.0.0  
**OpenAPI Version**: 3.0.3
