package db

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

func randomAccountParams() CreateAccountParams {
	return CreateAccountParams{
		Owner:    fmt.Sprintf("user_%d", rand.Int()),
		Balance:  fmt.Sprintf("%d", rand.Intn(1000)),
		Currency: "MXN",
	}
}

func createRandomAccount(t *testing.T) Account {
	accParams := randomAccountParams()
	account, err := testQueries.CreateAccount(context.Background(), accParams)
	require.NoError(t, err)
	require.NotEmpty(t, account)

	return account
}

func TestCreateAccount(t *testing.T) {
	accParams := randomAccountParams()
	account, err := testQueries.CreateAccount(context.Background(), accParams)

	require.NoError(t, err)
	require.NotEmpty(t, account)
	require.NotZero(t, account.ID)
	require.NotZero(t, account.CreatedAt)
	require.Equal(t, accParams.Owner, account.Owner)
	require.Equal(t, accParams.Balance, account.Balance)
	require.Equal(t, accParams.Currency, account.Currency)
}

func TestGetAccount(t *testing.T) {
	accParams := randomAccountParams()
	createdAccount, err := testQueries.CreateAccount(context.Background(), accParams)
	require.NoError(t, err)
	require.NotEmpty(t, createdAccount)

	account, err := testQueries.GetAccount(context.Background(), createdAccount.ID)
	require.NoError(t, err)
	require.NotEmpty(t, account)
	require.NotZero(t, account.ID)
	require.NotZero(t, account.CreatedAt)
	require.Equal(t, accParams.Owner, account.Owner)
	require.Equal(t, accParams.Balance, account.Balance)
	require.Equal(t, accParams.Currency, account.Currency)
}

func TestUpdateAccount(t *testing.T) {
	accParams := randomAccountParams()
	createdAccount, err := testQueries.CreateAccount(context.Background(), accParams)
	require.NoError(t, err)
	require.NotEmpty(t, createdAccount)

	updateAccParams := UpdateAccountParams{
		ID:      createdAccount.ID,
		Balance: "15.11",
	}
	err = testQueries.UpdateAccount(context.Background(), updateAccParams)
	require.NoError(t, err)

	updatedAccount, err := testQueries.GetAccount(context.Background(), createdAccount.ID)
	require.NoError(t, err)
	require.NotEmpty(t, updatedAccount)
	require.Equal(t, updateAccParams.Balance, updatedAccount.Balance)
}

func TestDeleteAccount(t *testing.T) {
	accParams := randomAccountParams()

	createdAccount, err := testQueries.CreateAccount(context.Background(), accParams)
	require.NoError(t, err)
	require.NotEmpty(t, createdAccount)

	err = testQueries.DeleteAccount(context.Background(), createdAccount.ID)
	require.NoError(t, err)

	_, err = testQueries.GetAccount(context.Background(), createdAccount.ID)
	require.EqualError(t, err, sql.ErrNoRows.Error())
}

func TestListAccounts(t *testing.T) {
	for i := 0; i < 10; i++ {
		createdAccount, err := testQueries.CreateAccount(context.Background(), randomAccountParams())
		require.NoError(t, err)
		require.NotEmpty(t, createdAccount)
	}

	params := ListAccountsParams{
		Limit:  5,
		Offset: 5,
	}

	accounts, err := testQueries.ListAccounts(context.Background(), params)
	require.NoError(t, err)
	require.Len(t, accounts, 5)

	for _, account := range accounts {
		require.NotEmpty(t, account)
	}
}
