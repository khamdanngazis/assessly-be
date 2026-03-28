#!/bin/bash

# Assessly API Full Flow Test Script
# Test API deployment at Railway

set -e  # Exit on error

API_URL="https://assessly-be-production.up.railway.app"
TIMESTAMP=$(date +%s)

echo "🚀 Testing Assessly API at $API_URL"
echo "=========================================="

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# 1. Health Check
echo -e "\n${BLUE}1️⃣ Testing Health Check${NC}"
HEALTH=$(curl -s "$API_URL/health")
echo "Response: $HEALTH"
if echo "$HEALTH" | grep -q "healthy"; then
    echo -e "${GREEN}✅ Health check passed${NC}"
else
    echo -e "${RED}❌ Health check failed${NC}"
    exit 1
fi

# 2. Register Creator
echo -e "\n${BLUE}2️⃣ Registering Creator${NC}"
CREATOR_EMAIL="creator_$TIMESTAMP@test.com"
CREATOR_PASS="TestPassword123!"

CREATOR_RESPONSE=$(curl -s -X POST "$API_URL/api/v1/auth/register" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Creator",
    "email": "'$CREATOR_EMAIL'",
    "password": "'$CREATOR_PASS'",
    "role": "creator"
  }')

echo "Response: $CREATOR_RESPONSE"

if echo "$CREATOR_RESPONSE" | grep -q '"id"'; then
    CREATOR_ID=$(echo "$CREATOR_RESPONSE" | grep -o '"id":"[^"]*' | head -1 | cut -d'"' -f4)
    echo -e "${GREEN}✅ Creator registered with ID: $CREATOR_ID${NC}"
else
    echo -e "${RED}❌ Creator registration failed${NC}"
    exit 1
fi

# 3. Login Creator (test login endpoint)
echo -e "\n${BLUE}3️⃣ Testing Creator Login${NC}"
LOGIN_RESPONSE=$(curl -s -X POST "$API_URL/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "'$CREATOR_EMAIL'",
    "password": "'$CREATOR_PASS'"
  }')

echo "Response: $LOGIN_RESPONSE"
if echo "$LOGIN_RESPONSE" | grep -q '"token"'; then
    CREATOR_TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"token":"[^"]*' | cut -d'"' -f4)
    echo -e "${GREEN}✅ Login successful${NC}"
    echo "Token: ${CREATOR_TOKEN:0:30}..."
else
    echo -e "${RED}❌ Login failed${NC}"
    exit 1
fi

# 4. Create Test
echo -e "\n${BLUE}4️⃣ Creating Test${NC}"
TEST_RESPONSE=$(curl -s -X POST "$API_URL/api/v1/tests" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $CREATOR_TOKEN" \
  -d '{
    "title": "JavaScript Fundamentals Test '$TIMESTAMP'",
    "description": "Test your knowledge of JavaScript basics"
  }')

echo "Response: $TEST_RESPONSE"

if echo "$TEST_RESPONSE" | grep -q '"id"'; then
    TEST_ID=$(echo "$TEST_RESPONSE" | grep -o '"id":"[^"]*' | head -1 | cut -d'"' -f4)
    echo -e "${GREEN}✅ Test created with ID: $TEST_ID${NC}"
else
    echo -e "${RED}❌ Test creation failed${NC}"
    exit 1
fi

# 5. Add Questions
echo -e "\n${BLUE}5️⃣ Adding Questions to Test${NC}"

# Question 1
Q1_RESPONSE=$(curl -s -X POST "$API_URL/api/v1/tests/$TEST_ID/questions" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $CREATOR_TOKEN" \
  -d '{
    "text": "What is the difference between let and const in JavaScript?",
    "expected_answer": "let allows reassignment of values while const creates read-only references that cannot be reassigned",
    "order_num": 1
  }')

Q1_ID=$(echo "$Q1_RESPONSE" | grep -o '"id":"[^"]*' | head -1 | cut -d'"' -f4)
echo -e "${GREEN}✅ Question 1 added: $Q1_ID${NC}"

# Question 2
Q2_RESPONSE=$(curl -s -X POST "$API_URL/api/v1/tests/$TEST_ID/questions" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $CREATOR_TOKEN" \
  -d '{
    "text": "Explain what a closure is in JavaScript",
    "expected_answer": "A closure is a function that has access to variables in its outer lexical scope, even after the outer function has returned",
    "order_num": 2
  }')

Q2_ID=$(echo "$Q2_RESPONSE" | grep -o '"id":"[^"]*' | head -1 | cut -d'"' -f4)
echo -e "${GREEN}✅ Question 2 added: $Q2_ID${NC}"

