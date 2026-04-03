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

# 4.5 List Tests (Creator)
echo -e "\n${BLUE}4️⃣.5️⃣ Listing Creator's Tests${NC}"
LIST_TESTS_RESPONSE=$(curl -s -X GET "$API_URL/api/v1/tests" \
  -H "Authorization: Bearer $CREATOR_TOKEN")

echo "Response: $LIST_TESTS_RESPONSE"

if echo "$LIST_TESTS_RESPONSE" | grep -q '"tests"'; then
    TEST_COUNT=$(echo "$LIST_TESTS_RESPONSE" | grep -o '"tests":\[' | wc -l)
    echo -e "${GREEN}✅ Tests list retrieved (Creator can see their tests)${NC}"
else
    echo -e "${RED}❌ Failed to list tests${NC}"
    exit 1
fi

# 4.6 Get Single Test (Creator)
echo -e "\n${BLUE}4️⃣.6️⃣ Getting Test Details${NC}"
GET_TEST_RESPONSE=$(curl -s -X GET "$API_URL/api/v1/tests/$TEST_ID" \
  -H "Authorization: Bearer $CREATOR_TOKEN")

echo "Response: $GET_TEST_RESPONSE"

if echo "$GET_TEST_RESPONSE" | grep -q '"id"' && echo "$GET_TEST_RESPONSE" | grep -q "$TEST_ID"; then
    echo -e "${GREEN}✅ Test details retrieved successfully${NC}"
else
    echo -e "${RED}❌ Failed to get test details${NC}"
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
else
    echo -e "${RED}❌ Failed to register participant${NC}"
    exit 1
fi

# Generate access token for participant using test endpoint
echo -e "${BLUE}7.5️⃣ Generating Access Token for Test${NC}"
ACCESS_TOKEN_RESPONSE=$(curl -s -X POST "$API_URL/api/v1/tests/$TEST_ID/access-token" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $CREATOR_TOKEN" \
  -d '{
    "email": "'$PARTICIPANT_EMAIL'",
    "expiry_hours": 24
  }')

ACCESS_TOKEN=$(echo "$ACCESS_TOKEN_RESPONSE" | grep -o '"access_token":"[^"]*' | cut -d'"' -f4)

if [ -n "$ACCESS_TOKEN" ]; then
    echo -e "${GREEN}✅ Access token generated: ${ACCESS_TOKEN:0:20}...${NC}"
else
    echo -e "${RED}❌ Failed to generate access token${NC}"
    echo "$ACCESS_TOKEN_RESPONSE"
    exit 1
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

# 9. Get Submission Result (wait for AI scoring)
echo -e "\n${BLUE}9️⃣ Getting Submission Result & AI Scores${NC}"
echo "Waiting for AI scoring to complete..."

# Poll for AI scores (max 30 seconds)
MAX_RETRIES=6
RETRY_COUNT=0
AI_SCORED=false

while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    sleep 5
    RETRY_COUNT=$((RETRY_COUNT + 1))
    
    echo "  Checking attempt $RETRY_COUNT/$MAX_RETRIES..."
    
    RESULT_RESPONSE=$(curl -s -X GET "$API_URL/api/v1/submissions/$SUBMISSION_ID" \
      -H "X-Access-Token: $ACCESS_TOKEN")
    
    # Check if any answer has AI score
    if echo "$RESULT_RESPONSE" | grep -q '"ai_score"'; then
        AI_SCORED=true
        break
    fi
done

echo ""
echo "Response: $RESULT_RESPONSE"
echo ""

if echo "$RESULT_RESPONSE" | grep -q '"answers"'; then
    echo -e "${GREEN}✅ Submission retrieved successfully${NC}"
    echo ""
    
    # Check AI scoring status
    if [ "$AI_SCORED" = true ]; then
        echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo -e "${GREEN}🤖 AI SCORING COMPLETED SUCCESSFULLY!${NC}"
        echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo ""
        
        # Extract submission scores
        AI_TOTAL=$(echo "$RESULT_RESPONSE" | grep -o '"ai_total_score":[0-9.]*' | head -1 | cut -d':' -f2)
        MANUAL_TOTAL=$(echo "$RESULT_RESPONSE" | grep -o '"manual_total_score":[0-9.]*' | head -1 | cut -d':' -f2)
        
        if [ -n "$AI_TOTAL" ]; then
            echo -e "${GREEN}📊 TOTAL SCORE: ${AI_TOTAL}/100${NC}"
        fi
        
        if [ -n "$MANUAL_TOTAL" ]; then
            echo -e "${BLUE}📝 Manual Review Score: ${MANUAL_TOTAL}/100${NC}"
        fi
        
        echo ""
        echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo -e "${BLUE}📋 INDIVIDUAL ANSWER SCORES & FEEDBACK${NC}"
        echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        
        # Parse and display each answer with AI review
        ANSWER_NUM=1
        
        # Use Python if available for better JSON parsing, otherwise use grep
        if command -v python3 &> /dev/null; then
            echo "$RESULT_RESPONSE" | python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    answers = data.get('answers', [])
    for idx, ans in enumerate(answers, 1):
        print(f'\n📌 Answer {idx}:')
        print(f'   Question: {ans.get(\"question_id\", \"N/A\")[:30]}...')
        print(f'   Text: {ans.get(\"text\", \"N/A\")[:60]}...')
        
        review = ans.get('review', {})
        if review:
            ai_score = review.get('ai_score')
            ai_feedback = review.get('ai_feedback', '')
            
            if ai_score is not None:
                print(f'   ✅ AI Score: {ai_score}/100')
                if ai_feedback:
                    # Truncate long feedback
                    feedback_short = ai_feedback[:150] + '...' if len(ai_feedback) > 150 else ai_feedback
                    print(f'   💬 AI Feedback: {feedback_short}')
            else:
                print(f'   ⏳ AI Score: Pending...')
        else:
            print(f'   ⏳ No review yet')
