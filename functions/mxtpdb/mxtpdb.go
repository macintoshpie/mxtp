package mxtpdb

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/guregu/dynamo"
	"golang.org/x/oauth2"
)

type DB struct {
	db    *dynamo.DB
	table dynamo.Table
}

type MxtpItem struct {
	PK                string
	SK                string
	Name              string   `dynamo:",omitempty"`
	Date              string   `dynamo:",omitempty"`
	Description       string   `dynamo:",omitempty"`
	SpotifyPlaylistId string   `dynamo:",omitempty"`
	UserId            string   `dynamo:",omitempty"`
	SubmissionId      string   `dynamo:",omitempty"`
	SubmissionIds     []string `dynamo:",omitempty,set"`
	SongUrl           string   `dynamo:",omitempty"`
	SpotifyTrackId    string   `dynamo:",omitempty"`
	Artists           []string `dynamo:",omitempty"`
	Role              string   `dynamo:",omitempty"`

	AccessToken  string    `dynamo:",omitempty"`
	TokenType    string    `dynamo:",omitempty"`
	RefreshToken string    `dynamo:",omitempty"`
	Expiry       time.Time `dynamo:",omitempty"`

	State string `dynamo:",omitempty"`
}

type League struct {
	Name              string `dynamo:",omitempty"`
	Description       string `dynamo:",omitempty"`
	SubmitTheme       Theme  `dynamo:",omitempty"`
	VoteTheme         Theme  `dynamo:",omitempty"`
	SpotifyPlaylistId string `dynamo:",omitempty"`
}

type Theme struct {
	Name        string `dynamo:",omitempty"`
	Description string `dynamo:",omitempty"`
	Date        string `dynamo:",omitempty"`
}

type ThemeItems struct {
	Id    string
	Songs []Song  `dynamo:",omitempty"`
	Votes []Votes `dynamo:",omitempty"`
}

type Song struct {
	UserId         string   `dynamo:",omitempty"`
	SubmissionId   string   `dynamo:",omitempty"`
	SongUrl        string   `dynamo:",omitempty"`
	SpotifyTrackId string   `dynamo:",omitempty"`
	Name           string   `dynamo:",omitempty"`
	Artists        []string `dynamo:",omitempty"`
}

type Votes struct {
	UserId        string   `dynamo:",omitempty"`
	SubmissionIds []string `dynamo:",omitempty"`
}

func validateCompoundKey(key string, expectedParts ...string) error {
	keyParts := strings.Split(key, "#")
	if len(keyParts) != len(expectedParts)*2 {
		return errors.New(fmt.Sprintf("Expected %v to have %v parts", key, len(expectedParts)*2))
	}

	idx := 0
	for idx*2 < len(keyParts) && idx < len(expectedParts) {
		actualPart := keyParts[idx*2]
		expectedPart := expectedParts[idx]
		if actualPart != expectedPart {
			return errors.New(fmt.Sprintf("Expected %v to be %v", actualPart, expectedPart))
		}
		idx += 1
	}

	return nil
}

func (item *MxtpItem) toLeague() (League, error) {
	err := validateCompoundKey(item.PK, "league")
	if err != nil {
		return League{
			SubmitTheme: Theme{},
			VoteTheme:   Theme{},
		}, errors.New(fmt.Sprintf("Failed to validate League: %v", err.Error()))
	}

	if item.SK != "~meta" {
		return League{
			SubmitTheme: Theme{},
			VoteTheme:   Theme{},
		}, errors.New(fmt.Sprintf("Failed to vaildate League: Expected SK to be ~meta but was %v", item.SK))
	}

	return League{
		Name:              item.Name,
		Description:       item.Description,
		SpotifyPlaylistId: item.SpotifyPlaylistId,
		SubmitTheme:       Theme{},
		VoteTheme:         Theme{},
	}, nil
}

func (item *MxtpItem) toTheme() (Theme, error) {
	result := Theme{}
	err := validateCompoundKey(item.PK, "league")
	if err != nil {
		return result, errors.New(fmt.Sprintf("Failed to validate Theme: %v", err.Error()))
	}

	err = validateCompoundKey(item.SK, "theme")
	if err != nil {
		return result, errors.New(fmt.Sprintf("Failed to validate Theme: %v", err.Error()))
	}

	return Theme{
		Name:        item.Name,
		Description: item.Description,
		Date:        item.Date,
	}, nil
}

