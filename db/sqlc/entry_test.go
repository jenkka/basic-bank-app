package db

import (
	"context"
	"database/sql"
	"testing"

	"github.com/jenkka/dummy-bank/util"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func randomEntryParams(accountID int64) CreateEntryParams {
	return CreateEntryParams{
		AccountID: accountID,
		Amount:    decimal.NewFromInt(util.RandomMoney()),
	}
}

func createRandomEntry(t *testing.T, accountID int64) Entry {
	entryParams := randomEntryParams(accountID)
	entry, err := testQueries.CreateEntry(context.Background(), entryParams)
	require.NoError(t, err)
	require.NotEmpty(t, entry)

	return entry
}

func TestCreateEntry(t *testing.T) {
	account := createRandomAccount(t)
	entryParams := randomEntryParams(account.ID)
	entry, err := testQueries.CreateEntry(context.Background(), entryParams)

	require.NoError(t, err)
	require.NotEmpty(t, entry)
	require.NotZero(t, entry.ID)
	require.NotZero(t, entry.CreatedAt)
	require.Equal(t, entryParams.AccountID, entry.AccountID)
	requireDecimalEqual(t, entryParams.Amount, entry.Amount)
}

func TestGetEntry(t *testing.T) {
	account := createRandomAccount(t)
	createdEntry := createRandomEntry(t, account.ID)

	entry, err := testQueries.GetEntry(context.Background(), createdEntry.ID)
	require.NoError(t, err)
	require.NotEmpty(t, entry)
	require.Equal(t, createdEntry.ID, entry.ID)
	require.Equal(t, createdEntry.AccountID, entry.AccountID)
	requireDecimalEqual(t, createdEntry.Amount, entry.Amount)
	require.Equal(t, createdEntry.CreatedAt, entry.CreatedAt)
}

func TestDeleteEntry(t *testing.T) {
	account := createRandomAccount(t)
	createdEntry := createRandomEntry(t, account.ID)

	_, err := testQueries.DeleteEntry(context.Background(), createdEntry.ID)
	require.NoError(t, err)

	_, err = testQueries.GetEntry(context.Background(), createdEntry.ID)
	require.EqualError(t, err, sql.ErrNoRows.Error())
}

func TestListEntries(t *testing.T) {
	account := createRandomAccount(t)
	for i := 0; i < 10; i++ {
		createRandomEntry(t, account.ID)
	}

	params := ListEntriesParams{
		AccountID: account.ID,
		Limit:     5,
		Offset:    5,
	}

	entries, err := testQueries.ListEntries(context.Background(), params)
	require.NoError(t, err)
	require.Len(t, entries, 5)

	for _, entry := range entries {
		require.NotEmpty(t, entry)
		require.Equal(t, account.ID, entry.AccountID)
	}
}
