package db

import (
	"context"
	"database/sql"
	"fmt"
)

type Store struct {
	*Queries
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{
		db:      db,
		Queries: New(db),
	}
}

func (store *Store) execTx(ctx context.Context, fn func(*Queries) error) error {
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	q := New(tx)
	err = fn(q)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			fmt.Println("error from transaction/rollback")
			return fmt.Errorf("tx err: %v, rb err: %v", err, rbErr)
		}

		return err
	}

	comerr := tx.Commit()
	if comerr != nil {
		fmt.Println("commit err: ", comerr)
		return comerr
	}
	fmt.Println("successful commit")
	return nil

}

type TransferTxParams struct {
	FromAccountID int64 `json:"from_account_id"`
	ToAccountID   int64 `json:"to_account_id"`
	Amount        int64 `json:"amount"`
}

type TransferTxResult struct {
	Transfer    Transfer `json:"transfer"`
	FromAccount Account  `json:"from_account"`
	ToAccount   Account  `json:"to_account"`
	FromEntry   Entry    `json:"from_entry"`
	ToEntry     Entry    `json:"to_entry"`
}

func (store *Store) TransferTx(ctx context.Context, arg TransferTxParams) (TransferTxResult, error) {
	var result TransferTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		var err error

		result.Transfer, err = q.CreateTransfer(ctx, CreateTransferParams(arg))

		if err != nil {
			return err
		}

		result.FromEntry, err = q.CreateEntry(ctx, CreateEntryParams{
			AccountID: arg.FromAccountID,
			Amount:    -arg.Amount,
		})

		if err != nil {
			return err
		}

		result.ToEntry, err = q.CreateEntry(ctx, CreateEntryParams{
			AccountID: arg.ToAccountID,
			Amount:    arg.Amount,
		})

		if err != nil {
			return err
		}
		if arg.FromAccountID < arg.ToAccountID {
			fromAccount, err := q.GetAccountForUpdate(ctx, arg.FromAccountID)
			if err != nil {
				return err
			}
			toAccount, err := q.GetAccountForUpdate(ctx, arg.ToAccountID)
			if err != nil {
				return err
			}

			UpdateFromAccountParams := UpdateAccountParams{
				ID:      fromAccount.ID,
				Balance: fromAccount.Balance - arg.Amount,
			}
			UpdateToAccountParams := UpdateAccountParams{
				ID:      toAccount.ID,
				Balance: toAccount.Balance + arg.Amount,
			}

			result.FromAccount, err = q.UpdateAccount(ctx, UpdateFromAccountParams)
			if err != nil {
				return err
			}

			result.ToAccount, err = q.UpdateAccount(ctx, UpdateToAccountParams)
			if err != nil {
				return err
			}
		} else {
			toAccount, err := q.GetAccountForUpdate(ctx, arg.ToAccountID)
			if err != nil {
				return err
			}
			fromAccount, err := q.GetAccountForUpdate(ctx, arg.FromAccountID)
			if err != nil {
				return err
			}

			UpdateFromAccountParams := UpdateAccountParams{
				ID:      fromAccount.ID,
				Balance: fromAccount.Balance - arg.Amount,
			}
			UpdateToAccountParams := UpdateAccountParams{
				ID:      toAccount.ID,
				Balance: toAccount.Balance + arg.Amount,
			}

			result.FromAccount, err = q.UpdateAccount(ctx, UpdateFromAccountParams)
			if err != nil {
				return err
			}

			result.ToAccount, err = q.UpdateAccount(ctx, UpdateToAccountParams)
			if err != nil {
				return err
			}
		}
		return nil
	})

	return result, err
}
