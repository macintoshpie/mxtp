package main

import (
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/macintoshpie/mxtp-fx/mxtpdb"
)

func LeaguesHandler(request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	err, db := mxtpdb.New()
	if err != nil {
		return &events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       "Failed to create db, " + err.Error(),
		}, nil
	}

	// get league and themes
	err, league, themes := db.GetLeagueAndThemes("devetry")
	if err != nil {
		fmt.Println(err.Error())
		return &events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       "Failed to fetch league, " + err.Error(),
		}, nil
	}

	return &events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       "The request:  " + fmt.Sprintf("%+v\n", request) + fmt.Sprintf("\n\n%v, %v", league, themes),
	}, nil
}

func main() {
	lambda.Start(LeaguesHandler)
}
