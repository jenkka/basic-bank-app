package db

import (
	"context"
	"database/sql"
	"testing"

	"github.com/jenkka/basic-bank-app/util"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func randomAccountParams() CreateAccountParams {
	return CreateAccountParams{
		Owner:    util.RandomOwner(),
		Balance:  decimal.NewFromInt(util.RandomMoney()),
		Currency: util.RandomCurrency(),
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
	requireDecimalEqual(t, accParams.Balance, account.Balance)
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
	requireDecimalEqual(t, accParams.Balance, account.Balance)
	require.Equal(t, accParams.Currency, account.Currency)
}

func TestUpdateAccount(t *testing.T) {
	accParams := randomAccountParams()
	createdAccount, err := testQueries.CreateAccount(context.Background(), accParams)
	require.NoError(t, err)
	require.NotEmpty(t, createdAccount)

	updateAccParams := UpdateAccountParams{
		ID:      createdAccount.ID,
		Balance: decimal.NewFromFloat(15.11),
	}
	_, err = testQueries.UpdateAccount(context.Background(), updateAccParams)
	require.NoError(t, err)

	updatedAccount, err := testQueries.GetAccount(context.Background(), createdAccount.ID)
	require.NoError(t, err)
	require.NotEmpty(t, updatedAccount)
	requireDecimalEqual(t, updateAccParams.Balance, updatedAccount.Balance)
}

func TestDeleteAccount(t *testing.T) {
	accParams := randomAccountParams()

	createdAccount, err := testQueries.CreateAccount(context.Background(), accParams)
	require.NoError(t, err)
	require.NotEmpty(t, createdAccount)

	_, err = testQueries.DeleteAccount(context.Background(), createdAccount.ID)
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
