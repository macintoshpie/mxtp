package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/google/uuid"
	"github.com/macintoshpie/mxtp-fx/bouncer"
	"github.com/macintoshpie/mxtp-fx/mxtpdb"
)

const CALLBACK_URI = "https://wwww.mxtp.xyz/.netlify/functions/jockey/callback"

type Game struct {
	League           mxtpdb.League
	SubmitThemeItems mxtpdb.ThemeItems
	VoteThemeItems   mxtpdb.ThemeItems
}

type jsonResponse struct {
	content interface{}
	status  int
}

type MessageResponse struct {
	Message string
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
	// TODO: restrict origins
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
		if len(authParts) != 2 || strings.ToLower(authParts[0]) != "bearer" {
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
			return handler(parameters, request)
		}

		parameters["username"] = authHeader
		// TODO: remove this hack once proper auth tokens are implemented
		if authHeader != "" && authHeader == os.Getenv("JOCKEY_SECRET") {
			parameters["ADMIN"] = "indeed"
		}
		return handler(parameters, request)
	}
}

func postSongsHandler(parameters map[string]string, request events.APIGatewayProxyRequest) *events.APIGatewayProxyResponse {
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

	var song mxtpdb.Song
	err = json.Unmarshal([]byte(request.Body), &song)
	if err != nil {
		fmt.Println("ERROR: failed to unmarshal song: ", err.Error())
		return newMessageResponse(400, "Bad song").toAPIGatewayProxyResponse()
	}

	err = db.UpdateSong(leagueName, themeId, username, song.SongUrl, uuid.New().String())
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

	var votes mxtpdb.Votes
	err = json.Unmarshal([]byte(request.Body), &votes)
	if err != nil {
		fmt.Println("ERROR: failed to unmarshal votes: ", err.Error())
		return newMessageResponse(400, "Bad votes").toAPIGatewayProxyResponse()
	}

	err = db.UpdateVotes(leagueName, themeId, username, votes.SubmissionIds)
	if err != nil {
		fmt.Println("ERROR: failed to put votes: ", err.Error())
		return newMessageResponse(500, "Internal Server Error").toAPIGatewayProxyResponse()
	}

	return newMessageResponse(200, "Successfully updated votes").toAPIGatewayProxyResponse()
}

func getGamesHandler(parameters map[string]string, request events.APIGatewayProxyRequest) *events.APIGatewayProxyResponse {
	username := parameters["username"]
	if username == "" {
		return newMessageResponse(400, "Invalid Authorization header").toAPIGatewayProxyResponse()
	}

	leagueName := parameters["leagueName"]
	if leagueName == "" {
		fmt.Println("ERROR: Parameter 'leagueName' not found")
		return newMessageResponse(500, "Internal Server Error").toAPIGatewayProxyResponse()
	}

	// This is ignored for now, just getting current game
	gameId := parameters["gameId"]
	if gameId == "" {
		fmt.Println("ERROR: Parameter 'gameId' not found")
		return newMessageResponse(500, "Internal Server Error").toAPIGatewayProxyResponse()
	}

	db, err := mxtpdb.New()
	if err != nil {
		fmt.Println("ERROR: ", err.Error())
		return newMessageResponse(500, "Internal Server Error").toAPIGatewayProxyResponse()
	}

	league, err := db.GetLeague(leagueName)
	if err != nil {
		fmt.Println("ERROR: ", err.Error())
		return newMessageResponse(500, "Internal Server Error").toAPIGatewayProxyResponse()
	}

	// get the submit theme data
	submitThemeItems, err := db.GetThemeItems(leagueName, league.SubmitTheme.Date)
	if err != nil {
		fmt.Println("ERROR: ", err.Error())
		return newMessageResponse(500, "Internal Server Error").toAPIGatewayProxyResponse()
	}
	// Get the vote theme data
	voteThemeItems, err := db.GetThemeItems(leagueName, league.VoteTheme.Date)
	if err != nil {
		fmt.Println("ERROR: ", err.Error())
		return newMessageResponse(500, "Internal Server Error").toAPIGatewayProxyResponse()
	}

	// return everything if this is an admin request
	if _, ok := parameters["ADMIN"]; ok {
		response := jsonResponse{
			content: Game{
				League:           league,
				SubmitThemeItems: submitThemeItems,
				VoteThemeItems:   voteThemeItems,
			},
			status: 200,
		}
		return response.toAPIGatewayProxyResponse()
	}

	// make sure votes aren't included
	submitThemeItems.Votes = nil
	// only include this user's song
	userSong := mxtpdb.Song{}
	for _, song := range submitThemeItems.Songs {
		if song.UserId == username {
			userSong = song
			break
		}
	}
	submitThemeItems.Songs = []mxtpdb.Song{userSong}

	// anonymize the songs
	for idx := range voteThemeItems.Songs {
		voteThemeItems.Songs[idx].UserId = ""
	}
	// find this user's votes
	userVotes := mxtpdb.Votes{}
	for _, vote := range voteThemeItems.Votes {
		if vote.UserId == username {
			userVotes = vote
			break
		}
	}
	voteThemeItems.Votes = []mxtpdb.Votes{userVotes}

	response := jsonResponse{
		content: Game{
			League:           league,
			SubmitThemeItems: submitThemeItems,
			VoteThemeItems:   voteThemeItems,
		},
		status: 200,
	}

	return response.toAPIGatewayProxyResponse()
}

