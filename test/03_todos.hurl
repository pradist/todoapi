# Step 1: Login and capture token
POST {{base_url}}/tokenz
Content-Type: application/json
{
  "username": "{{admin_user}}",
  "password": "{{admin_pass}}"
}

HTTP 200
[Captures]
token: jsonpath "$.token"

###

# Step 2: Create todo with valid token returns 201
POST {{base_url}}/todos
Authorization: Bearer {{token}}
Content-Type: application/json
{
  "text": "integration test item"
}

HTTP 201
[Asserts]
jsonpath "$.ID" isInteger

###

# Step 3: Create todo without token returns 401
POST {{base_url}}/todos
Content-Type: application/json
{
  "text": "should be rejected"
}

HTTP 401

###

# Step 4: Create todo with invalid token returns 401
POST {{base_url}}/todos
Authorization: Bearer invalid.token.here
Content-Type: application/json
{
  "text": "should also be rejected"
}

HTTP 401
