package db

import (
	"database/sql"
	"log"
	"os"
	"testing"

	"github.com/jenkka/dummy-bank/util"
	_ "github.com/lib/pq"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

var testQueries *Queries
var testDB *sql.DB

func requireDecimalEqual(t *testing.T, expected, actual decimal.Decimal) {
	t.Helper()
	require.True(
		t, expected.Equal(actual),
		"expected %s, got %s", expected, actual,
	)
}

func TestMain(m *testing.M) {
	config, err := util.LoadConfig("../..")
	if err != nil {
		log.Fatal("Failed to load config file:", err)
	}

	testDB, err = sql.Open(config.DbDriver, config.DbSource)
	if err != nil {
		log.Fatal("Failed to connect to the database:", err)
	}
	defer testDB.Close()

	testQueries = New(testDB)

	os.Exit(m.Run())
}