func (item *MxtpItem) toSong() (Song, error) {
	err := validateCompoundKey(item.PK, "league", "theme")
	if err != nil {
		return Song{}, errors.New(fmt.Sprintf("Failed to validate Song: %v", err.Error()))
	}

	err = validateCompoundKey(item.SK, "song")
	if err != nil {
		return Song{}, errors.New(fmt.Sprintf("Failed to validate Song: %v", err.Error()))
	}

	return Song{
		UserId:         item.UserId,
		SubmissionId:   item.SubmissionId,
		SongUrl:        item.SongUrl,
		SpotifyTrackId: item.SpotifyTrackId,
		Name:           item.Name,
		Artists:        item.Artists,
	}, nil
}

func (item *MxtpItem) toVotes() (Votes, error) {
	err := validateCompoundKey(item.PK, "league", "theme")
	if err != nil {
		return Votes{}, errors.New(fmt.Sprintf("Failed to validate Votes: %v", err.Error()))
	}

	err = validateCompoundKey(item.SK, "votes")
	if err != nil {
		return Votes{}, errors.New(fmt.Sprintf("Failed to validate Votes: %v", err.Error()))
	}

	return Votes{
		UserId:        item.UserId,
		SubmissionIds: item.SubmissionIds,
	}, nil
}

func (item *MxtpItem) toOAuthToken() (*oauth2.Token, error) {
	err := validateCompoundKey(item.PK, "secret")
	if err != nil {
		return &oauth2.Token{}, err
	}

	return &oauth2.Token{
		AccessToken:  item.AccessToken,
		TokenType:    item.TokenType,
		RefreshToken: item.RefreshToken,
		Expiry:       item.Expiry,
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
		db.Table("mxtp"),
	}, nil
}

func makeLeaguePK(leagueName string) (string, error) {
	err := validateIds(leagueName)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("league#%v", leagueName), nil
}

func (db *DB) GetLeague(leagueName string) (League, error) {
	pk, err := makeLeaguePK(leagueName)
	if err != nil {
		return League{}, err
	}

	var items []MxtpItem
	err = db.table.Get("PK", pk).
		Order(dynamo.Descending).
		Limit(3).
		All(&items)

	if err != nil {
		return League{
			SubmitTheme: Theme{},
			VoteTheme:   Theme{},
		}, err
	}

	// Expect the items to be the following:
	// [0] - League info
	// [1] - Submitting theme (if exists)
	// [2] - Voting theme (if exists)
	if len(items) == 0 {
		return League{
			SubmitTheme: Theme{},
			VoteTheme:   Theme{},
		}, errors.New("No items found for league")
	}

	league, err := items[0].toLeague()
	if err != nil {
		return League{
			SubmitTheme: Theme{},
			VoteTheme:   Theme{},
		}, err
	}

	if len(items) == 1 {
		// no themes yet for league
		return league, nil
	}
	submitTheme, err := items[1].toTheme()
	if err != nil {
		return league, err
	}
	league.SubmitTheme = submitTheme

	if len(items) == 2 {
		// only one theme (submit theme)
		return league, nil
	}
	voteTheme, err := items[2].toTheme()
	if err != nil {
		return league, err
	}
	league.VoteTheme = voteTheme

	return league, nil
}

func makeSongKeys(leagueName, themeId, userId string) (pk, sk string, err error) {
	err = validateIds(leagueName, themeId, userId)
	if err != nil {
		return "", "", nil
	}

	pk, err = makeThemeItemsPK(leagueName, themeId)
	if err != nil {
		return "", "", err
	}

	sk = fmt.Sprintf("song#%v", userId)
	return pk, sk, err
}

func makeVotesKeys(leagueName, themeId, userId string) (pk, sk string, err error) {
	err = validateIds(leagueName, themeId, userId)
	if err != nil {
		return "", "", nil
	}

	pk, err = makeThemeItemsPK(leagueName, themeId)
	if err != nil {
		return "", "", err
	}

	sk = fmt.Sprintf("votes#%v", userId)
	return pk, sk, err
}

func (db *DB) GetSong(leagueName, themeId, userId string) (Song, error) {
	pk, sk, err := makeSongKeys(leagueName, themeId, userId)
	if err != nil {
		return Song{}, err
	}
	var item MxtpItem
	err = db.table.Get("PK", pk).
		Range("SK", dynamo.Equal, sk).
		One(&item)

	if err != nil {
		return Song{}, err
	}

	song, err := item.toSong()
	if err != nil {
		return Song{}, err
	}

	return song, nil
}

func validateIds(ids ...string) error {
	var badIds []string
	for _, id := range ids {
		if strings.Contains(id, "#") {
			badIds = append(badIds, id)
		}
	}

	if len(badIds) == 0 {
		return nil
	}

	return errors.New("Invalid ids: " + strings.Join(badIds, ","))
}

