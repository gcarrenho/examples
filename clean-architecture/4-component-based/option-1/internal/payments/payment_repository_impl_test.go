package payments

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
)

func TestGetPaymentByID(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	assert.NoError(t, err)
	defer db.Close()

	// now only check that no crash

	repo := newPaymentRepositoryImpl(db)

	payment, err := repo.GetPaymentByID("some-id")
	assert.NoError(t, err)
	assert.Equal(t, "", payment.ID)
}
