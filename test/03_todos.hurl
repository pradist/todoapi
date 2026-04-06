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
