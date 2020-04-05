package fauna_db_example

import (
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	f "github.com/fauna/faunadb-go/faunadb"
)

func FaunaDbExampleHandler(request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	if os.Getenv("MXTP_TESTING") == "true" {
		return &events.APIGatewayProxyResponse{
			StatusCode: 200,
			Body:       "Finished",
		}, nil
	}

	secret := os.Getenv("FAUNA_DB_SECRET")
	if secret == "" {
		panic("FAUNA_DB_SECRET environment variable missing")
	}

	endpoint := f.Endpoint("https://db.fauna.com")

	adminClient := f.NewFaunaClient(secret, endpoint)
	dbName := "learn-fauna-go"

	_, err := adminClient.Query(
		f.If(
			f.Exists(f.Database(dbName)),
			true,
			f.CreateDatabase(f.Obj{"name": dbName})))

	if err != nil {
		panic(err)
	}

	return &events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       "Finished",
	}, nil
}

func main() {
	// Make the handler available for Remote Procedure Call by AWS Lambda
	lambda.Start(FaunaDbExampleHandler)
}
