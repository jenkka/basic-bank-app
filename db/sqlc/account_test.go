package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/jenkka/basic-bank-app/util"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func randomAccountParams(username string) CreateAccountParams {
	return CreateAccountParams{
		Owner:    username,
		Balance:  decimal.NewFromInt(util.RandomMoney()),
		Currency: util.RandomCurrency(),
	}
}

func createRandomAccount(t *testing.T) Account {
	user := createRandomUser(t)
	accParams := randomAccountParams(user.Username)
	account, err := testQueries.CreateAccount(context.Background(), accParams)
	require.NoError(t, err)
	require.NotEmpty(t, account)

	return account
}

func TestCreateAccount(t *testing.T) {
	user := createRandomUser(t)
	accParams := randomAccountParams(user.Username)
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
	createdAccount := createRandomAccount(t)

	getAccount, err := testQueries.GetAccount(
		context.Background(), createdAccount.ID,
	)
	require.NoError(t, err)
	require.NotEmpty(t, getAccount)
	require.NotZero(t, getAccount.ID)
	require.Equal(t, createdAccount.Owner, getAccount.Owner)
	requireDecimalEqual(t, createdAccount.Balance, getAccount.Balance)
	require.Equal(t, createdAccount.Currency, getAccount.Currency)
	require.WithinDuration(
		t, createdAccount.CreatedAt, getAccount.CreatedAt, time.Second,
	)
}

func TestUpdateAccount(t *testing.T) {
	createdAccount := createRandomAccount(t)

	updateAccParams := UpdateAccountParams{
		ID:      createdAccount.ID,
		Balance: decimal.NewFromFloat(15.11),
	}
	_, err := testQueries.UpdateAccount(context.Background(), updateAccParams)
	require.NoError(t, err)

	updatedAccount, err := testQueries.GetAccount(
		context.Background(), createdAccount.ID,
	)
	require.NoError(t, err)
	require.NotEmpty(t, updatedAccount)
	requireDecimalEqual(t, updateAccParams.Balance, updatedAccount.Balance)
}

func TestDeleteAccount(t *testing.T) {
	createdAccount := createRandomAccount(t)

	_, err := testQueries.DeleteAccount(context.Background(), createdAccount.ID)
	require.NoError(t, err)

	_, err = testQueries.GetAccount(context.Background(), createdAccount.ID)
	require.EqualError(t, err, sql.ErrNoRows.Error())
}

func TestListAccounts(t *testing.T) {
	user := createRandomUser(t)

	for _, currency := range util.SupportedCurrencies {
		_, err := testQueries.CreateAccount(
			context.Background(),
			CreateAccountParams{
				Owner:    user.Username,
				Balance:  decimal.NewFromInt(util.RandomMoney()),
				Currency: currency,
			},
		)
		require.NoError(t, err)
	}

	// Create some random accounts to introduce noise
	for i := 0; i < 5; i++ {
		createRandomAccount(t)
	}

	params := ListAccountsParams{
		Owner:  user.Username,
		Limit:  5,
		Offset: 5,
	}

	accounts, err := testQueries.ListAccounts(context.Background(), params)
	require.NoError(t, err)
	require.Len(t, accounts, 5)

	for _, account := range accounts {
		require.NotEmpty(t, account)
		require.Equal(t, user.Username, account.Owner)
	}
}
