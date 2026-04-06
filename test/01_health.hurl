# Health check
GET {{base_url}}/ping

HTTP 200
[Asserts]
jsonpath "$.message" == "pong"
