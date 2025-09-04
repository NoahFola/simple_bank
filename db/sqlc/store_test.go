package db

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTransferTx(t *testing.T) {
	account1 := createRandomAccount(t)
	account2 := createRandomAccount(t)
	fmt.Println(">> before:", account1.Balance, account2.Balance)

	testStore := NewStore(testDB)

	n := 24
	amount := int64(10)

	errs := make(chan error)
	// results := make(chan TransferTxResult)

	// run n concurrent transfer transactions
	for i := 0; i < n; i++ {
		txName := fmt.Sprintf("tx %d", i+1)

		fromAccountID := account1.ID
		toAccountID := account2.ID

		if i%2 == 0 {
			fromAccountID = account2.ID
			toAccountID = account1.ID
		}
		go func(name string) {
			ctx := context.WithValue(context.Background(), txKey, name)
			_, err := testStore.TransferTx(ctx, TransferTxParams{
				FromAccountID: fromAccountID,
				ToAccountID:   toAccountID,
				Amount:        amount,
			})

			errs <- err
		}(txName)
	}

	// check results
	// existed := make(map[int]bool)

	for i := 0; i < n; i++ {
		err := <-errs
		require.NoError(t, err)

		// result := <-results
		// require.NotEmpty(t, result)

		// // check transfer
		// transfer := result.Transfer
		// require.NotEmpty(t, transfer)
		// require.Equal(t, account1.ID, transfer.FromAccountID)
		// require.Equal(t, account2.ID, transfer.ToAccountID)
		// require.Equal(t, amount, transfer.Amount)

		// // check entries
		// fromEntry := result.FromEntry
		// require.Equal(t, account1.ID, fromEntry.AccountID)
		// require.Equal(t, -amount, fromEntry.Amount)

		// toEntry := result.ToEntry
		// require.Equal(t, account2.ID, toEntry.AccountID)
		// require.Equal(t, amount, toEntry.Amount)

		// // check accounts
		// fromAccount := result.FromAccount
		// toAccount := result.ToAccount
		// require.Equal(t, account1.ID, fromAccount.ID)
		// require.Equal(t, account2.ID, toAccount.ID)

		// // check balances
		// diff1 := account1.Balance - fromAccount.Balance
		// diff2 := toAccount.Balance - account2.Balance
		// require.Equal(t, diff1, diff2)
		// require.True(t, diff1 > 0)
		// require.True(t, diff1%amount == 0)

		// k := int(diff1 / amount)
		// require.True(t, k >= 1 && k <= n)
		// require.NotContains(t, existed, k)
		// existed[k] = true
	}

	// check final balances
	updatedAccount1, err := testStore.GetAccount(context.Background(), account1.ID)
	require.NoError(t, err)

	updatedAccount2, err := testStore.GetAccount(context.Background(), account2.ID)
	require.NoError(t, err)

	fmt.Println(">> after:", updatedAccount1.Balance, updatedAccount2.Balance)

	require.Equal(t, account1.Balance, updatedAccount1.Balance)
	require.Equal(t, account2.Balance, updatedAccount2.Balance)
}
