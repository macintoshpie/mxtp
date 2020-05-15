package main

import (
	"encoding/json"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/macintoshpie/mxtp-fx/bouncer"
	"github.com/macintoshpie/mxtp-fx/mxtpdb"
)

type jsonResponse struct {
	content interface{}
	status  int
}

type ErrorResponse struct {
	Message string
}

type LeagueAndThemesResponse struct {
	League mxtpdb.League
	Themes []mxtpdb.Theme
}

type ThemeAndSubmissionsResponse struct {
	Theme       mxtpdb.Theme
	Submissions []mxtpdb.Submission
}

func newErrorResponse(status int, message string) *jsonResponse {
	return &jsonResponse{
		content: ErrorResponse{
			Message: message,
		},
		status: status,
	}
}

func (response *jsonResponse) toAPIGatewayProxyResponse() *events.APIGatewayProxyResponse {
	bytes, err := json.MarshalIndent(response.content, "", "    ")
	if err != nil {
		fmt.Println("ERROR: failed to marshal content: ", err.Error())
		return &events.APIGatewayProxyResponse{
			Body:       "Internal server error",
			StatusCode: 500,
		}
	}
	return &events.APIGatewayProxyResponse{
		Body:       string(bytes),
		StatusCode: response.status,
	}
}

func getLeaguesHandler(parameters map[string]string, request events.APIGatewayProxyRequest) *events.APIGatewayProxyResponse {
	db, err := mxtpdb.New()
	if err != nil {
		fmt.Println("ERROR: ", err.Error())
		return newErrorResponse(500, "Internal Server Error").toAPIGatewayProxyResponse()
	}

	leagueName := parameters["leagueName"]
	if leagueName == "" {
		fmt.Println("ERROR: Parameter 'leagueName' not found")
		return newErrorResponse(500, "Internal Server Error").toAPIGatewayProxyResponse()
	}

	league, themes, err := db.GetLeagueAndThemes(leagueName)
	if err != nil {
		fmt.Println("ERROR: ", err.Error())
		return newErrorResponse(500, "Internal Server Error").toAPIGatewayProxyResponse()
	}

	response := jsonResponse{
		content: LeagueAndThemesResponse{
			League: *league,
			Themes: themes,
		},
		status: 200,
	}
	return response.toAPIGatewayProxyResponse()
}

func getThemesHandler(parameters map[string]string, request events.APIGatewayProxyRequest) *events.APIGatewayProxyResponse {
	db, err := mxtpdb.New()
	if err != nil {
		fmt.Println("ERROR: ", err.Error())
		return newErrorResponse(500, "Internal Server Error").toAPIGatewayProxyResponse()
	}

	leagueName := parameters["leagueName"]
	if leagueName == "" {
		fmt.Println("ERROR: Parameter 'leagueName' not found")
		return newErrorResponse(500, "Internal Server Error").toAPIGatewayProxyResponse()
	}

	themeId := parameters["themeId"]
	if themeId == "" {
		fmt.Println("ERROR: Parameter 'themeId' not found")
		return newErrorResponse(500, "Internal Server Error").toAPIGatewayProxyResponse()
	}

	theme, submissions, err := db.GetThemeAndSubmissions(leagueName, themeId)
	if err != nil {
		fmt.Println("ERROR: ", err.Error())
		return newErrorResponse(500, "Internal Server Error").toAPIGatewayProxyResponse()
	}

	response := jsonResponse{
		content: ThemeAndSubmissionsResponse{
			Theme:       *theme,
			Submissions: submissions,
		},
		status: 200,
	}
	return response.toAPIGatewayProxyResponse()
}

func JockeyHandler(request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	b := bouncer.New("/.netlify/functions/jockey")

	b.Handle(bouncer.Get, "/leagues/{leagueName}", getLeaguesHandler)
	b.Handle(bouncer.Get, "/leagues/{leagueName}/themes/{themeId}", getThemesHandler)

	return b.Route(request)
}

func main() {
	lambda.Start(JockeyHandler)
}