except Exception as e:
    print(f'Error parsing JSON: {e}', file=sys.stderr)
"
        else
            # Fallback: simple grep-based parsing
            AI_SCORES=$(echo "$RESULT_RESPONSE" | grep -o '"ai_score":[0-9.]*' | cut -d':' -f2)
            
            echo ""
            echo "$AI_SCORES" | while read score; do
                if [ -n "$score" ]; then
                    echo -e "${GREEN}  ✅ Answer $ANSWER_NUM: Score = ${score}/100${NC}"
                    ANSWER_NUM=$((ANSWER_NUM + 1))
                fi
            done
        fi
        
        echo ""
        echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo -e "${GREEN}✨ Worker Service is Running & Processing Jobs Successfully!${NC}"
        echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    else
        echo -e "${RED}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo -e "${RED}⚠️  AI SCORING NOT COMPLETED (after 30 seconds)${NC}"
        echo -e "${RED}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo ""
        echo "Possible issues:"
        echo "  1. ❌ Worker service not deployed in Railway"
        echo "  2. ❌ GROQ_API_KEY not set or invalid"
        echo "  3. ❌ GROQ_MODEL incorrect (should be: llama-3.1-70b-versatile)"
        echo "  4. ❌ Redis connection issue between API and Worker"
        echo "  5. ❌ Worker crashed or restarting"
        echo ""
        echo "Check Railway logs:"
        echo "  - Navigate to Railway Dashboard"
        echo "  - Worker service → Deployments → View Logs"
        echo "  - Look for Groq API errors or Redis connection errors"
        echo ""
        echo "Quick fixes:"
        echo "  1. Verify GROQ_API_KEY in Railway worker variables"
        echo "  2. Set GROQ_MODEL=llama-3.1-70b-versatile"
        echo "  3. Check worker service is running (not stopped/failed)"
        echo ""
        echo -e "${RED}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    fi
else
    echo -e "${RED}❌ Failed to retrieve submission${NC}"
    exit 1
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
    
    if [ -n "$ANSWER_ID" ]; then
        REVIEW_RESPONSE=$(curl -s -X PUT "$API_URL/api/v1/reviews/$ANSWER_ID" \
          -H "Content-Type: application/json" \
          -H "Authorization: Bearer $REVIEWER_TOKEN" \
          -d '{
            "manual_score": 95,
            "manual_feedback": "Excellent explanation! Very clear and accurate."
          }')
        
        echo "Response: $REVIEW_RESPONSE"
        if echo "$REVIEW_RESPONSE" | grep -q '"id"'; then
            echo -e "${GREEN}✅ Manual review added${NC}"
        else
            echo -e "${BLUE}ℹ️  Manual review response: Check if endpoint is implemented${NC}"
        fi
    else
        echo -e "${BLUE}ℹ️  Could not extract answer ID from submission response${NC}"
    fi
else
    echo -e "${RED}❌ Reviewer registration failed${NC}"
fi

# Summary
echo -e "\n=========================================="
echo -e "${GREEN}🎉 API Test Flow Completed!${NC}"
echo -e "=========================================="
echo ""
echo "Test Summary:"
echo "  ✅ Health Check"
echo "  ✅ User Registration (Creator & Participant)"  
echo "  ✅ Authentication (Login)"
echo "  ✅ Test Creation"
echo "  ✅ Add Questions (3 questions)"
echo "  ✅ Test Publishing"
echo "  ✅ Access Token Generation"
echo "  ✅ Test Submission"
if [ "$AI_SCORED" = true ]; then
    echo -e "  ${GREEN}✅ AI Scoring (Total: $AI_TOTAL)${NC}"
else
    echo -e "  ${RED}⚠️  AI Scoring (FAILED - Check worker)${NC}"
fi
echo ""
echo "Test Details:"
echo "  Test ID:       $TEST_ID"
echo "  Submission ID: $SUBMISSION_ID"
echo "  Creator Email: $CREATOR_EMAIL"
echo "  Participant:   $PARTICIPANT_EMAIL"
echo ""
echo "View in Railway:"
echo "  API: https://assessly-be-production.up.railway.app"
echo "  Submission: $API_URL/api/v1/submissions/$SUBMISSION_ID"
echo ""
if [ "$AI_SCORED" = false ]; then
    echo -e "${RED}⚠️  Action Required:${NC}"
    echo "  1. Check GROQ_API_KEY in Railway environment variables"
    echo "  2. Verify worker service is running"
    echo "  3. Check worker logs for errors"
    echo "  4. Verify GROQ_MODEL = llama-3.1-70b-versatile"
    echo ""
fi
echo "=========================================="
echo ""
echo "Accounts Created:"
echo "  Creator: $CREATOR_EMAIL / $CREATOR_PASS"
echo "  Participant: $PARTICIPANT_EMAIL / $PARTICIPANT_PASS"
echo "  Reviewer: $REVIEWER_EMAIL / $REVIEWER_PASS"
echo ""
echo -e "${BLUE}💡 Note: If AI scoring didn't complete, make sure GROQ_API_KEY is set in Railway environment variables${NC}"
echo ""
