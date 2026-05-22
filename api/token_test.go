package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	mockdb "github.com/jenkka/dummy-bank/db/mock"
	db "github.com/jenkka/dummy-bank/db/sqlc"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestRenewAccessTokenAPI(t *testing.T) {
	user, _ := randomUser(t)

	refreshDuration := time.Hour

	testCases := []struct {
		name          string
		build         func(t *testing.T, server *Server, store *mockdb.MockStore) []byte
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			build: func(t *testing.T, server *Server, store *mockdb.MockStore) []byte {
				refreshToken, payload, err := server.tokenMaker.CreateToken(
					user.Username, refreshDuration,
				)
				require.NoError(t, err)

				sessionID, err := uuid.Parse(payload.ID)
				require.NoError(t, err)

				store.EXPECT().
					GetSession(gomock.Any(), gomock.Eq(sessionID)).
					Times(1).
					Return(db.Session{
						ID:           sessionID,
						Username:     user.Username,
						RefreshToken: refreshToken,
						UserAgent:    "test-agent",
						ClientIp:     "127.0.0.1",
						IsBlocked:    false,
						ExpiresAt:    time.Now().Add(refreshDuration),
						CreatedAt:    time.Now(),
					}, nil)

				body, err := json.Marshal(map[string]any{"refresh_token": refreshToken})
				require.NoError(t, err)
				return body
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)

				var got renewAccessTokenResponse
				err := json.Unmarshal(recorder.Body.Bytes(), &got)
				require.NoError(t, err)
				require.NotEmpty(t, got.AccessToken)
				require.WithinDuration(
					t, time.Now().Add(time.Minute), got.AccessTokenExpiresAt, 2*time.Second,
				)
			},
		},
		{
			name: "MissingRefreshToken",
			build: func(t *testing.T, server *Server, store *mockdb.MockStore) []byte {
				store.EXPECT().GetSession(gomock.Any(), gomock.Any()).Times(0)
				body, _ := json.Marshal(map[string]any{})
				return body
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "InvalidRefreshToken",
			build: func(t *testing.T, server *Server, store *mockdb.MockStore) []byte {
				store.EXPECT().GetSession(gomock.Any(), gomock.Any()).Times(0)
				body, _ := json.Marshal(map[string]any{
					"refresh_token": "not.a.valid.jwt",
				})
				return body
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "ExpiredRefreshToken",
			build: func(t *testing.T, server *Server, store *mockdb.MockStore) []byte {
				refreshToken, _, err := server.tokenMaker.CreateToken(
					user.Username, -time.Minute,
				)
				require.NoError(t, err)

				store.EXPECT().GetSession(gomock.Any(), gomock.Any()).Times(0)
				body, _ := json.Marshal(map[string]any{"refresh_token": refreshToken})
				return body
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "SessionNotFound",
			build: func(t *testing.T, server *Server, store *mockdb.MockStore) []byte {
				refreshToken, _, err := server.tokenMaker.CreateToken(
					user.Username, refreshDuration,
				)
				require.NoError(t, err)

				store.EXPECT().
					GetSession(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.Session{}, sql.ErrNoRows)

				body, _ := json.Marshal(map[string]any{"refresh_token": refreshToken})
				return body
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "GetSessionInternalError",
			build: func(t *testing.T, server *Server, store *mockdb.MockStore) []byte {
				refreshToken, _, err := server.tokenMaker.CreateToken(
					user.Username, refreshDuration,
				)
				require.NoError(t, err)

				store.EXPECT().
					GetSession(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.Session{}, sql.ErrConnDone)

				body, _ := json.Marshal(map[string]any{"refresh_token": refreshToken})
				return body
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "BlockedSession",
			build: func(t *testing.T, server *Server, store *mockdb.MockStore) []byte {
				refreshToken, payload, err := server.tokenMaker.CreateToken(
					user.Username, refreshDuration,
				)
				require.NoError(t, err)
				sessionID, _ := uuid.Parse(payload.ID)

				store.EXPECT().
					GetSession(gomock.Any(), gomock.Eq(sessionID)).
					Times(1).
					Return(db.Session{
						ID:           sessionID,
						Username:     user.Username,
						RefreshToken: refreshToken,
						IsBlocked:    true,
						ExpiresAt:    time.Now().Add(refreshDuration),
					}, nil)

				body, _ := json.Marshal(map[string]any{"refresh_token": refreshToken})
				return body
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
				require.Contains(t, recorder.Body.String(), "blocked")
			},
		},
		{
			name: "WrongSessionUser",
			build: func(t *testing.T, server *Server, store *mockdb.MockStore) []byte {
				refreshToken, payload, err := server.tokenMaker.CreateToken(
					user.Username, refreshDuration,
				)
				require.NoError(t, err)
				sessionID, _ := uuid.Parse(payload.ID)

				store.EXPECT().
					GetSession(gomock.Any(), gomock.Eq(sessionID)).
					Times(1).
					Return(db.Session{
						ID:           sessionID,
						Username:     "someone_else",
						RefreshToken: refreshToken,
						ExpiresAt:    time.Now().Add(refreshDuration),
					}, nil)

				body, _ := json.Marshal(map[string]any{"refresh_token": refreshToken})
				return body
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
				require.Contains(t, recorder.Body.String(), "wrong session user")
			},
		},
		{
			name: "MismatchedRefreshToken",
			build: func(t *testing.T, server *Server, store *mockdb.MockStore) []byte {
				refreshToken, payload, err := server.tokenMaker.CreateToken(
					user.Username, refreshDuration,
				)
				require.NoError(t, err)
				sessionID, _ := uuid.Parse(payload.ID)

				store.EXPECT().
					GetSession(gomock.Any(), gomock.Eq(sessionID)).
					Times(1).
					Return(db.Session{
						ID:           sessionID,
						Username:     user.Username,
						RefreshToken: "a-different-token-string",
						ExpiresAt:    time.Now().Add(refreshDuration),
					}, nil)

				body, _ := json.Marshal(map[string]any{"refresh_token": refreshToken})
				return body
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
				require.Contains(t, recorder.Body.String(), "mismatched")
			},
		},
		{
			name: "ExpiredSession",
			build: func(t *testing.T, server *Server, store *mockdb.MockStore) []byte {
				refreshToken, payload, err := server.tokenMaker.CreateToken(
					user.Username, refreshDuration,
				)
				require.NoError(t, err)
				sessionID, _ := uuid.Parse(payload.ID)

				store.EXPECT().
					GetSession(gomock.Any(), gomock.Eq(sessionID)).
					Times(1).
					Return(db.Session{
						ID:           sessionID,
						Username:     user.Username,
						RefreshToken: refreshToken,
						ExpiresAt:    time.Now().Add(-time.Hour),
					}, nil)

				body, _ := json.Marshal(map[string]any{"refresh_token": refreshToken})
				return body
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
				require.Contains(t, recorder.Body.String(), "expired session")
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			server := newTestServer(t, store)

			body := tc.build(t, server, store)

			recorder := httptest.NewRecorder()
			request, err := http.NewRequest(
				http.MethodPost,
				"/tokens/renew_access",
				bytes.NewReader(body),
			)
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}
