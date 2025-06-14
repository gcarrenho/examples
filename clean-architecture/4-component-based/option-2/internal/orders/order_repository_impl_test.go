package orders

import (
	"database/sql"
	"testing"

	"github.com/gcarrenho/component-based/option-2/internal/orders/model"
	_ "github.com/mattn/go-sqlite3"

	"github.com/stretchr/testify/assert"
)

func TestGetOrderByID(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	assert.NoError(t, err)
	defer db.Close()

	// now only check that no crash

	repo := newOrderRepositoryImpl(db)

	order, err := repo.GetOrderByID("some-id")
	assert.NoError(t, err)
	assert.Equal(t, model.Order{ID: "1234"}, order)
}
