package hello

import (
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func HelloHandler(request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {

	return &events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       "Hello, " + request.Body,
	}, nil
}

func main() {
	// Make the handler available for Remote Procedure Call by AWS Lambda
	lambda.Start(HelloHandler)
}
