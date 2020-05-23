package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/google/uuid"
	"github.com/macintoshpie/mxtp-fx/bouncer"
	"github.com/macintoshpie/mxtp-fx/mxtpdb"
	"github.com/zmb3/spotify"
)

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randSeq(n int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

type Game struct {
	League           mxtpdb.League
	SubmitThemeItems mxtpdb.ThemeItems
	VoteThemeItems   mxtpdb.ThemeItems
	SpotifyAuthUrl   string
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

const oneWeekTime = time.Hour * 24 * 7
const themeDuration = oneWeekTime * 2

// themeInSubmitPhase determines if a theme is in the song submission phase
func themeInSubmitPhase(themeId string) bool {
	themeStartDate, err := time.Parse("2006-01-02", themeId)
	if err != nil {
		return false
	}

	themeEndDate := themeStartDate.Add(themeDuration)
	now := time.Now()
	return (now.After(themeStartDate) || now.Equal(themeStartDate)) &&
		(now.Before(themeEndDate) || now.Equal(themeEndDate))
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

	// verify the song can still be updated
	if !themeInSubmitPhase(themeId) {
		return newMessageResponse(400, "Submission phase for theme is over").toAPIGatewayProxyResponse()
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

	parsedUrl, err := url.Parse(song.SongUrl)
	if err != nil {
		fmt.Println("ERROR: failed to parse song url: ", err.Error())
		return newMessageResponse(400, "Bad song").toAPIGatewayProxyResponse()
	}

	// TODO: use a regex to grab the track ID (and make sure it's a track resource)
	// TODO: consider making request to spotify to verify track ID
	song.SpotifyTrackId = ""
	if parsedUrl.Host == "open.spotify.com" {
		pathParts := strings.Split(parsedUrl.Path, "/")
		if len(pathParts) > 0 {
			song.SpotifyTrackId = pathParts[len(pathParts)-1]

			// get track info from spotify
			spotifyTrackId := spotify.ID(song.SpotifyTrackId)
			client, err := NewClient(db, username)
			if err != nil {
				fmt.Println("ERROR: failed to initialize spotify client: ", err.Error())
				return newMessageResponse(500, "Internal Server Error").toAPIGatewayProxyResponse()
			}
			track, err := client.GetTrack(spotifyTrackId)
			if err != nil {
				if err != nil {
					fmt.Println("ERROR: failed to get track: ", err.Error())
					return newMessageResponse(400, "Failed to get spotify track id "+song.SpotifyTrackId).toAPIGatewayProxyResponse()
				}
			}

			song.Name = track.Name
			var artistNames []string
			for _, artist := range track.Artists {
				artistNames = append(artistNames, artist.Name)
			}
			song.Artists = artistNames
		}
	}

	err = db.UpdateSong(
		leagueName,
		themeId,
		username,
		song.SongUrl,
		uuid.New().String(),
		song.SpotifyTrackId,
		song.Name,
		song.Artists,
	)
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

	// TODO: change this to check if the requesting user is admin of league? Also this shouldn't be a part of the Games response...
	// maybe not necessary if they auth when creating league, but helpful for now I suppose
	spotifyAuthUrl := ""
	if username == "ted@devetry.com" {
		// generate an auth url
		state := randSeq(30)
		err = db.UpdateUserState(username, state)
		if err != nil {
			fmt.Println("ERROR: failed to update user state: ", err.Error())
			return newMessageResponse(500, "Internal Server Error").toAPIGatewayProxyResponse()
		}

		spotifyAuthUrl = Auth.AuthURL(state)
	}

	response := jsonResponse{
		content: Game{
			League:           league,
			SubmitThemeItems: submitThemeItems,
			VoteThemeItems:   voteThemeItems,
			SpotifyAuthUrl:   spotifyAuthUrl,
		},
		status: 200,
	}

	return response.toAPIGatewayProxyResponse()
}

func callbackHandler(parameters map[string]string, request events.APIGatewayProxyRequest) *events.APIGatewayProxyResponse {
	code := request.QueryStringParameters["code"]
	state := request.QueryStringParameters["state"]
	if code == "" || state == "" {
		return newMessageResponse(500, "Missing code or state in request query params").toAPIGatewayProxyResponse()
	}

	// get the user associated with the state
	db, err := mxtpdb.New()
	if err != nil {
		fmt.Println("ERROR: ", err.Error())
		return newMessageResponse(500, "Internal Server Error").toAPIGatewayProxyResponse()
	}

	username, err := db.GetUserFromState(state)
	if err != nil || username == "" {
		fmt.Println("ERROR: failed to get user: ", err.Error())
		return newMessageResponse(400, "Unknown state provided").toAPIGatewayProxyResponse()
	}

	// exchange the code for a token and put it in the database
	token, err := Auth.Exchange(code)
	err = db.UpdateSpotifyToken(token, username)
	if err != nil {
		fmt.Println("ERROR: failed to update token: ", err.Error())
		return newMessageResponse(500, "Internal Server Error").toAPIGatewayProxyResponse()
	}

	return newMessageResponse(200, "Success").toAPIGatewayProxyResponse()
}

func postBuildPlaylistHandler(parameters map[string]string, request events.APIGatewayProxyRequest) *events.APIGatewayProxyResponse {
	if _, ok := parameters["ADMIN"]; !ok {
		return &events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       "Unauthorized",
		}
	}

	// get the playlist id
	leagueName := parameters["leagueName"]
	if leagueName == "" {
		fmt.Println("ERROR: Parameter 'leagueName' not found")
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
	if league.SpotifyPlaylistId == "" {
		fmt.Println("ERROR: league's spotify playlist id does not exist")
		return newMessageResponse(500, "Internal Server Error").toAPIGatewayProxyResponse()
	}
	playlistId := spotify.ID(league.SpotifyPlaylistId)

	// setup our spotify client
	// TODO: hardcoded my id, once using user roles we can actually get clients by users
	client, err := NewClient(db, "ted@devetry.com")
	if err != nil {
		fmt.Println("ERROR: ", err.Error())
		return newMessageResponse(500, "Internal Server Error").toAPIGatewayProxyResponse()
	}

	// update the playlist description
	themeDescription := fmt.Sprintf("%v - %v", league.SubmitTheme.Name, league.SubmitTheme.Description)
	err = client.ChangePlaylistDescription(playlistId, themeDescription)
	if err != nil {
		fmt.Println("ERROR: ", err.Error())
		return newMessageResponse(500, "Internal Server Error").toAPIGatewayProxyResponse()
	}

	// get existing track IDs so we can remove them from the playlist
	trackPage, err := client.GetPlaylistTracks(playlistId)
	if err != nil {
		fmt.Println("ERROR: ", err.Error())
		return newMessageResponse(500, "Internal Server Error").toAPIGatewayProxyResponse()
	}

	var existingTrackIDs []spotify.ID
	for _, playlistTrack := range trackPage.Tracks {
		existingTrackIDs = append(existingTrackIDs, playlistTrack.Track.SimpleTrack.ID)
	}

	if len(existingTrackIDs) > 0 {
		_, err := client.RemoveTracksFromPlaylist(playlistId, existingTrackIDs...)
		if err != nil {
			fmt.Println("ERROR: ", err.Error())
			return newMessageResponse(500, "Internal Server Error").toAPIGatewayProxyResponse()
		}
	}

	// add new tracks to the playlist
	themeItems, err := db.GetThemeItems(league.Name, league.SubmitTheme.Date)
	if err != nil {
		fmt.Println("ERROR: ", err.Error())
		return newMessageResponse(500, "Internal Server Error").toAPIGatewayProxyResponse()
	}
	for _, song := range themeItems.Songs {
		if song.SpotifyTrackId == "" {
			continue
		}

		_, err := client.AddTracksToPlaylist(playlistId, spotify.ID(song.SpotifyTrackId))
		if err != nil {
			fmt.Printf("WARNING: skipping SubmissionId %v due to error: %v\n", song.SubmissionId, err.Error())
		}
	}

	return newMessageResponse(200, "Successfully updated playlist").toAPIGatewayProxyResponse()
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

	b.Handle(bouncer.Post, "/leagues/{leagueName}/buildPlaylist", authMiddleware(postBuildPlaylistHandler))
	b.Handle(bouncer.Post, "/leagues/{leagueName}/themes/{themeId}/songs", authMiddleware(postSongsHandler))
	b.Handle(bouncer.Post, "/leagues/{leagueName}/themes/{themeId}/votes", authMiddleware(postVotesHandler))
	b.Handle(bouncer.Get, "/leagues/{leagueName}/games/{gameId}", authMiddleware(getGamesHandler))
	b.Handle(bouncer.Get, "/callback", authMiddleware(callbackHandler))

	return b.Route(request)
}

func main() {
	lambda.Start(JockeyHandler)
}
