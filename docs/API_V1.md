# Inventory Optimizer REST API - v1

All API endpoints are prefixed with `/api/v1` and return JSON responses.

### Authentication

The API uses standard JWT Bearer tokens for authentication. Include the access token in the `Authorization` header of all protected requests:

```http
Authorization: Bearer <access_token>
```

---

## Endpoints

### 1. Register
Create a new user account.

**Request**
```bash
curl -X POST http://localhost:3000/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "securepassword123"}'
```

**Response (201 Created)**
```json
{
  "access_token": "eyJhb...",
  "refresh_token": "eyJhb...",
  "expires_in": 3600
}
```

### 2. Login
Authenticate an existing user.

**Request**
```bash
curl -X POST http://localhost:3000/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "securepassword123"}'
```

**Response (200 OK)**
```json
{
  "access_token": "eyJhb...",
  "refresh_token": "eyJhb...",
  "expires_in": 3600
}
```

### 3. Run Analysis
Run an inventory analysis. Returns full results and saves a report if authenticated.

**Request (Multipart Form Data)**
```bash
curl -X POST http://localhost:3000/api/v1/analyze \
  -H "Authorization: Bearer <access_token>" \
  -F "title=Q2 Inventory Overview" \
  -F "sales_file=@data/sample_csv/sales_history.csv" \
  -F "params_file=@data/sample_csv/sku_parameters.csv"
```

**Response (200 OK)**
```json
{
  "skus_analyzed": 3,
  "warnings": [],
  "elapsed_ms": 45,
  "saved_report_id": "8bb3d...",
  "results": [
    {
      "sku": "SKU001",
      "reorder_point": 70,
      "eoq": 186
      // ... full calculations
    }
  ]
}
```

### 4. List Reports
Fetch all saved reports for the authenticated user.

**Request**
```bash
curl -H "Authorization: Bearer <access_token>" http://localhost:3000/api/v1/reports
```

**Response (200 OK)**
```json
{
  "reports": [
    {
      "id": "8bb3d...",
      "title": "Q2 Inventory Overview",
      "created_at": "2026-05-05T12:00:00Z"
    }
  ]
}
```

### 5. Get Report Detail
Fetch the full results of a specific report.

**Request**
```bash
curl -H "Authorization: Bearer <access_token>" http://localhost:3000/api/v1/reports/<report_id>
```

**Response (200 OK)**
```json
{
  "id": "8bb3d...",
  "title": "Q2 Inventory Overview",
  "results": [ ... ]
}
```

### 6. Delete Report
Delete a saved report.

**Request**
```bash
curl -X DELETE -H "Authorization: Bearer <access_token>" http://localhost:3000/api/v1/reports/<report_id>
```

**Response (200 OK)**
```json
{
  "message": "deleted"
}
```

---
