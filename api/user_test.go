package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	mockdb "github.com/jenkka/basic-bank-app/db/mock"
	db "github.com/jenkka/basic-bank-app/db/sqlc"
	"github.com/jenkka/basic-bank-app/util"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type eqCreateUserParamsMatcher struct {
	arg      db.CreateUserParams
	password string
}

func (e eqCreateUserParamsMatcher) Matches(x any) bool {
	arg, ok := x.(db.CreateUserParams)
	if !ok {
		return false
	}

	if !util.CheckPassword(e.password, arg.HashedPwd) {
		return false
	}

	e.arg.HashedPwd = arg.HashedPwd
	return reflect.DeepEqual(e.arg, arg)
}

func (e eqCreateUserParamsMatcher) String() string {
	return fmt.Sprintf("matches arg %v and password %v", e.arg, e.password)
}

func eqCreateUserParams(
	arg db.CreateUserParams, password string,
) gomock.Matcher {
	return eqCreateUserParamsMatcher{arg: arg, password: password}
}

func randomUser(t *testing.T) (db.User, string) {
	password := util.RandomString(8)
	hashedPassword, err := util.HashPassword(password)
	require.NoError(t, err)

	user := db.User{
		Username:     util.RandomUsername(),
		HashedPwd:    hashedPassword,
		Email:        util.RandomEmail(),
		FullName:     util.RandomFullName(),
		CreatedAt:    time.Now(),
		PwdUpdatedAt: time.Now(),
	}
	return user, password
}

func requireBodyMatchUser(t *testing.T, body *bytes.Buffer, user db.User) {
	t.Helper()
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var got userResponse
	err = json.Unmarshal(data, &got)
	require.NoError(t, err)

	require.Equal(t, user.Username, got.Username)
	require.Equal(t, user.Email, got.Email)
	require.Equal(t, user.FullName, got.FullName)

	require.NotContains(t, string(data), "hashed_pwd")
	require.NotContains(t, string(data), user.HashedPwd)
}

func TestCreateUserAPI(t *testing.T) {
	user, password := randomUser(t)

	testCases := []struct {
		name          string
		body          map[string]any
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(
			t *testing.T, recorder *httptest.ResponseRecorder,
		)
	}{
		{
			name: "OK",
			body: map[string]any{
				"username":  user.Username,
				"password":  password,
				"email":     user.Email,
				"full_name": user.FullName,
			},
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.CreateUserParams{
					Username: user.Username,
					Email:    user.Email,
					FullName: user.FullName,
				}
				store.EXPECT().
					CreateUser(gomock.Any(), eqCreateUserParams(arg, password)).
					Times(1).
					Return(user, nil)
			},
			checkResponse: func(
				t *testing.T,
				recorder *httptest.ResponseRecorder,
			) {
				require.Equal(t, http.StatusCreated, recorder.Code)
				requireBodyMatchUser(t, recorder.Body, user)
			},
		},
		{
			name: "InternalError",
			body: map[string]any{
				"username":  user.Username,
				"password":  password,
				"email":     user.Email,
				"full_name": user.FullName,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.User{}, sql.ErrConnDone)
			},
			checkResponse: func(
				t *testing.T,
				recorder *httptest.ResponseRecorder,
			) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "DuplicateUsername",
			body: map[string]any{
				"username":  user.Username,
				"password":  password,
				"email":     user.Email,
				"full_name": user.FullName,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Times(1).
					Return(
						db.User{},
						&pq.Error{
							Code:       "23505",
							Constraint: usersPkeyConstraint,
						},
					)
			},
			checkResponse: func(
				t *testing.T,
				recorder *httptest.ResponseRecorder,
			) {
				require.Equal(t, http.StatusConflict, recorder.Code)
			},
		},
		{
			name: "DuplicateEmail",
			body: map[string]any{
				"username":  user.Username,
				"password":  password,
				"email":     user.Email,
				"full_name": user.FullName,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Times(1).
					Return(
						db.User{},
						&pq.Error{
							Code:       "23505",
							Constraint: usersEmailKeyConstraint,
						},
					)
			},
			checkResponse: func(
				t *testing.T,
				recorder *httptest.ResponseRecorder,
			) {
				require.Equal(t, http.StatusConflict, recorder.Code)
			},
		},
		{
			name: "InvalidUsername",
			body: map[string]any{
				"username":  "invalid-username!",
				"password":  password,
				"email":     user.Email,
				"full_name": user.FullName,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(
				t *testing.T,
				recorder *httptest.ResponseRecorder,
			) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "InvalidEmail",
			body: map[string]any{
				"username":  user.Username,
				"password":  password,
				"email":     "not-an-email",
				"full_name": user.FullName,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(
				t *testing.T,
				recorder *httptest.ResponseRecorder,
			) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "PasswordTooShort",
			body: map[string]any{
				"username":  user.Username,
				"password":  "short",
				"email":     user.Email,
				"full_name": user.FullName,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(
				t *testing.T,
				recorder *httptest.ResponseRecorder,
			) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "MissingFullName",
			body: map[string]any{
				"username": user.Username,
				"password": password,
				"email":    user.Email,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(
				t *testing.T,
				recorder *httptest.ResponseRecorder,
			) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
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
				http.MethodPost, "/users", bytes.NewReader(body),
			)
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}
