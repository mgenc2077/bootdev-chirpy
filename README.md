# Bootdev Guided Project Chirpy
## Main Features
Chirpy is a API based social media platform that you can create your user, login then start sending your chirps.

It uses mostly standard packages (net/http, encoding/json...) with a couple of functional ones (like google/uuid and more) to provide more industry standard practices.

As a database it uses a local PostgreSQL database handled migrations with [Goose](https://github.com/pressly/goose) and compiled querry packages with [SQLC](https://github.com/sqlc-dev/sqlc) 

For authentication it has functionality to create JSON Web Token and refresh tokens to authenticate users.

It has a Webhook endpoint for setting a subscription like is_chirpy_red for users (false is default for all users). POLKA_KEY is used as an authentication for Polka provider.
## Folder Structure
### /assets 
static files for the /assets/ endpoint
### /Internal
- /auth

Contains auth package that used for making and validating tokens and related test files.
- /database

SQLC generated query packages for queries
### /sql
- /queries

SQL query files for interacting with database.
- /schema

SQL Migration files for database structured in Goose syntax
## Usage
### Requirements
- Local PostgreSQL configured and connection string noted
- [Goose](https://github.com/pressly/goose)
- [SQLC](https://github.com/sqlc-dev/sqlc) 
### Setup
- cd into sql/schema directory and run goose up to apply latest schema into PostgreSQL
```shell
goose <connection-string> up
```
- Create env file at the repo
```shell
touch .env
```
Required values:
```
DB_URL="postgres://<connection-string>:5432/chirpy?sslmode=disable"
PLATFORM="dev"
jwt_Secret="<jwt-sign-key>"
POLKA_KEY="<polka-key>"
```
- Build and run
```shell
go build -o out && ./out
```
## Endpoints
### /app/
Its an almost empty with just a header. serves index.html at the root of the repo
### /assets/
This endpoint serves static files inside assets folder. There is only one .png file exist so only viable url is /assets/logo.png
### /admin/metrics
Only support one method
- GET

This endpoint has minimal html with the number of times the api called since run.
### /admin/reset
Only support one method
- POST

When called this endpoints resets the database and hit count for the metrics.
### /api/healthz
Only support one method
- GET

This endpoint returns 200 OK when called to indicate server status.
### /api/chirps
Supports two methods
- GET

Returns currently available chirps with optional parameters; author_id for filtering for users, sort for sorting in descending or ascending order ("asc" for ascending "desc" for descending)
Example Return:
```json
[
    {
        "id": "569eabff-f792-47b6-af74-77eac7533eec",
        "created_at": "2025-03-21T15:19:04.553378Z",
        "updated_at": "2025-03-21T15:19:04.553378Z",
        "body": "I'm the one who knocks!",
        "user_id": "557cce37-dcdd-4c50-9ef9-2ad9cf3a31fb"
    },
    {
        "id": "cf3090a0-396d-4178-b61f-cac21a961cba",
        "created_at": "2025-03-21T15:19:04.556643Z",
        "updated_at": "2025-03-21T15:19:04.556643Z",
        "body": "Gale!",
        "user_id": "557cce37-dcdd-4c50-9ef9-2ad9cf3a31fb"
    },
    {
        "id": "61689915-b931-4725-8104-5ee9775ac7eb",
        "created_at": "2025-03-21T15:19:04.757616Z",
        "updated_at": "2025-03-21T15:19:04.757616Z",
        "body": "Mr President....",
        "user_id": "f636fe27-dfcf-460e-b3f8-cfb70151e65d"
    }
]
```

- POST

Creates and saves a chirp. (Requires JWT_token in Authorization header in "Authorization":"Bearer JWT_TOKEN" format)
Expects:
```json
{
  "body": "I'm the one who knocks!"
}
```
Returns:
```json
{
  "id": "<chirpID-As-UUID>",
  "body": "<chirp-body>",
  "created_at": "<creation-time>",
  "updated_at": "<update-time>",
  "user_id": "<user-id-UUID>"
}
```
### /api/users
Supports two methods
- PUT

Changes password for the email in json body (Expects jwt token)
```json
{
  "email": "personal@email.com",
  "password": "12345678"
}
```
- POST

Creates and saves a user in json body
```json
{
  "email": "personal@email.com",
  "password": "123456"
}
```
### /api/login
Supports one method
- POST

Returns jwt token for api access expects:
```json
{
  "email": "walt@breakingbad.com",
  "password": "123456"
}
```

Returns:
```json
{
  "email": "<email>",
  "id": "<uuid-user-id>",
  "is_chirpy_red": false,
  "refresh_token": "<refresh-token>",
  "token": "<jwt-token>",
  "created_at": "<creation-time>",
  "updated_at": "<update-time>"
}
```
### /api/chirps/{chirpID}
Supports two methods
- GET

Returns the user with the chirpID

Returns:
```json
{
  "email": "<email>",
  "id": "<uuid-user-id>",
  "is_chirpy_red": false,
  "refresh_token": "<refresh-token>",
  "token": "<jwt-token>",
  "created_at": "<creation-time>",
  "updated_at": "<update-time>"
}
```

- DELETE

Deletes the posted chirp. Return 204 when successful.

### /api/refresh
Support one method
- POST

Returns JWT_TOKEN from an refresh_token expects refresh token in authorization header

Returns:
```json
{
    "token": "<JWT-Token>"
}
```

### /api/revoke
Support one method
- POST

Revokes a refresh token. Expects refresh token in the authorization header and returns 204 when successful.

### /api/polka/webhooks
Support one method
- POST

Upgrades an existing user to Chirpy Red. Returns 204 when successful. 

Expects:
```json
{
  "event": "user.upgraded",
  "data": {
    "user_id": "3311741c-680c-4546-99f3-fc9efac2036c"
  }
}
```

