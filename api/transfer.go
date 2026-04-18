package api

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	db "github.com/jenkka/basic-bank-app/db/sqlc"
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

	decimalAmount, err := decimal.NewFromString(req.Amount)
	if err != nil || decimalAmount.LessThanOrEqual(decimal.Zero) {
		ctx.JSON(
			http.StatusBadRequest,
			errorResponse(errors.New("amount must be a positive number")),
		)
		return
	}

	if !server.validAccountCurrency(ctx, req.FromAccountID, req.Currency) {
		return
	}

	if !server.validAccountCurrency(ctx, req.ToAccountID, req.Currency) {
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

	ctx.JSON(http.StatusOK, transfer)
}

func (server *Server) validAccountCurrency(ctx *gin.Context, accountId int64, currency string) bool {
	account, err := server.store.GetAccount(ctx, accountId)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			err = fmt.Errorf("could not find an account with the ID %d.", accountId)
			ctx.JSON(http.StatusNotFound, errorResponse(err))
		default:
			ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		}
		return false
	}

	if account.Currency != currency {
		err = fmt.Errorf("currency mismatch for account %d: %s, vs %s", accountId, account.Currency, currency)
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return false
	}

	return true
}
