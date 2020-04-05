package fauna_db_example

import (
	"os"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/require"
)

func TestFaunaDbExampleHandler(t *testing.T) {
	tests := []struct {
		request events.APIGatewayProxyRequest
		expect  string
		err     error
	}{
		{
			// Test that the handler responds with the correct response
			// when a valid name is provided in the HTTP body
			request: events.APIGatewayProxyRequest{Body: ""},
			expect:  "Finished",
			err:     nil,
		},
	}

	os.Setenv("MXTP_TESTING", "true")
	for _, test := range tests {
		response, err := FaunaDbExampleHandler(test.request)
		require.IsType(t, test.err, err)
		require.Equal(t, test.expect, response.Body)
	}
}
