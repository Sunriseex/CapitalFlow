# CapitalFlow

CapitalFlow is a private ledger for accounts, transactions, transfers, and interest-bearing balances.

## Language

**Legacy deposit snapshot**:
A read-only JSON export produced by the predecessor deposit tracker. It is migration input, never a runtime ledger.
_Avoid_: JSON ledger, deposit database

**Legacy import**:
A one-way conversion from a legacy deposit snapshot into the PostgreSQL ledger. Imported data is subsequently owned only by the PostgreSQL ledger.
_Avoid_: JSON sync, dual storage
