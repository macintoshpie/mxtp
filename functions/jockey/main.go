package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/macintoshpie/mxtp-fx/bouncer"
	"github.com/macintoshpie/mxtp-fx/mxtpdb"
)

type jsonResponse struct {
	content interface{}
	status  int
}

type MessageResponse struct {
	Message string
}

type LeagueAndThemesResponse struct {
	League      mxtpdb.League
	VoteTheme   mxtpdb.Theme
	SubmitTheme mxtpdb.Theme
	Themes      []mxtpdb.Theme
}

type ThemeAndSubmissionsResponse struct {
	Theme       mxtpdb.Theme
	Submissions []mxtpdb.Submission
}

func newMessageResponse(status int, message string) *jsonResponse {
	return &jsonResponse{
		content: MessageResponse{
			Message: message,
		},
		status: status,
	}
}

func (response *jsonResponse) toAPIGatewayProxyResponse() *events.APIGatewayProxyResponse {
	defaultHeaders := make(map[string]string)
	defaultHeaders["Access-Control-Allow-Origin"] = "*"
	defaultHeaders["Access-Control-Allow-Headers"] = "*"
	defaultHeaders["Access-Control-Allow-Methods"] = "POST, GET, OPTIONS, DELETE"
	defaultHeaders["Access-Control-Max-Age"] = "86400"
	bytes, err := json.MarshalIndent(response.content, "", "    ")
	if err != nil {
		fmt.Println("ERROR: failed to marshal content: ", err.Error())
		return &events.APIGatewayProxyResponse{
			Body:       "Internal server error",
			StatusCode: 500,
			Headers:    defaultHeaders,
		}
	}

	return &events.APIGatewayProxyResponse{
		Body:       string(bytes),
		StatusCode: response.status,
		Headers:    defaultHeaders,
	}
}

func authMiddleware(handler bouncer.ApiHandler) bouncer.ApiHandler {
	return func(parameters map[string]string, request events.APIGatewayProxyRequest) *events.APIGatewayProxyResponse {
		authHeader := strings.TrimSpace(request.Headers["authorization"])
		if authHeader == "" {
			authHeader = strings.TrimSpace(request.Headers["Authorization"])
		}
		authParts := strings.Fields(authHeader)
		fmt.Println("Auth header: ", authHeader)
		fmt.Println("Auth parts: ", authParts)
		if len(authParts) != 2 || strings.ToLower(authParts[0]) != "basic" {
			fmt.Println("WARNING: auth header was not what was expected")
			return handler(parameters, request)
		}
		authHeaderDecodedBytes, err := base64.StdEncoding.DecodeString(authParts[1])
		if err != nil {
			fmt.Println("ERROR: failed to decode authorization header: ", err.Error())
			return handler(parameters, request)
		}

		authHeader = string(authHeaderDecodedBytes)
		if authHeader == "" {
			fmt.Println("Somehow it was empty")
			return handler(parameters, request)
		}

		parameters["username"] = authHeader
		return handler(parameters, request)
	}
}

func getLeaguesHandler(parameters map[string]string, request events.APIGatewayProxyRequest) *events.APIGatewayProxyResponse {
	db, err := mxtpdb.New()
	if err != nil {
		fmt.Println("ERROR: ", err.Error())
		return newMessageResponse(500, "Internal Server Error").toAPIGatewayProxyResponse()
	}

	leagueName := parameters["leagueName"]
	if leagueName == "" {
		fmt.Println("ERROR: Parameter 'leagueName' not found")
		return newMessageResponse(500, "Internal Server Error").toAPIGatewayProxyResponse()
	}

	league, themes, err := db.GetLeagueAndThemes(leagueName)
	if err != nil {
		fmt.Println("ERROR: ", err.Error())
		return newMessageResponse(500, "Internal Server Error").toAPIGatewayProxyResponse()
	}

	// find the vote and submit theme in the themes
	var voteTheme, submitTheme mxtpdb.Theme
	for _, theme := range themes {
		if theme.Id == league.VoteThemeId {
			voteTheme = theme
		}
		if theme.Id == league.SubmitThemeId {
			submitTheme = theme
		}
	}
	response := jsonResponse{
		content: LeagueAndThemesResponse{
			League:      *league,
			Themes:      themes,
			VoteTheme:   voteTheme,
			SubmitTheme: submitTheme,
		},
		status: 200,
	}
	return response.toAPIGatewayProxyResponse()
}

