# Valid credentials return a token
POST {{base_url}}/tokenz
Content-Type: application/json
{
  "username": "{{admin_user}}",
  "password": "{{admin_pass}}"
}

HTTP 200
[Asserts]
jsonpath "$.token" isString
jsonpath "$.token" != ""

###

# Wrong password returns 401
POST {{base_url}}/tokenz
Content-Type: application/json
{
  "username": "{{admin_user}}",
  "password": "wrongpassword"
}

HTTP 401
[Asserts]
jsonpath "$.error" == "invalid credentials"

###

# Missing required field returns 400
POST {{base_url}}/tokenz
Content-Type: application/json
{
  "username": "{{admin_user}}"
}

HTTP 400
