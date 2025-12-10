#  Database Schema Documentation

The architecture uses a **SQL (Relational)** database with a ledger approach for financial accuracy and a denormalized summary table for fast read performance.

## 1. Design Philosophy

The schema separates the **Ledger (Source of Truth)** from the **Summary (Performance Cache)**:

1.  **Ledger Tables (`Expenses`, `Expense_Splits`):** These are the atomic facts. They are **append-only** and ensure that a user's final balance can always be recalculated accurately from the beginning of time.
2.  **Summary Table (`Balances`):** This table is updated transactionally with every new expense. It allows the most frequent query ("What is my total debt with User X?") to be answered with a single, fast indexed lookup, avoiding costly table aggregations.


---

## 2. Table Definitions

### 2.1. `Users`

Stores core user profile data.

| Column | Data Type | Constraint/Notes |
| :--- | :--- | :--- |
| **`id`** | `INTEGER` | **Primary Key** (PK) |
| **`name`** | `VARCHAR` | |
| **`email`** | `VARCHAR` | **Unique Index.** Used for login and lookups. |
| **`created_at`** | `TIMESTAMP` | |

### 2.2. `Expenses`

Stores the metadata for a single transaction (e.g., "Dinner," "Rent").

| Column | Data Type | Constraint/Notes |
| :--- | :--- | :--- |
| **`id`** | `INTEGER` | **Primary Key** (PK) |
| **`description`** | `VARCHAR` | E.g., "Lunch at Corner Dhaba" |
| **`total_amount`** | `DECIMAL` | The full cost of the expense. |
| **`created_by`** | `INTEGER` | **Foreign Key** (`Users.id`). The user who recorded the expense. |
| **`created_at`** | `TIMESTAMP` | |

### 2.3. `Expense_Splits` (The Ledger)

The core table where the financial truth is recorded. **Every participant in an expense (payer or debtor) gets one row here.**

| Column | Data Type | Constraint/Notes |
| :--- | :--- | :--- |
| **`id`** | `INTEGER` | **Primary Key** (PK) |
| **`expense_id`** | `INTEGER` | **Foreign Key** (`Expenses.id`). |
| **`user_id`** | `INTEGER` | **Foreign Key** (`Users.id`). **Indexed for fast access.** |
| **`amount_paid`** | `DECIMAL` | The money this user physically contributed (Credit). |
| **`amount_owed`** | `DECIMAL` | The portion of the bill this user is responsible for (Debit). |

**Ledger Calculation:** For any single transaction, the user's **Net Change** is simply `amount_paid - amount_owed`.

### 2.4. `Balances` (The Debt Cache)

This denormalized table stores the **running net debt** between every pair of users. It is designed for extreme read speed.

| Column | Data Type | Constraint/Notes |
| :--- | :--- | :--- |
| **`user1_id`** | `INTEGER` | **Composite PK, FK** (`Users.id`). The user with the lower ID (by convention). |
| **`user2_id`** | `INTEGER` | **Composite PK, FK** (`Users.id`). The user with the higher ID (by convention). |
| **`balance`** | `DECIMAL` | **Net Balance.** If `balance > 0`, `user1` owes `user2`. If `balance < 0`, `user2` owes `user1`. |
| **`last_updated`** | `TIMESTAMP` | |

---

## 3. Indexing Strategy

Indexes are optimized for the frequent read pattern of checking balances and for joining the ledger.

| Table | Index Field(s) | Type | Purpose |
| :--- | :--- | :--- | :--- |
| `Users` | `email` | Unique | Login/Authentication and unique constraint enforcement. |
| `Expense_Splits`| **`user_id`** | **Standard** | **Crucial** for finding *all* transactions involving a specific user quickly. |
| `Expense_Splits`| `(expense_id, user_id)` | Composite | Optimizes joins between `Expenses` and `Expense_Splits`. |
| `Balances` | `(user1_id, user2_id)` | Unique/PK | Ensures fast, single-row lookup for the net debt between any two users. |

---

## 4. Relationships

* `Expenses.created_by` $\rightarrow$ `Users.id`
* `Expense_Splits.expense_id` $\rightarrow$ `Expenses.id` (One expense has many split entries)
* `Expense_Splits.user_id` $\rightarrow$ `Users.id` (Many split entries belong to one user)
* `Balances.user1_id` $\rightarrow$ `Users.id`
* `Balances.user2_id` $\rightarrow$ `Users.id`

***