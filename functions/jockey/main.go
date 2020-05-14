package main

import (
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/macintoshpie/mxtp-fx/bouncer"
	"github.com/macintoshpie/mxtp-fx/mxtpdb"
)

func leaguesHandler(parameters map[string]string, request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
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
		Body:       fmt.Sprintf("%v\n%v", league, themes),
	}, nil
}

func JockeyHandler(request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	b := bouncer.New("/.netlify/functions")
	b.Handle("/leagues/{leagueName}", leaguesHandler)

	return b.Route(request)
}

func main() {
	lambda.Start(JockeyHandler)
}
