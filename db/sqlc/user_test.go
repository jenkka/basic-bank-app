package db

import (
	"context"
	"testing"
	"time"

	"github.com/jenkka/basic-bank-app/util"
	"github.com/stretchr/testify/require"
)

func randomUserParams() CreateUserParams {
	return CreateUserParams{
		Username:  util.RandomUsername(),
		HashedPwd: "secret",
		Email:     util.RandomEmail(),
		FullName:  util.RandomFullName(),
	}
}

func createRandomUser(t *testing.T) User {
	userParams := randomUserParams()
	user, err := testQueries.CreateUser(context.Background(), userParams)
	require.NoError(t, err)
	require.NotEmpty(t, user)

	return user
}

func TestCreateUser(t *testing.T) {
	userParams := randomUserParams()
	user, err := testQueries.CreateUser(context.Background(), userParams)

	require.NoError(t, err)
	require.NotEmpty(t, user)
	require.NotZero(t, user.CreatedAt)
	require.NotZero(t, user.PwdUpdatedAt)
	require.Equal(t, userParams.Username, user.Username)
	require.Equal(t, userParams.HashedPwd, user.HashedPwd)
	require.Equal(t, userParams.Email, user.Email)
	require.Equal(t, userParams.FullName, user.FullName)
}

func TestGetUser(t *testing.T) {
	createdUser := createRandomUser(t)

	getUser, err := testQueries.GetUser(context.Background(), createdUser.Username)
	require.NoError(t, err)
	require.NotEmpty(t, getUser)
	require.Equal(t, createdUser.Username, getUser.Username)
	require.Equal(t, createdUser.HashedPwd, getUser.HashedPwd)
	require.Equal(t, createdUser.Email, getUser.Email)
	require.Equal(t, createdUser.FullName, getUser.FullName)
	require.WithinDuration(t, createdUser.CreatedAt, getUser.CreatedAt, time.Second)
	require.WithinDuration(t, createdUser.PwdUpdatedAt, getUser.PwdUpdatedAt, time.Second)
}
