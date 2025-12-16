

## Assumption:
1. amount less than 1 paisa is rounded down.
Explanation:
If A pays rs 100 for a group of 3, split equally.
B & C have to pay Rs 33.33 each. (A has to bear Rs 33.34)
This would be notisable for a very large group, difference is added to first user for simplicity. 

2. Application recives more read traffic than write traffic.

3. Overall outstanding balance = amout that has to be received - amount that needs to be payed.

## Non-functional requirements assumptions
- **Consistency**: strongly consistent;
- **Availability**: Single-region deployment with a single primary DB; tolerate brief maintenance windows (no multi-region active-active requirement).
- **Latency**: Read-heavy endpoints (e.g., "get my balances") should be fast and served via indexed lookups; history endpoints may be slower and paginated.
- **Data retention**: Ledger data is retained long-term; older history may be archived/partitioned as data grows.
- **Scale expectation**: Users create ~3–4 expenses/day; typical expense has 2–4 participants; reads significantly outnumber writes.
- **Daily active transacting users**: Assume ~100K transacting users/day create at least one expense on a given day (order-of-magnitude load driver for writes).


## Testing:
Postman collection is added in Resources folder. 


## DB Schema
[Database Schema](db/schema.md)


## Steps to run

1.  **Install dependencies**: (go 1.25)
    ```bash
    go mod tidy
    ```
2.  **Run database in docker**:
    ```bash
    docker-compose up -d mysql --remove-orphans
    ```
    *Note: Replace the database connection string with your actual database URL.*
3.  **Run the application**:
    ```bash
    go run cmd/server/main.go
    ```
