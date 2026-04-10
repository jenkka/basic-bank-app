package db

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTransferTxn(t *testing.T) {
	store := NewStore(testDB)
	account1 := createRandomAccount(t)
	account2 := createRandomAccount(t)

	nTransfers := 5
	params := CreateTransferParams{
		FromAccountID: account1.ID,
		ToAccountID:   account2.ID,
		Amount:        "10.0",
	}
	errors := make(chan error)
	results := make(chan TransferTxnRes)

	for i := 0; i < nTransfers; i++ {
		go func() {
			res, err := store.TransferTxn(context.Background(), params)
			errors <- err
			results <- res
		}()
	}

	for i := 0; i < nTransfers; i++ {
		require.NoError(t, <-errors)

		res := <-results

		transfer := res.Transfer
		require.NotEmpty(t, transfer.ID)
		require.NotEmpty(t, transfer.CreatedAt)
		require.Equal(t, params.Amount, transfer.Amount)
		require.Equal(t, params.FromAccountID, transfer.FromAccountID)
		require.Equal(t, params.ToAccountID, transfer.ToAccountID)

		_, err := testQueries.GetTransfer(context.Background(), transfer.ID)
		require.NoError(t, err)

		fromEntry := res.FromEntry
		require.NotEmpty(t, fromEntry.ID)
		require.NotEmpty(t, fromEntry.CreatedAt)
		require.Equal(t, fmt.Sprintf("-%s", params.Amount), fromEntry.Amount)
		require.Equal(t, params.FromAccountID, fromEntry.AccountID)

		_, err = testQueries.GetEntry(context.Background(), fromEntry.ID)
		require.NoError(t, err)

		toEntry := res.ToEntry
		require.NotEmpty(t, toEntry.ID)
		require.NotEmpty(t, toEntry.CreatedAt)
		require.Equal(t, params.Amount, toEntry.Amount)
		require.Equal(t, params.ToAccountID, toEntry.AccountID)

		_, err = testQueries.GetEntry(context.Background(), toEntry.ID)
		require.NoError(t, err)

		// TODO: Test accounts' balance
	}
}
