# Health check
GET {{base_url}}/healthz

HTTP 200
[Asserts]
jsonpath "$.status" == "ok"
