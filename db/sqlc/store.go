package db

import (
	"context"
	"database/sql"
	"fmt"
)

type Store interface {
	Querier
	TransferTx(ctx context.Context, arg TransferTxParams) (TransferTxResult, error)
}

type SQLStore struct {
	*Queries
	db *sql.DB
}

func NewStore(db *sql.DB) Store {
	return &SQLStore{
		db:      db,
		Queries: New(db),
	}
}

func (store *SQLStore) execTx(ctx context.Context, fn func(*Queries) error) error {
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	q := New(tx)
	err = fn(q)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx err: %v, rb err: %v", err, rbErr)
		}
		return err
	}

	if comErr := tx.Commit(); comErr != nil {
		return comErr
	}
	return nil
}

// type TransferTxParams struct {
// 	FromAccountID int64 `json:"from_account_id"`
// 	ToAccountID   int64 `json:"to_account_id"`
// 	Amount        int64 `json:"amount"`
// }

// type TransferTxResult struct {
// 	Transfer    Transfer `json:"transfer"`
// 	FromAccount Account  `json:"from_account"`
// 	ToAccount   Account  `json:"to_account"`
// 	FromEntry   Entry    `json:"from_entry"`
// 	ToEntry     Entry    `json:"to_entry"`
// }

// // context key for debugging
// type txKeyType string

// var txKey = txKeyType("txName")

// func (store *Store) TransferTx(ctx context.Context, arg TransferTxParams) (TransferTxResult, error) {
// 	var result TransferTxResult

// 	err := store.execTx(ctx, func(q *Queries) error {
// 		var err error

// 		// Extract transaction name for debugging
// 		txName, _ := ctx.Value(txKey).(string)

// 		fmt.Printf("[%s] >> START transaction\n", txName)

// 		// Create transfer record
// 		fmt.Printf("[%s] Creating transfer record\n", txName)
// 		result.Transfer, err = q.CreateTransfer(ctx, CreateTransferParams(arg))
// 		if err != nil {
// 			return err
// 		}

// 		// Create debit entry
// 		fmt.Printf("[%s] Creating debit entry for account %d\n", txName, arg.FromAccountID)
// 		result.FromEntry, err = q.CreateEntry(ctx, CreateEntryParams{
// 			AccountID: arg.FromAccountID,
// 			Amount:    -arg.Amount,
// 		})
// 		if err != nil {
// 			return err
// 		}

// 		// Create credit entry
// 		fmt.Printf("[%s] Creating credit entry for account %d\n", txName, arg.ToAccountID)
// 		result.ToEntry, err = q.CreateEntry(ctx, CreateEntryParams{
// 			AccountID: arg.ToAccountID,
// 			Amount:    arg.Amount,
// 		})
// 		if err != nil {
// 			return err
// 		}

// 		// Lock accounts in consistent order
// 		var fromAccount, toAccount Account
// 		if arg.FromAccountID < arg.ToAccountID {
// 			fmt.Printf("[%s] Locking fromAccount %d\n", txName, arg.FromAccountID)
// 			fromAccount, err = q.GetAccountForUpdate(ctx, arg.FromAccountID)
// 			if err != nil {
// 				return err
// 			}

// 			fmt.Printf("[%s] Locking toAccount %d\n", txName, arg.ToAccountID)
// 			toAccount, err = q.GetAccountForUpdate(ctx, arg.ToAccountID)
// 			if err != nil {
// 				return err
// 			}
// 		} else {
// 			fmt.Printf("[%s] Locking toAccount %d\n", txName, arg.ToAccountID)
// 			toAccount, err = q.GetAccountForUpdate(ctx, arg.ToAccountID)
// 			if err != nil {
// 				return err
// 			}

// 			fmt.Printf("[%s] Locking fromAccount %d\n", txName, arg.FromAccountID)
// 			fromAccount, err = q.GetAccountForUpdate(ctx, arg.FromAccountID)
// 			if err != nil {
// 				return err
// 			}
// 		}

// 		// Update balances
// 		fmt.Printf("[%s] Updating balance of fromAccount %d: %d -> %d\n", txName,
// 			fromAccount.ID, fromAccount.Balance, fromAccount.Balance-arg.Amount)

// 		result.FromAccount, err = q.UpdateAccount(ctx, UpdateAccountParams{
// 			ID:      fromAccount.ID,
// 			Balance: fromAccount.Balance - arg.Amount,
// 		})
// 		if err != nil {
// 			return err
// 		}

// 		fmt.Printf("[%s] Updating balance of toAccount %d: %d -> %d\n", txName,
// 			toAccount.ID, toAccount.Balance, toAccount.Balance+arg.Amount)

// 		result.ToAccount, err = q.UpdateAccount(ctx, UpdateAccountParams{
// 			ID:      toAccount.ID,
// 			Balance: toAccount.Balance + arg.Amount,
// 		})
// 		if err != nil {
// 			return err
// 		}

// 		fmt.Printf("[%s] >> END transaction\n", txName)
// 		return nil
// 	})

// 	return result, err
// }
