package users

import (
	"database/sql"
	"testing"

	"github.com/gcarrenho/component-based/option-1/internal/users/model"
	_ "github.com/mattn/go-sqlite3"

	"github.com/stretchr/testify/assert"
)

func TestGetUserByID(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	assert.NoError(t, err)
	defer db.Close()

	// now only check that no crash

	repo := newUserRepositoryImpl(db)

	user, err := repo.GetUserByID("some-id")
	assert.NoError(t, err)
	assert.Equal(t, model.User{}, user)
}
