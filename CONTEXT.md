# CapitalFlow

CapitalFlow is a private ledger for accounts, transactions, transfers, and interest-bearing balances.

## Language

**Legacy deposit snapshot**:
A read-only JSON export produced by the predecessor deposit tracker. It is migration input, never a runtime ledger.
_Avoid_: JSON ledger, deposit database

**Legacy import**:
A one-way conversion from a legacy deposit snapshot into the PostgreSQL ledger. Imported data is subsequently owned only by the PostgreSQL ledger.
_Avoid_: JSON sync, dual storage

**Dashboard report**:
A read-only projection of ledger balances, cashflow, goals, and limits. It derives state from the PostgreSQL ledger and never owns financial data.
_Avoid_: dashboard state, dashboard ledger

**Financial goal**:
A user-owned savings target tied to one account. Its currency follows that account, while its status records whether the target is active, completed, or archived.
_Avoid_: unlinked goal, goal balance

**Category**:
A shared classification for ledger transactions.
_Avoid_: user category

**Category limit**:
A user-owned spending threshold for one category and currency. It reports progress but does not block transactions.
_Avoid_: budget enforcement, transaction cap