func getThemesHandler(parameters map[string]string, request events.APIGatewayProxyRequest) *events.APIGatewayProxyResponse {
	db, err := mxtpdb.New()
	if err != nil {
		fmt.Println("ERROR: ", err.Error())
		return newMessageResponse(500, "Internal Server Error").toAPIGatewayProxyResponse()
	}

	leagueName := parameters["leagueName"]
	if leagueName == "" {
		fmt.Println("ERROR: Parameter 'leagueName' not found")
		return newMessageResponse(500, "Internal Server Error").toAPIGatewayProxyResponse()
	}

	themeId := parameters["themeId"]
	if themeId == "" {
		fmt.Println("ERROR: Parameter 'themeId' not found")
		return newMessageResponse(500, "Internal Server Error").toAPIGatewayProxyResponse()
	}

	theme, submissions, err := db.GetThemeAndSubmissions(leagueName, themeId)
	if err != nil {
		fmt.Println("ERROR: ", err.Error())
		return newMessageResponse(500, "Internal Server Error").toAPIGatewayProxyResponse()
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

func postSubmissionsHandler(parameters map[string]string, request events.APIGatewayProxyRequest) *events.APIGatewayProxyResponse {
	username := parameters["username"]
	if username == "" {
		return newMessageResponse(400, "Invalid Authorization header").toAPIGatewayProxyResponse()
	}

	leagueName := parameters["leagueName"]
	if leagueName == "" {
		fmt.Println("ERROR: Parameter 'leagueName' not found")
		return newMessageResponse(500, "Internal Server Error").toAPIGatewayProxyResponse()
	}

	themeId := parameters["themeId"]
	if themeId == "" {
		fmt.Println("ERROR: Parameter 'themeId' not found")
		return newMessageResponse(500, "Internal Server Error").toAPIGatewayProxyResponse()
	}

	db, err := mxtpdb.New()
	if err != nil {
		fmt.Println("ERROR: ", err.Error())
		return newMessageResponse(500, "Internal Server Error").toAPIGatewayProxyResponse()
	}

	var submission mxtpdb.Submission
	err = json.Unmarshal([]byte(request.Body), &submission)
	if err != nil {
		fmt.Println("ERROR: failed to unmarshal submission: ", err.Error())
		return newMessageResponse(400, "Bad submission").toAPIGatewayProxyResponse()
	}

	err = db.UpdateSubmission(leagueName, themeId, username, submission.SongUrl, nil)
	if err != nil {
		fmt.Println("ERROR: failed to put submission: ", err.Error())
		return newMessageResponse(500, "Internal Server Error").toAPIGatewayProxyResponse()
	}

	return newMessageResponse(200, "Successfully put submission").toAPIGatewayProxyResponse()
}

func postVotesHandler(parameters map[string]string, request events.APIGatewayProxyRequest) *events.APIGatewayProxyResponse {
	username := parameters["username"]
	if username == "" {
		return newMessageResponse(400, "Invalid Authorization header").toAPIGatewayProxyResponse()
	}

	leagueName := parameters["leagueName"]
	if leagueName == "" {
		fmt.Println("ERROR: Parameter 'leagueName' not found")
		return newMessageResponse(500, "Internal Server Error").toAPIGatewayProxyResponse()
	}

	themeId := parameters["themeId"]
	if themeId == "" {
		fmt.Println("ERROR: Parameter 'themeId' not found")
		return newMessageResponse(500, "Internal Server Error").toAPIGatewayProxyResponse()
	}

	db, err := mxtpdb.New()
	if err != nil {
		fmt.Println("ERROR: ", err.Error())
		return newMessageResponse(500, "Internal Server Error").toAPIGatewayProxyResponse()
	}

	var submission mxtpdb.Submission
	err = json.Unmarshal([]byte(request.Body), &submission)
	if err != nil {
		fmt.Println("ERROR: failed to unmarshal submission: ", err.Error())
		return newMessageResponse(400, "Bad submission").toAPIGatewayProxyResponse()
	}

	err = db.UpdateSubmission(leagueName, themeId, username, "", &submission.Votes)
	if err != nil {
		fmt.Println("ERROR: failed to put submission: ", err.Error())
		return newMessageResponse(500, "Internal Server Error").toAPIGatewayProxyResponse()
	}

	return newMessageResponse(200, "Successfully updated submission").toAPIGatewayProxyResponse()
}

func JockeyHandler(request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	// handle OPTIONS preflight request
	if request.HTTPMethod == bouncer.Options {
		response := newMessageResponse(204, "").toAPIGatewayProxyResponse()
		response.Headers["Access-Control-Allow-Origin"] = "*"
		response.Headers["Access-Control-Allow-Headers"] = "*"
		response.Headers["Access-Control-Allow-Methods"] = "POST, GET, OPTIONS, DELETE"
		response.Headers["Access-Control-Max-Age"] = "86400"
		return response, nil
	}

	b := bouncer.New("/.netlify/functions/jockey")

	b.Handle(bouncer.Get, "/leagues/{leagueName}", authMiddleware(getLeaguesHandler))
	b.Handle(bouncer.Get, "/leagues/{leagueName}/themes/{themeId}", authMiddleware(getThemesHandler))
	b.Handle(bouncer.Post, "/leagues/{leagueName}/themes/{themeId}/submissions", authMiddleware(postSubmissionsHandler))
	b.Handle(bouncer.Post, "/leagues/{leagueName}/themes/{themeId}/votes", authMiddleware(postVotesHandler))

	return b.Route(request)
}

func main() {
	lambda.Start(JockeyHandler)
}
