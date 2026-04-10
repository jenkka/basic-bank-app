package db

import (
	"context"
	"database/sql"
	"fmt"
)

type Store struct {
	queries *Queries
	db      *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{
		queries: New(db),
		db:      db,
	}
}

func (store *Store) execTxn(ctx context.Context, fn func(*Queries) error) error {
	txn, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	queries := New(txn)
	err = fn(queries)
	if err != nil {
		rollbackErr := txn.Rollback()
		if rollbackErr != nil {
			return fmt.Errorf("Transaction error: %v, Rollback error: %v", err, rollbackErr)
		}

		return err
	}

	return txn.Commit()
}

type TransferTxnRes struct {
	Transfer    Transfer `json:"transfer"`
	FromAccount Account  `json:"from_account"`
	ToAccount   Account  `json:"to_account"`
	FromEntry   Entry    `json:"from_entry"`
	ToEntry     Entry    `json:"to_entry"`
}

func (store *Store) TransferTxn(ctx context.Context, params CreateTransferParams) (TransferTxnRes, error) {
	var txnRes TransferTxnRes
	var err error

	store.execTxn(ctx, func(queries *Queries) error {
		txnRes.Transfer, err = queries.CreateTransfer(ctx, params)
		if err != nil {
			return err
		}

		fromEntryParams := CreateEntryParams{
			AccountID: params.FromAccountID,
			Amount:    fmt.Sprintf("-%s", params.Amount),
		}
		txnRes.FromEntry, err = queries.CreateEntry(ctx, fromEntryParams)
		if err != nil {
			return err
		}

		toEntryParams := CreateEntryParams{
			AccountID: params.ToAccountID,
			Amount:    params.Amount,
		}
		txnRes.ToEntry, err = queries.CreateEntry(ctx, toEntryParams)
		if err != nil {
			return err
		}

		// TODO: Update accounts' balance
		// txnRes.FromAccount, err = queries.GetAccount(ctx, params.FromAccountID)
		// if err != nil {
		// 	return err
		// }
		//
		// txnRes.ToAccount, err = queries.GetAccount(ctx, params.ToAccountID)
		// if err != nil {
		// 	return err
		// }

		return nil
	})

	return txnRes, err
}
