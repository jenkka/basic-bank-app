package db

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

func randomTransferParams(fromAccountID, toAccountID int64) CreateTransferParams {
	return CreateTransferParams{
		FromAccountID: fromAccountID,
		ToAccountID:   toAccountID,
		Amount:        fmt.Sprintf("%d", rand.Intn(1000)),
	}
}

func createRandomTransfer(t *testing.T, fromAccountID, toAccountID int64) Transfer {
	transferParams := randomTransferParams(fromAccountID, toAccountID)
	transfer, err := testQueries.CreateTransfer(context.Background(), transferParams)
	require.NoError(t, err)
	require.NotEmpty(t, transfer)

	return transfer
}

func TestCreateTransfer(t *testing.T) {
	fromAccount := createRandomAccount(t)
	toAccount := createRandomAccount(t)
	transferParams := randomTransferParams(fromAccount.ID, toAccount.ID)
	transfer, err := testQueries.CreateTransfer(context.Background(), transferParams)

	require.NoError(t, err)
	require.NotEmpty(t, transfer)
	require.NotZero(t, transfer.ID)
	require.NotZero(t, transfer.CreatedAt)
	require.Equal(t, transferParams.FromAccountID, transfer.FromAccountID)
	require.Equal(t, transferParams.ToAccountID, transfer.ToAccountID)
	require.Equal(t, transferParams.Amount, transfer.Amount)
}

func TestGetTransfer(t *testing.T) {
	fromAccount := createRandomAccount(t)
	toAccount := createRandomAccount(t)
	createdTransfer := createRandomTransfer(t, fromAccount.ID, toAccount.ID)

	transfer, err := testQueries.GetTransfer(context.Background(), createdTransfer.ID)
	require.NoError(t, err)
	require.NotEmpty(t, transfer)
	require.Equal(t, createdTransfer.ID, transfer.ID)
	require.Equal(t, createdTransfer.FromAccountID, transfer.FromAccountID)
	require.Equal(t, createdTransfer.ToAccountID, transfer.ToAccountID)
	require.Equal(t, createdTransfer.Amount, transfer.Amount)
	require.Equal(t, createdTransfer.CreatedAt, transfer.CreatedAt)
}

func TestDeleteTransfer(t *testing.T) {
	fromAccount := createRandomAccount(t)
	toAccount := createRandomAccount(t)
	createdTransfer := createRandomTransfer(t, fromAccount.ID, toAccount.ID)

	err := testQueries.DeleteTransfer(context.Background(), createdTransfer.ID)
	require.NoError(t, err)

	_, err = testQueries.GetTransfer(context.Background(), createdTransfer.ID)
	require.EqualError(t, err, sql.ErrNoRows.Error())
}

func TestListTransfers(t *testing.T) {
	fromAccount := createRandomAccount(t)
	toAccount := createRandomAccount(t)
	for i := 0; i < 10; i++ {
		createRandomTransfer(t, fromAccount.ID, toAccount.ID)
	}

	params := ListTransfersParams{
		FromAccountID: fromAccount.ID,
		ToAccountID:   toAccount.ID,
		Limit:         5,
		Offset:        5,
	}

	transfers, err := testQueries.ListTransfers(context.Background(), params)
	require.NoError(t, err)
	require.Len(t, transfers, 5)

	for _, transfer := range transfers {
		require.NotEmpty(t, transfer)
		require.Equal(t, fromAccount.ID, transfer.FromAccountID)
		require.Equal(t, toAccount.ID, transfer.ToAccountID)
	}
}