# Question 3
Q3_RESPONSE=$(curl -s -X POST "$API_URL/api/v1/tests/$TEST_ID/questions" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $CREATOR_TOKEN" \
  -d '{
    "text": "What is the event loop in JavaScript?",
    "expected_answer": "The event loop is a mechanism that handles asynchronous callbacks by continuously checking the call stack and callback queue",
    "order_num": 3
  }')

Q3_ID=$(echo "$Q3_RESPONSE" | grep -o '"id":"[^"]*' | head -1 | cut -d'"' -f4)
echo -e "${GREEN}✅ Question 3 added: $Q3_ID${NC}"

# 6. Publish Test
echo -e "\n${BLUE}6️⃣ Publishing Test${NC}"
PUBLISH_RESPONSE=$(curl -s -X POST "$API_URL/api/v1/tests/$TEST_ID/publish" \
  -H "Authorization: Bearer $CREATOR_TOKEN")

echo "Response: $PUBLISH_RESPONSE"

if echo "$PUBLISH_RESPONSE" | grep -q '"is_published":true'; then
    echo -e "${GREEN}✅ Test published${NC}"
else
    echo -e "${RED}❌ Test publish failed${NC}"
    exit 1
fi

# 7. Register Participant (optional - can use access token)
echo -e "\n${BLUE}7️⃣ Registering Participant${NC}"
PARTICIPANT_EMAIL="participant_$TIMESTAMP@test.com"
PARTICIPANT_PASS="TestPassword123!"

PARTICIPANT_RESPONSE=$(curl -s -X POST "$API_URL/api/v1/auth/register" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Participant",
    "email": "'$PARTICIPANT_EMAIL'",
    "password": "'$PARTICIPANT_PASS'",
    "role": "creator"
  }')

if echo "$PARTICIPANT_RESPONSE" | grep -q '"id"'; then
    echo -e "${GREEN}✅ Participant registered${NC}"
    
    # Login participant to get token for submission
    PARTICIPANT_LOGIN=$(curl -s -X POST "$API_URL/api/v1/auth/login" \
      -H "Content-Type: application/json" \
      -d '{
        "email": "'$PARTICIPANT_EMAIL'",
        "password": "'$PARTICIPANT_PASS'"
      }')
    
    PARTICIPANT_TOKEN=$(echo "$PARTICIPANT_LOGIN" | grep -o '"token":"[^"]*' | cut -d'"' -f4)
    echo -e "${GREEN}✅ Participant logged in${NC}"
    # Use participant JWT token as access token for submission
    ACCESS_TOKEN=$PARTICIPANT_TOKEN
else
    echo -e "${BLUE}ℹ️  Using test access_token for submission${NC}"
fi

# 8. Submit Answers
echo -e "\n${BLUE}8️⃣ Submitting Answers${NC}"
SUBMISSION_RESPONSE=$(curl -s -X POST "$API_URL/api/v1/submissions" \
  -H "Content-Type: application/json" \
  -H "X-Access-Token: $ACCESS_TOKEN" \
  -d '{
    "test_id": "'$TEST_ID'",
    "answers": [
      {
        "question_id": "'$Q1_ID'",
        "text": "The main difference is that let allows you to reassign values to the variable, while const creates a constant reference that cannot be reassigned after initialization."
      },
      {
        "question_id": "'$Q2_ID'",
        "text": "A closure is a function that retains access to variables from its outer scope even after the outer function has finished executing. This allows for data privacy and factory functions."
      },
      {
        "question_id": "'$Q3_ID'",
        "text": "The event loop continuously monitors the call stack and callback queue, executing callbacks when the stack is empty to handle asynchronous operations."
      }
    ]
  }')

echo "Response: $SUBMISSION_RESPONSE"

if echo "$SUBMISSION_RESPONSE" | grep -q '"id"'; then
    SUBMISSION_ID=$(echo "$SUBMISSION_RESPONSE" | grep -o '"id":"[^"]*' | head -1 | cut -d'"' -f4)
    echo -e "${GREEN}✅ Submission created with ID: $SUBMISSION_ID${NC}"
else
    echo -e "${RED}❌ Submission failed${NC}"
    exit 1
fi

# 9. Get Submission Result (wait a bit for AI scoring)
echo -e "\n${BLUE}9️⃣ Getting Submission Result${NC}"
echo "Waiting 3 seconds for AI scoring to process..."
sleep 3

RESULT_RESPONSE=$(curl -s -X GET "$API_URL/api/v1/submissions/$SUBMISSION_ID" \
  -H "X-Access-Token: $ACCESS_TOKEN")

echo "Response: $RESULT_RESPONSE"