func callbackHandler(parameters map[string]string, request events.APIGatewayProxyRequest) *events.APIGatewayProxyResponse {
	fmt.Printf("Got query params: %+v\n", request.QueryStringParameters)
	code := request.QueryStringParameters["code"]
	// state := request.QueryStringParameters["state"]
	if errorParam, ok := request.QueryStringParameters["error"]; ok {
		fmt.Println("Got an error param: ", errorParam)
	}

	spotifyTokenEndpoint := "https://accounts.spotify.com/api/token"
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", CALLBACK_URI)
	data.Set("client_id", os.Getenv("SPOTIFY_CLIENT_ID"))
	data.Set("client_secret", os.Getenv("SPOTIFY_CLIENT_SECRET"))

	client := &http.Client{}
	r, _ := http.NewRequest("POST", spotifyTokenEndpoint, strings.NewReader(data.Encode())) // URL-encoded payload
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))

	resp, err := client.Do(r)
	if err != nil {
		return newMessageResponse(500, err.Error()).toAPIGatewayProxyResponse()
	}

	fullBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return newMessageResponse(500, err.Error()).toAPIGatewayProxyResponse()
	}

	fmt.Println("Got body: ", string(fullBody))
	return newMessageResponse(200, "Hello world").toAPIGatewayProxyResponse()
}

func authorizeSpotifyHandler(parameters map[string]string, request events.APIGatewayProxyRequest) *events.APIGatewayProxyResponse {
	var queryParams url.Values = url.Values{}
	queryParams.Add("client_id", os.Getenv("SPOTIFY_CLIENT_ID"))
	queryParams.Add("response_type", "code")
	queryParams.Add("redirect_uri", CALLBACK_URI)
	queryParams.Add("scope", "user-read-private user-read-email")

	headers := make(map[string]string)
	headers["Location"] = fmt.Sprintf("https://accounts.spotify.com/authorize?%v", queryParams.Encode())
	return &events.APIGatewayProxyResponse{
		StatusCode: 302,
		Headers:    headers,
	}
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

	b.Handle(bouncer.Post, "/leagues/{leagueName}/themes/{themeId}/songs", authMiddleware(postSongsHandler))
	b.Handle(bouncer.Post, "/leagues/{leagueName}/themes/{themeId}/votes", authMiddleware(postVotesHandler))
	b.Handle(bouncer.Get, "/leagues/{leagueName}/games/{gameId}", authMiddleware(getGamesHandler))
	b.Handle(bouncer.Get, "/spotify", authMiddleware(authorizeSpotifyHandler))
	b.Handle(bouncer.Get, "/callback", authMiddleware(callbackHandler))

	return b.Route(request)
}

func main() {
	lambda.Start(JockeyHandler)
}
