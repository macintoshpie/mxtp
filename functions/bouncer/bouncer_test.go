package bouncer

import (
	"fmt"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/require"
)

func handlerA(parameters map[string]string, req events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	return &events.APIGatewayProxyResponse{
		Body: "hello from A",
	}, nil
}

func handlerB(parameters map[string]string, req events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	return &events.APIGatewayProxyResponse{
		Body: "hello from B",
	}, nil
}

func paramPrinter(parameters map[string]string, req events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	return &events.APIGatewayProxyResponse{
		Body: fmt.Sprintf("%+v", parameters),
	}, nil
}

func TestBouncerSimple(t *testing.T) {
	b := New("")
	b.Handle("/hello", handlerA)
	res, err := b.Route(events.APIGatewayProxyRequest{Path: "/hello"})
	require.Nil(t, err)
	require.Equal(t, "hello from A", res.Body)
}

func TestBouncerReturnsErrorWithoutMatch(t *testing.T) {
	b := New("")
	b.Handle("/hello", handlerA)
	_, err := b.Route(events.APIGatewayProxyRequest{Path: "/world"})
	require.NotNil(t, err)
}

func TestBouncerWithBasePath(t *testing.T) {
	b := New("/my/base")
	b.Handle("/hello", handlerA)
	res, err := b.Route(events.APIGatewayProxyRequest{Path: "/my/base/hello"})
	require.Nil(t, err)
	require.Equal(t, "hello from A", res.Body)
}

func TestBouncerMultipleHandlers(t *testing.T) {
	b := New("")
	b.Handle("/handlerA", handlerA)
	b.Handle("/handlerB", handlerB)
	res, err := b.Route(events.APIGatewayProxyRequest{Path: "/handlerA"})
	require.Nil(t, err)
	require.Equal(t, "hello from A", res.Body)
	res, err = b.Route(events.APIGatewayProxyRequest{Path: "/handlerB"})
	require.Nil(t, err)
	require.Equal(t, "hello from B", res.Body)
}

func TestBouncerNestedHandlers(t *testing.T) {
	b := New("")
	b.Handle("/root/handlerA", handlerA)
	res, err := b.Route(events.APIGatewayProxyRequest{Path: "/root/handlerA"})
	require.Nil(t, err)
	require.Equal(t, "hello from A", res.Body)
}

func TestBouncerSimpleParameters(t *testing.T) {
	b := New("")
	b.Handle("/{paramA}", paramPrinter)
	res, err := b.Route(events.APIGatewayProxyRequest{Path: "/hello"})
	require.Nil(t, err)
	require.Equal(t, "map[paramA:hello]", res.Body)
}

func TestBouncerSimpleParametersInPath(t *testing.T) {
	b := New("")
	b.Handle("/authors/{authorId}", paramPrinter)
	res, err := b.Route(events.APIGatewayProxyRequest{Path: "/authors/123"})
	require.Nil(t, err)
	require.Equal(t, "map[authorId:123]", res.Body)
}

func TestBouncerMultipleParametersInPath(t *testing.T) {
	b := New("")
	b.Handle("/authors/{authorId}/books/{bookId}", paramPrinter)
	res, err := b.Route(events.APIGatewayProxyRequest{Path: "/authors/123/books/666"})
	require.Nil(t, err)
	require.Equal(t, "map[authorId:123 bookId:666]", res.Body)
}

func TestBouncerSimpleRestUseCase(t *testing.T) {
	b := New("")
	b.Handle("/authors/{authorId}", paramPrinter)
	b.Handle("/authors/{authorId}/books/{bookId}", paramPrinter)
	b.Handle("/authors/{authorId}/books/{bookId}/pages/{pageNumber}", paramPrinter)

	res, err := b.Route(events.APIGatewayProxyRequest{Path: "/authors/123"})
	require.Nil(t, err)
	require.Equal(t, "map[authorId:123]", res.Body)

	res, err = b.Route(events.APIGatewayProxyRequest{Path: "/authors/123/books/666"})
	require.Nil(t, err)
	require.Equal(t, "map[authorId:123 bookId:666]", res.Body)

	res, err = b.Route(events.APIGatewayProxyRequest{Path: "/authors/123/books/666/pages/41"})
	require.Nil(t, err)
	require.Equal(t, "map[authorId:123 bookId:666 pageNumber:41]", res.Body)

}
