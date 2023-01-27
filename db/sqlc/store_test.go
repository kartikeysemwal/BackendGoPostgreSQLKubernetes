package db

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTransferTx(t *testing.T) {
	store := NewStore(testDB)

	fromAccount := createRandomAccount(t)
	toAccount := createRandomAccount(t)

	fmt.Println("Before balance formAccount:", fromAccount.Balance, " toAccount:", toAccount.Balance)

	// running concurrent transactions
	n := 10
	amount := int64(10)

	errs := make(chan error)
	results := make(chan TransferTxResults)

	existed := map[int]bool{}

	for i := 0; i < n; i++ {
		go func() {
			ctx := context.Background()
			result, err := store.TransferTx(ctx, TransferTxParams{
				FromAccountID: fromAccount.ID,
				ToAccountID:   toAccount.ID,
				Amount:        amount,
			})

			errs <- err
			results <- result
		}()
	}

	// check results
	for i := 0; i < n; i++ {
		err := <-errs
		result := <-results

		require.NoError(t, err)
		require.NotEmpty(t, result)

		// check transfer
		transfer := result.Transfer
		require.NotEmpty(t, transfer)
		require.Equal(t, transfer.FromAccountID, fromAccount.ID)
		require.Equal(t, transfer.ToAccountID, toAccount.ID)
		require.Equal(t, transfer.Amount, amount)
		require.NotZero(t, transfer.ID)
		require.NotZero(t, transfer.CreatedAt)

		_, err = store.Queries.GetTransfer(context.Background(), transfer.ID)

		require.NoError(t, err)

		// check entries
		fromEntry := result.FromEntry
		require.NotEmpty(t, fromEntry)
		require.Equal(t, fromEntry.AccountID, fromAccount.ID)
		require.Equal(t, fromEntry.Amount, -amount)
		require.NotZero(t, fromEntry.ID)
		require.NotZero(t, fromEntry.CreatedAt)

		_, err = store.Queries.GetEntry(context.Background(), fromEntry.ID)
		require.NoError(t, err)

		toEntry := result.ToEntry
		require.NotEmpty(t, toEntry)
		require.Equal(t, toEntry.AccountID, toAccount.ID)
		require.Equal(t, toEntry.Amount, amount)
		require.NotZero(t, toEntry.ID)
		require.NotZero(t, toEntry.CreatedAt)

		_, err = store.Queries.GetEntry(context.Background(), toEntry.ID)
		require.NoError(t, err)

		// check accounts
		fromAccountRes := result.FromAccount
		require.NotEmpty(t, fromAccountRes)
		require.Equal(t, fromAccountRes.ID, fromAccount.ID)

		toAccountRes := result.ToAccount
		require.NotEmpty(t, toAccountRes)
		require.Equal(t, toAccountRes.ID, toAccount.ID)

		fmt.Println("Balance in transaction fromAccount:", fromAccountRes.Balance, " toAccount:", toAccountRes.Balance)

		// check accounts balance
		diff1 := fromAccount.Balance - fromAccountRes.Balance
		diff2 := toAccountRes.Balance - toAccount.Balance

		require.Equal(t, diff1, diff2)
		require.True(t, diff1 > 0)
		require.True(t, diff1%amount == 0)

		k := int(diff1 / amount)
		require.True(t, k >= 1 && k <= n)
		require.NotContains(t, existed, k)
		existed[k] = true
	}

	// check the final balance
	updatedFromAccount, err := store.Queries.GetAccount(context.Background(), fromAccount.ID)
	require.NoError(t, err)

	updatedToAccount, err := store.Queries.GetAccount(context.Background(), toAccount.ID)
	require.NoError(t, err)

	fmt.Println("After balance formAccount:", updatedFromAccount.Balance, " toAccount:", updatedToAccount.Balance)

	require.Equal(t, fromAccount.Balance-int64(n)*amount, updatedFromAccount.Balance)
	require.Equal(t, toAccount.Balance+int64(n)*amount, updatedToAccount.Balance)
}

func TestTransferTxDeadlock(t *testing.T) {
	store := NewStore(testDB)

	fromAccount := createRandomAccount(t)
	toAccount := createRandomAccount(t)

	fmt.Println("Before balance formAccount:", fromAccount.Balance, " toAccount:", toAccount.Balance)

	// running concurrent transactions
	n := 10
	amount := int64(10)

	errs := make(chan error)

	for i := 0; i < n; i++ {
		fromAccountId := fromAccount.ID
		toAccountId := toAccount.ID

		if i%2 == 1 {
			fromAccountId = toAccount.ID
			toAccountId = fromAccount.ID
		}

		go func() {
			ctx := context.Background()
			_, err := store.TransferTx(ctx, TransferTxParams{
				FromAccountID: fromAccountId,
				ToAccountID:   toAccountId,
				Amount:        amount,
			})

			errs <- err
		}()
	}

	// check results
	for i := 0; i < n; i++ {
		err := <-errs
		require.NoError(t, err)
	}

	// check the final balance
	updatedFromAccount, err := store.Queries.GetAccount(context.Background(), fromAccount.ID)
	require.NoError(t, err)

	updatedToAccount, err := store.Queries.GetAccount(context.Background(), toAccount.ID)
	require.NoError(t, err)

	fmt.Println("After balance formAccount:", updatedFromAccount.Balance, " toAccount:", updatedToAccount.Balance)

	// as even number of transaction are going from From to To, and then from To to From
	// so the balance at the end should be equal

	require.Equal(t, fromAccount.Balance, updatedFromAccount.Balance)
	require.Equal(t, toAccount.Balance, updatedToAccount.Balance)
}