func makeThemeItemsPK(leagueName, themeId string) (string, error) {
	err := validateIds(leagueName, themeId)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("league#%v#theme#%v", leagueName, themeId), nil
}

func (db *DB) GetThemeItems(leagueName, themeId string) (ThemeItems, error) {
	pk, err := makeThemeItemsPK(leagueName, themeId)
	if err != nil {
		return ThemeItems{}, err
	}

	var items []MxtpItem
	err = db.table.Get("PK", pk).
		All(&items)

	if err != nil {
		return ThemeItems{}, err
	}

	// Expected order of results
	// [0:x] - Songs
	// [x:] - Votes
	if len(items) == 0 {
		// there were no songs or votes for the theme
		return ThemeItems{}, nil
	}

	// try to parse items as songs until we can't anymore
	songs := []Song{}
	idx := 0
	for idx < len(items) {
		song, err := items[idx].toSong()
		if err != nil {
			break
		}
		songs = append(songs, song)
		idx += 1
	}

	// parse remaining items as Votes
	votes := []Votes{}
	for _, item := range items[idx:] {
		vote, err := item.toVotes()
		if err != nil {
			return ThemeItems{}, err
		}
		votes = append(votes, vote)
	}

	return ThemeItems{
		Id:    themeId,
		Songs: songs,
		Votes: votes,
	}, nil
}

func (db *DB) UpdateSong(leagueName, themeId, userId, songUrl, submissionId, spotifyTrackId, songName string, songArtists []string) error {
	pk, sk, err := makeSongKeys(leagueName, themeId, userId)
	if err != nil {
		return err
	}

	song := MxtpItem{
		PK:             pk,
		SK:             sk,
		UserId:         userId,
		SongUrl:        songUrl,
		SubmissionId:   submissionId,
		SpotifyTrackId: spotifyTrackId,
		Name:           songName,
		Artists:        songArtists,
	}

	return db.table.Put(song).Run()
}

func (db *DB) UpdateVotes(leagueName, themeId, userId string, submissionIds []string) error {
	pk, sk, err := makeVotesKeys(leagueName, themeId, userId)
	if err != nil {
		return err
	}
	song := MxtpItem{
		PK:            pk,
		SK:            sk,
		UserId:        userId,
		SubmissionIds: submissionIds,
	}

	return db.table.Put(song).Run()
}

func makeSpotifyTokenKeys(userId string) (pk, sk string, err error) {
	err = validateIds(userId)
	if err != nil {
		return "", "", nil
	}

	pk = fmt.Sprintf("secret#%v", userId)
	sk = fmt.Sprintf("spotify")
	return pk, sk, err
}

func (db *DB) GetSpotifyToken(userId string) (*oauth2.Token, error) {
	pk, sk, err := makeSpotifyTokenKeys(userId)
	if err != nil {
		return nil, err
	}

	var item MxtpItem
	err = db.table.Get("PK", pk).
		Range("SK", dynamo.Equal, sk).
		One(&item)
	if err != nil {
		return nil, err
	}

	token, err := item.toOAuthToken()
	if err != nil {
		return nil, err
	}
	return token, nil
}

func (db *DB) UpdateSpotifyToken(token *oauth2.Token, userId string) error {
	pk, sk, err := makeSpotifyTokenKeys(userId)
	if err != nil {
		return err
	}

	tokenItem := MxtpItem{
		PK:           pk,
		SK:           sk,
		AccessToken:  token.AccessToken,
		TokenType:    token.TokenType,
		RefreshToken: token.RefreshToken,
		Expiry:       token.Expiry,
	}

	return db.table.Put(tokenItem).Run()
}

func makeUserStateKeys(state string) (pk, sk string, err error) {
	err = validateIds(state)
	if err != nil {
		return "", "", nil
	}

	pk = fmt.Sprintf("state#%v", state)
	sk = fmt.Sprintf("state")
	return pk, sk, err
}

func (db *DB) GetUserFromState(state string) (string, error) {
	pk, sk, err := makeUserStateKeys(state)
	if err != nil {
		return "", err
	}

	var item MxtpItem
	err = db.table.Get("PK", pk).
		Range("SK", dynamo.Equal, sk).
		One(&item)
	if err != nil {
		return "", err
	}

	return item.UserId, nil
}

func (db *DB) UpdateUserState(userId, state string) error {
	pk, sk, err := makeUserStateKeys(state)
	if err != nil {
		return err
	}

	tokenItem := MxtpItem{
		PK:     pk,
		SK:     sk,
		UserId: userId,
	}

	return db.table.Put(tokenItem).Run()
}
