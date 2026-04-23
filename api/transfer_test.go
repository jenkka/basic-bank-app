package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	mockdb "github.com/jenkka/basic-bank-app/db/mock"
	db "github.com/jenkka/basic-bank-app/db/sqlc"
	"github.com/jenkka/basic-bank-app/util"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestCreateTransferAPI(t *testing.T) {
	currency := util.RandomCurrency()
	fromAccount := db.Account{
		ID:        util.RandomInt(1, 1000),
		Owner:     util.RandomUsername(),
		Balance:   decimal.NewFromInt(util.RandomMoney()),
		Currency:  currency,
		CreatedAt: time.Now(),
	}
	toAccount := db.Account{
		ID:        util.RandomInt(1001, 2000),
		Owner:     util.RandomUsername(),
		Balance:   decimal.NewFromInt(util.RandomMoney()),
		Currency:  currency,
		CreatedAt: time.Now(),
	}
	amount := decimal.NewFromInt(util.RandomMoney())
	otherCurrency := util.USD
	if currency == util.USD {
		otherCurrency = util.EUR
	}

	testCases := []struct {
		name          string
		body          map[string]any
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			body: map[string]any{
				"from_account_id": fromAccount.ID,
				"to_account_id":   toAccount.ID,
				"amount":          amount.String(),
				"currency":        currency,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(fromAccount.ID)).
					Times(1).
					Return(fromAccount, nil)
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(toAccount.ID)).
					Times(1).
					Return(toAccount, nil)

				store.EXPECT().
					TransferTxn(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.TransferTxnRes{
						Transfer: db.Transfer{
							ID:            1,
							FromAccountID: fromAccount.ID,
							ToAccountID:   toAccount.ID,
							Amount:        amount,
						},
					}, nil)
			},
			checkResponse: func(
				t *testing.T, recorder *httptest.ResponseRecorder,
			) {
				require.Equal(t, http.StatusCreated, recorder.Code)
			},
		},
		{
			name: "MissingCurrency",
			body: map[string]any{
				"from_account_id": fromAccount.ID,
				"to_account_id":   toAccount.ID,
				"amount":          "50",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Any()).Times(0)
				store.EXPECT().TransferTxn(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(
				t *testing.T, recorder *httptest.ResponseRecorder,
			) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "InvalidAmount_Negative",
			body: map[string]any{
				"from_account_id": fromAccount.ID,
				"to_account_id":   toAccount.ID,
				"amount":          "-10",
				"currency":        currency,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Any()).Times(0)
				store.EXPECT().TransferTxn(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(
				t *testing.T, recorder *httptest.ResponseRecorder,
			) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "InvalidAmount_NonNumeric",
			body: map[string]any{
				"from_account_id": fromAccount.ID,
				"to_account_id":   toAccount.ID,
				"amount":          "abc",
				"currency":        currency,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Any()).Times(0)
				store.EXPECT().TransferTxn(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(
				t *testing.T, recorder *httptest.ResponseRecorder,
			) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "InvalidAmount_Zero",
			body: map[string]any{
				"from_account_id": fromAccount.ID,
				"to_account_id":   toAccount.ID,
				"amount":          "0",
				"currency":        currency,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Any()).Times(0)
				store.EXPECT().TransferTxn(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(
				t *testing.T, recorder *httptest.ResponseRecorder,
			) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "FromAccountNotFound",
			body: map[string]any{
				"from_account_id": fromAccount.ID,
				"to_account_id":   toAccount.ID,
				"amount":          "50",
				"currency":        currency,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(fromAccount.ID)).
					Times(1).
					Return(db.Account{}, sql.ErrNoRows)
				store.EXPECT().TransferTxn(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(
				t *testing.T, recorder *httptest.ResponseRecorder,
			) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name: "ToAccountNotFound",
			body: map[string]any{
				"from_account_id": fromAccount.ID,
				"to_account_id":   toAccount.ID,
				"amount":          "50",
				"currency":        currency,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(fromAccount.ID)).
					Times(1).
					Return(fromAccount, nil)
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(toAccount.ID)).
					Times(1).
					Return(db.Account{}, sql.ErrNoRows)
				store.EXPECT().TransferTxn(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(
				t *testing.T, recorder *httptest.ResponseRecorder,
			) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name: "FromAccountCurrencyMismatch",
			body: map[string]any{
				"from_account_id": fromAccount.ID,
				"to_account_id":   toAccount.ID,
				"amount":          "50",
				"currency":        otherCurrency,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(fromAccount.ID)).
					Times(1).
					Return(fromAccount, nil)
				store.EXPECT().TransferTxn(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(
				t *testing.T, recorder *httptest.ResponseRecorder,
			) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "ToAccountCurrencyMismatch",
			body: map[string]any{
				"from_account_id": fromAccount.ID,
				"to_account_id":   toAccount.ID,
				"amount":          "50",
				"currency":        currency,
			},
			buildStubs: func(store *mockdb.MockStore) {
				mismatchedToAccount := toAccount
				mismatchedToAccount.Currency = otherCurrency

				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(fromAccount.ID)).
					Times(1).
					Return(fromAccount, nil)
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(toAccount.ID)).
					Times(1).
					Return(mismatchedToAccount, nil)
				store.EXPECT().TransferTxn(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(
				t *testing.T, recorder *httptest.ResponseRecorder,
			) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "GetAccountInternalError",
			body: map[string]any{
				"from_account_id": fromAccount.ID,
				"to_account_id":   toAccount.ID,
				"amount":          "50",
				"currency":        currency,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(fromAccount.ID)).
					Times(1).
					Return(db.Account{}, sql.ErrConnDone)
				store.EXPECT().TransferTxn(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(
				t *testing.T, recorder *httptest.ResponseRecorder,
			) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "TransferTxnInternalError",
			body: map[string]any{
				"from_account_id": fromAccount.ID,
				"to_account_id":   toAccount.ID,
				"amount":          "50",
				"currency":        currency,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(fromAccount.ID)).
					Times(1).
					Return(fromAccount, nil)
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(toAccount.ID)).
					Times(1).
					Return(toAccount, nil)
				store.EXPECT().
					TransferTxn(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.TransferTxnRes{}, sql.ErrConnDone)
			},
			checkResponse: func(
				t *testing.T, recorder *httptest.ResponseRecorder,
			) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			tc.buildStubs(store)

			server := newTestServer(t, store)
			recorder := httptest.NewRecorder()

			body, err := json.Marshal(tc.body)
			require.NoError(t, err)

			request, err := http.NewRequest(
				http.MethodPost, "/transfers", bytes.NewReader(body),
			)
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}
