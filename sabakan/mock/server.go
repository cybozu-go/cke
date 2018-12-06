package mock

import (
	"net/http/httptest"

	"github.com/99designs/gqlgen/handler"
)

// Server creates a mock server that implements sabakan GraphQL API.
func Server() *httptest.Server {
	h := handler.GraphQL(NewExecutableSchema(Config{
		Resolvers: mockResolver{},
	}))
	return httptest.NewServer(h)
}
