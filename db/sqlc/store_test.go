package db

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func init() {
	var err error
	if testDB == nil {
		testDB, err = sql.Open(dbDriver, dbSource)
		fmt.Println("SQL conn err:", err)
	}
}

// Test if transfer transaction between accounts supports concurrency
func TestTransferTx(t *testing.T) {
	store := NewStore(testDB)
	account1 := createRandomAccount(t)
	account2 := createRandomAccount(t)
	fmt.Println(">> before:", account1.Balance, account2.Balance)

	// run n concurrent transaction
	n := 5
	amount := int64(10)

	errs := make(chan error)
	results := make(chan TransferTxResult)

	// use go routine to execute transfers
	for i := 0; i < n; i++ {
		go func() {
			result, err := store.TransferTx(context.Background(), TransferTxParams{
				FromAccountID: account1.ID,
				ToAccountID:   account2.ID,
				Amount:        amount,
			})

			errs <- err
			results <- result
		}()
	}

	for i := 0; i < n; i++ {
		err := <-errs
		require.NoError(t, err)

		result := <-results
		require.NotEmpty(t, result)
		transfer := result.Transfer
		require.NotEmpty(t, transfer)
		require.Equal(t, account1.ID, transfer.FromAccountID)
		require.Equal(t, account2.ID, transfer.ToAccountID)
		require.Equal(t, amount, transfer.Amount)
		require.NotZero(t, transfer.ID)
		require.NotZero(t, transfer.CreatedAt)

		// make sure transfer record is actually created in database
		_, err = store.GetTransfer(context.Background(), transfer.ID)
		require.NoError(t, err)

		// check entries in database
		fromEntry := result.FromEntry
		toEntry := result.ToEntry
		require.NotEmpty(t, fromEntry)
		require.NotEmpty(t, toEntry)
		require.Equal(t, account1.ID, fromEntry.AccountID)
		require.Equal(t, account2.ID, toEntry.AccountID)
		require.Equal(t, -amount, fromEntry.Amount)
		require.Equal(t, amount, toEntry.Amount)

		require.NotZero(t, fromEntry.ID)
		require.NotZero(t, toEntry.ID)
		require.NotZero(t, fromEntry.CreatedAt)
		require.NotZero(t, toEntry.CreatedAt)

		_, err = store.GetEntry(context.Background(), fromEntry.ID)
		require.NoError(t, err)
		_, err = store.GetEntry(context.Background(), toEntry.ID)
		require.NoError(t, err)
	}
}
