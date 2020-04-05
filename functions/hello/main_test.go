package hello

import (
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/require"
)

func TestHelloHandler(t *testing.T) {
	tests := []struct {
		request events.APIGatewayProxyRequest
		expect  string
		err     error
	}{
		{
			// Test that the handler responds with the correct response
			// when a valid name is provided in the HTTP body
			request: events.APIGatewayProxyRequest{Body: "Paul"},
			expect:  "Hello, Paul",
			err:     nil,
		},
	}

	for _, test := range tests {
		response, err := HelloHandler(test.request)
		require.IsType(t, test.err, err)
		require.Equal(t, test.expect, response.Body)
	}
}
