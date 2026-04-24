package api

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	db "github.com/jenkka/basic-bank-app/db/sqlc"
	"github.com/jenkka/basic-bank-app/token"
	"github.com/shopspring/decimal"
)

type createTransferRequest struct {
	FromAccountID int64  `json:"from_account_id" binding:"required,gt=0"`
	ToAccountID   int64  `json:"to_account_id" binding:"required,gt=0"`
	Amount        string `json:"amount" binding:"required"`
	Currency      string `json:"currency" binding:"required,validcurrency"`
}

func (server *Server) createTransfer(ctx *gin.Context) {
	var req createTransferRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	if req.FromAccountID == req.ToAccountID {
		err := errors.New(
			"origin and destination account IDs cannot be the same",
		)
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	decimalAmount, err := decimal.NewFromString(req.Amount)
	if err != nil || decimalAmount.LessThanOrEqual(decimal.Zero) {
		ctx.JSON(
			http.StatusBadRequest,
			errorResponse(errors.New("amount must be a positive number")),
		)
		return
	}

	fromAccount := server.verifyAccountExists(ctx, req.FromAccountID)
	if fromAccount == nil {
		return
	}

	if fromAccount.Owner != authPayload.Username {
		err := errors.New(
			"you are not authorized to send money from this account",
		)
		ctx.JSON(http.StatusForbidden, errorResponse(err))
		return
	}

	toAccount := server.verifyAccountExists(ctx, req.ToAccountID)
	if toAccount == nil {
		return
	}

	if !verifyAccountCurrency(ctx, fromAccount, req.Currency) {
		return
	}

	if !verifyAccountCurrency(ctx, toAccount, req.Currency) {
		return
	}

	transferParams := db.CreateTransferParams{
		FromAccountID: req.FromAccountID,
		ToAccountID:   req.ToAccountID,
		Amount:        decimalAmount,
	}

	transfer, err := server.store.TransferTxn(ctx, transferParams)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusCreated, transfer)
}

func (server *Server) verifyAccountExists(
	ctx *gin.Context, accountId int64,
) *db.Account {
	account, err := server.store.GetAccount(ctx, accountId)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			err = fmt.Errorf("could not find an account with the ID %d.", accountId)
			ctx.JSON(http.StatusNotFound, errorResponse(err))
		default:
			ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		}
		return nil
	}

	return &account
}

func verifyAccountCurrency(
	ctx *gin.Context, account *db.Account, currency string,
) bool {
	if account.Currency != currency {
		err := fmt.Errorf(
			"currency mismatch for account %d: %s, vs %s",
			account.ID, account.Currency, currency,
		)
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return false
	}

	return true
}
