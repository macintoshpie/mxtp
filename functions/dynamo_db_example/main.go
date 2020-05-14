package main

import (
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/macintoshpie/mxtp-fx/mxtpdb"
)

func DynamoHandler(request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	err, db := mxtpdb.New()
	if err != nil {
		return &events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       "Some error, " + err.Error(),
		}, nil
	}

	err, theme, _ := db.GetThemeAndSubmissions("devetry", "123")
	if err != nil {
		return &events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       "Some error, " + err.Error(),
		}, nil
	}

	return &events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       "Hello, " + theme.Name,
	}, nil
}

func main() {
	// Make the handler available for Remote Procedure Call by AWS Lambda
	// lambda.Start(DynamoHandler)

	err, db := mxtpdb.New()
	if err != nil {
		fmt.Println(err.Error())
	}

	err, theme, submissions := db.GetThemeAndSubmissions("devetry", "123")
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	fmt.Println("Theme: ", theme.Name)
	fmt.Println("Submissions:")
	for _, submission := range submissions {
		fmt.Println(submission.SongUrl)
	}

	// add/update submission
	newSubmission := mxtpdb.Submission{
		UserId:  "test2@devetry.com",
		SongUrl: "http://www.spotify.com/123",
		Votes:   []string{"vote1", "vote2"},
	}
	if err := db.PutSubmission("devetry", "123", newSubmission); err != nil {
		fmt.Println(err.Error())
		return
	}

	// update votes
	if err := db.UpdateSubmission("devetry", "123", "test2@devetry.com", "", &[]string{"abc", "123"}); err != nil {
		fmt.Println(err.Error())
		return
	}

	// get theme and submissions again to verify it was updated
	err, theme, submissions = db.GetThemeAndSubmissions("devetry", "123")
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	fmt.Println("Theme: ", theme.Name)
	fmt.Println("Submissions:")
	for _, submission := range submissions {
		fmt.Println(submission.SongUrl, submission.Votes)
	}

	// add/update a theme
	newTheme := mxtpdb.Theme{
		Id:          "ABC",
		Name:        "A Theme",
		Description: "Theme about music",
	}
	if err := db.PutTheme("devetry", newTheme); err != nil {
		fmt.Println(err.Error())
		return
	}

	// get the theme
	err, theme, _ = db.GetThemeAndSubmissions("devetry", newTheme.Id)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	fmt.Println("New Theme: ", theme.Name)

	// update the league
	updatedLeague := mxtpdb.League{
		Name:          "devetry",
		SubmitThemeId: "submitting",
		VoteThemeId:   "voting",
	}
	if err := db.UpdateLeague(updatedLeague); err != nil {
		fmt.Println(err.Error())
		return
	}

	// get league and themes
	err, league, themes := db.GetLeagueAndThemes("devetry")
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	fmt.Println("League: ", league.Name, league.Description, league.SubmitThemeId, league.VoteThemeId)
	for _, theme := range themes {
		fmt.Println("League theme: ", theme.Id, theme.Name, theme.Description)
	}
}
