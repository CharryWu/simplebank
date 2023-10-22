package db

import (
	"context"
	"database/sql"
	"fmt"
)

// Extend functionality of db.go to provide transactions,
// which may consist of multiple queries to be executed at once
type Store struct {
	*Queries // embedding queries inside store
	db       *sql.DB
}

// Create new store
func NewStore(db *sql.DB) *Store {
	return &Store{
		db:      db,
		Queries: New(db),
	}
}

// executes a function within a db transaction
// commit or rollback the transaction based on the error returned by the function
// unexported, only the specific transaction type exec function will be exported
func (store *Store) execTx(ctx context.Context, fn func(*Queries) error) error {
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	q := New(tx)
	err = fn(q)

	if err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil { // even rolling back transaction has error
			return fmt.Errorf("tx err: %v, rb err: %v", err, rollbackErr)
		}
		return err
	}

	return tx.Commit()
}

// Contains input params of the transfer transaction
type TransferTxParams struct {
	FromAccountID int64 `json:"from_account_id"`
	ToAccountID   int64 `json:"to_account_id"`
	Amount        int64 `json:"amount"`
}

// result of transaction
type TransferTxResult struct {
	Transfer    Transfer `json:"transfer"`
	FromAccount Account  `json:"from_account"`
	ToAccount   Account  `json:"to_account"`
	FromEntry   Entry    `json:"from_entry"` // entry in the from account which records the money that is moving out
	ToEntry     Entry    `json:"to_entry"`   // entry in the from account which records the money that is moving out
}

// TransferTx performs a money transfer from one account to the other
// It creates a new transfer record, add new account entries, and update account balance within a single transaction
func (store *Store) TransferTx(ctx context.Context, arg TransferTxParams) (TransferTxResult, error) {
	var result TransferTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		var err error

		result.Transfer, err = q.CreateTransfer(ctx, CreateTransferParams{
			FromAccountID: arg.FromAccountID,
			ToAccountID:   arg.ToAccountID,
			Amount:        arg.Amount,
		})
		if err != nil {
			return err
		}

		result.FromEntry, err = q.CreateEntry(ctx, CreateEntryParams{
			AccountID: arg.FromAccountID,
			Amount:    -arg.Amount,
		})
		if err != nil {
			return err
		}

		result.ToEntry, err = q.CreateEntry(ctx, CreateEntryParams{
			AccountID: arg.ToAccountID,
			Amount:    arg.Amount,
		})
		if err != nil {
			return err
		}

		// if arg.FromAccountID < arg.ToAccountID {
		// 	result.FromAccount, result.ToAccount, err = addMoney(ctx, q, arg.FromAccountID, -arg.Amount, arg.ToAccountID, arg.Amount)
		// } else {
		// 	result.ToAccount, result.FromAccount, err = addMoney(ctx, q, arg.ToAccountID, arg.Amount, arg.FromAccountID, -arg.Amount)
		// }

		return err
	})

	return result, err
}
