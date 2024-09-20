package mock

import (
	"net/http/httptest"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
)

// Server creates a mock server that implements sabakan GraphQL API.
func Server() *httptest.Server {
	h := handler.New(NewExecutableSchema(Config{
		Resolvers: mockResolver{},
	}))
	h.AddTransport(transport.GET{})
	h.AddTransport(transport.POST{})
	return httptest.NewServer(h)
}
