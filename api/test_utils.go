// api/test_util.go
package api

import (
	db "github.com/NoahFola/simple_bank/db/sqlc"

	"github.com/gin-gonic/gin"
)

func init() { gin.SetMode(gin.TestMode) }

func newTestServer(store db.Store) *Server {
	return NewServer(store)
}
