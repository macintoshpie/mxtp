package mxtpdb

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/guregu/dynamo"
)

type DB struct {
	db    *dynamo.DB
	table dynamo.Table
}

type MxtpItem struct {
	PK string // partition key
	SK string // sort key

	// League items
	LeagueDescription string `dynamo:",omitempty"`
	SubmitThemeId     string `dynamo:",omitempty"`
	VoteThemeId       string `dynamo:",omitempty"`

	// Theme items
	ThemeName        string `dynamo:",omitempty"`
	ThemeDescription string `dynamo:",omitempty"`

	// Submission items
	UserId  string   `dynamo:",omitempty"`
	SongUrl string   `dynamo:",omitempty"`
	Votes   []string `dynamo:",omitempty"`

	// Global Secondary Key 1
	GSI1PK string `dynamo:",omitempty"`
	GSI1SK string `dynamo:",omitempty"`
}

type League struct {
	Name          string
	Description   string
	SubmitThemeId string
	VoteThemeId   string
}

type Theme struct {
	Id          string
	Name        string
	Description string
}

type Submission struct {
	UserId  string
	SongUrl string
	Votes   []string
}

func (item *MxtpItem) toLeague() (*League, error) {
	splitId := strings.Split(item.PK, "#")
	if len(splitId) != 2 {
		return nil, errors.New("Invalid league item")
	}
	if splitId[0] != "league" {
		return nil, errors.New("Invalid league item")
	}

	return &League{
		Name:          splitId[1],
		Description:   item.LeagueDescription,
		SubmitThemeId: item.SubmitThemeId,
		VoteThemeId:   item.VoteThemeId,
	}, nil
}

func (item *MxtpItem) toTheme() (*Theme, error) {
	splitId := strings.Split(item.SK, "#")
	if len(splitId) != 2 {
		return nil, errors.New("Invalid theme item")
	}
	if splitId[0] != "theme" {
		return nil, errors.New("Invalid theme item")
	}

	return &Theme{
		Id:          splitId[1],
		Name:        item.ThemeName,
		Description: item.ThemeDescription,
	}, nil
}

func (item *MxtpItem) toSubmission() (*Submission, error) {
	splitId := strings.Split(item.SK, "#")
	if len(splitId) != 2 {
		return nil, errors.New("Invalid submission item")
	}
	if splitId[0] != "user" {
		return nil, errors.New("Invalid submission item")
	}

	return &Submission{
		UserId:  item.UserId,
		SongUrl: item.SongUrl,
		Votes:   item.Votes,
	}, nil
}

func New() (*DB, error) {
	accessKeyId := os.Getenv("PERSONAL_AWS_ACCESS_KEY_ID")
	secretAccessKey := os.Getenv("PERSONAL_AWS_SECRET_ACCESS_KEY")
	if accessKeyId == "" || secretAccessKey == "" {
		return nil, errors.New("Missing required environment vars")
	}

	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String("us-west-2"),
		Credentials: credentials.NewStaticCredentials(accessKeyId, secretAccessKey, ""),
	})

	if err != nil {
		return nil, err
	}

	db := dynamo.New(sess, &aws.Config{Region: aws.String("us-west-2")})
	return &DB{
		db,
		db.Table("mxtp_test"),
	}, nil
}

func (db *DB) GetThemeAndSubmissions(leagueName, themeId string) (*Theme, []Submission, error) {
	var items []MxtpItem
	err := db.table.Get("GSI1PK", fmt.Sprintf("league#%v#theme#%v", leagueName, themeId)).
		Index("GSI1").
		All(&items)
	if err != nil {
		return nil, []Submission{}, err
	}

	theme, err := items[0].toTheme()
	if err != nil {
		return nil, []Submission{}, err
	}
	var submissions []Submission
	for _, item := range items[1:] {
		submission, err := item.toSubmission()
		if err != nil {
			fmt.Println("ERROR: skipping submission due to error: ", err.Error())
			continue
		}
		submissions = append(submissions, *submission)
	}

	return theme, submissions, nil
}

func (db *DB) PutSubmission(leagueName, themeId, userId string, submission Submission) error {
	item := MxtpItem{
		PK:      fmt.Sprintf("theme#%v#user#%v", themeId, userId),
		SK:      fmt.Sprintf("user#%v", userId),
		GSI1PK:  fmt.Sprintf("league#%v#theme#%v", leagueName, themeId),
		GSI1SK:  fmt.Sprintf("user#%v", userId),
		SongUrl: submission.SongUrl,
		Votes:   submission.Votes,
	}
	err := db.table.Put(item).Run()
	return err
}

func (db *DB) PutTheme(leagueName string, theme Theme) error {
	item := MxtpItem{
		PK:               fmt.Sprintf("league#%v", leagueName),
		SK:               fmt.Sprintf("theme#%v", theme.Id),
		GSI1PK:           fmt.Sprintf("league#%v#theme#%v", leagueName, theme.Id),
		GSI1SK:           fmt.Sprintf("theme#%v", theme.Id),
		ThemeName:        theme.Name,
		ThemeDescription: theme.Description,
	}
	err := db.table.Put(item).Run()
	return err
}

func (db *DB) UpdateSubmission(leagueName, themeId, userId, songUrl string, votes *[]string) error {
	update := db.table.Update("PK", fmt.Sprintf("theme#%v#user#%v", themeId, userId)).
		Range("SK", fmt.Sprintf("user#%v", userId))

	if songUrl != "" {
		update = update.Set("SongUrl", songUrl)
	}

	if votes != nil {
		update = update.SetSet("Votes", votes)
	}

	return update.Run()
}

func (db *DB) UpdateLeague(league League) error {
	if league.Name == "" {
		return errors.New("League must have a name")
	}

	update := db.table.Update("PK", fmt.Sprintf("league#%v", league.Name)).
		Range("SK", fmt.Sprintf("meta#%v", league.Name))

	if league.Description != "" {
		update = update.Set("LeagueDescription", league.Description)
	}

	if league.SubmitThemeId != "" {
		update = update.Set("SubmitThemeId", league.SubmitThemeId)
	}

	if league.VoteThemeId != "" {
		update = update.Set("VoteThemeId", league.VoteThemeId)
	}

	return update.Run()
}

func (db *DB) GetLeagueAndThemes(leagueName string) (*League, []Theme, error) {
	var items []MxtpItem
	err := db.table.Get("PK", fmt.Sprintf("league#%v", leagueName)).
		All(&items)

	if err != nil {
		return nil, []Theme{}, err
	}

	if len(items) == 0 {
		return nil, []Theme{}, errors.New("No league found")
	}

	league, err := items[0].toLeague()

	var themes []Theme
	for _, item := range items[1:] {
		theme, err := item.toTheme()
		if err != nil {
			fmt.Println("ERROR: skipping theme item due to error: ", err.Error())
			continue
		}
		themes = append(themes, *theme)
	}

	return league, themes, nil
}