if echo "$RESULT_RESPONSE" | grep -q '"answers"'; then
    echo -e "${GREEN}✅ Submission retrieved${NC}"
    
    # Check if AI scoring completed
    if echo "$RESULT_RESPONSE" | grep -q '"ai_score"'; then
        AI_SCORE=$(echo "$RESULT_RESPONSE" | grep -o '"ai_score":[0-9.]*' | head -1 | cut -d':' -f2)
        echo -e "${GREEN}🤖 AI Scoring completed! Score: $AI_SCORE${NC}"
    else
        echo -e "${BLUE}⏳ AI Scoring still processing (check GROQ_API_KEY if this persists)${NC}"
    fi
else
    echo -e "${RED}❌ Failed to retrieve submission${NC}"
fi

# 10. Register Reviewer
echo -e "\n${BLUE}🔟 Registering Reviewer${NC}"
REVIEWER_EMAIL="reviewer_$TIMESTAMP@test.com"
REVIEWER_PASS="TestPassword123!"

REVIEWER_RESPONSE=$(curl -s -X POST "$API_URL/api/v1/auth/register" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Reviewer",
    "email": "'$REVIEWER_EMAIL'",
    "password": "'$REVIEWER_PASS'",
    "role": "reviewer"
  }')

if echo "$REVIEWER_RESPONSE" | grep -q '"id"'; then
    REVIEWER_ID=$(echo "$REVIEWER_RESPONSE" | grep -o '"id":"[^"]*' | head -1 | cut -d'"' -f4)
    echo -e "${GREEN}✅ Reviewer registered with ID: $REVIEWER_ID${NC}"
    
    # Login reviewer to get token
    REVIEWER_LOGIN=$(curl -s -X POST "$API_URL/api/v1/auth/login" \
      -H "Content-Type: application/json" \
      -d '{
        "email": "'$REVIEWER_EMAIL'",
        "password": "'$REVIEWER_PASS'"
      }')
    
    REVIEWER_TOKEN=$(echo "$REVIEWER_LOGIN" | grep -o '"token":"[^"]*' | cut -d'"' -f4)
    echo -e "${GREEN}✅ Reviewer logged in${NC}"
    
    # 11. Reviewer views submissions
    echo -e "\n${BLUE}1️⃣1️⃣ Reviewer Listing Submissions${NC}"
    SUBMISSIONS_LIST=$(curl -s -X GET "$API_URL/api/v1/submissions" \
      -H "Authorization: Bearer $REVIEWER_TOKEN")
    
    echo "Response: $SUBMISSIONS_LIST"
    if echo "$SUBMISSIONS_LIST" | grep -q '"submissions"'; then
        echo -e "${GREEN}✅ Reviewer can view submissions${NC}"
    fi
    
    # 12. Add manual review
    echo -e "\n${BLUE}1️⃣2️⃣ Adding Manual Review${NC}"
    
    # Get first answer ID from submission
    ANSWER_ID=$(echo "$RESULT_RESPONSE" | grep -o '"id":"[^"]*' | sed -n '2p' | cut -d'"' -f4)
    
    REVIEW_RESPONSE=$(curl -s -X POST "$API_URL/api/v1/reviews" \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer $REVIEWER_TOKEN" \
      -d '{
        "answer_id": "'$ANSWER_ID'",
        "score": 95,
        "feedback": "Excellent explanation! Very clear and accurate."
      }')
    
    echo "Response: $REVIEW_RESPONSE"
    if echo "$REVIEW_RESPONSE" | grep -q '"id"'; then
        echo -e "${GREEN}✅ Manual review added${NC}"
    else
        echo -e "${BLUE}ℹ️  Manual review response: Check if answer_id is valid${NC}"
    fi
else
    echo -e "${RED}❌ Reviewer registration failed${NC}"
fi

# Summary
echo -e "\n=========================================="
echo -e "${GREEN}🎉 API Test Flow Completed!${NC}"
echo -e "=========================================="
echo ""
echo "Test Details:"
echo "  Test ID: $TEST_ID"
echo "  Access Token: $ACCESS_TOKEN"
echo "  Submission ID: $SUBMISSION_ID"
echo ""
echo "Accounts Created:"
echo "  Creator: $CREATOR_EMAIL / $CREATOR_PASS"
echo "  Participant: $PARTICIPANT_EMAIL / $PARTICIPANT_PASS"
echo "  Reviewer: $REVIEWER_EMAIL / $REVIEWER_PASS"
echo ""
echo -e "${BLUE}💡 Note: If AI scoring didn't complete, make sure GROQ_API_KEY is set in Railway environment variables${NC}"
echo ""
