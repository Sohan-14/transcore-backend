package auth

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Module is the public auth module entry point.
type Module struct {
}

// New build the module
func New(pool *pgxpool.Pool) *Module {
	return &Module{}
}

// RegisterRoutes register all the routes/endpoint for the auth module
func (m *Module) RegisterRoutes(mux *http.ServeMux) {

}
