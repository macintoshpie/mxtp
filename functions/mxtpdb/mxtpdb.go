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
	Role              string   `dynamo:",omitempty"`

	AccessToken  string    `dynamo:",omitempty"`
	TokenType    string    `dynamo:",omitempty"`
	RefreshToken string    `dynamo:",omitempty"`
	Expiry       time.Time `dynamo:",omitempty"`
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
	UserId         string `dynamo:",omitempty"`
	SubmissionId   string `dynamo:",omitempty"`
	SongUrl        string `dynamo:",omitempty"`
	SpotifyTrackId string `dynamo:",omitempty"`
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

func (db *DB) GetLeague(leagueName string) (League, error) {
	var items []MxtpItem
	err := db.table.Get("PK", fmt.Sprintf("league#%v", leagueName)).
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

func (db *DB) GetSong(leagueName, themeId, userId string) (Song, error) {
	var item MxtpItem
	err := db.table.Get("PK", fmt.Sprintf("league#%v#theme#%v", leagueName, themeId)).
		Range("SK", dynamo.Equal, fmt.Sprintf("song#%v", userId)).
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

func (db *DB) GetThemeItems(leagueName, themeId string) (ThemeItems, error) {
	var items []MxtpItem
	err := db.table.Get("PK", fmt.Sprintf("league#%v#theme#%v", leagueName, themeId)).
		// only Songs have SubmissionId, and the only other item that matches the UserId must be the user's Votes (if it exists)
		// Filter("attribute_exists(SubmissionId) OR UserId = ?", userId).
		// Order(dynamo.Descending).
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

	// try to parse the first item as Votes
	// songs := []Song{}
	// votes, err := items[0].toVotes()
	// if err != nil {
	// 	votes = Votes{}
	// 	// try to parse the first item as Song
	// 	var songItem MxtpItem
	// 	songItem, items = items[0], items[1:]
	// 	song, err := songItem.toSong()
	// 	if err != nil {
	// 		return []Song{}, Votes{}, errors.New("Failed to parse first Theme item as Votes or Song")
	// 	}
	// 	songs = append(songs, song)
	// }

	// // iterate through remaining songs
	// for _, item := range items[1:] {
	// 	song, err := item.toSong()
	// 	if err != nil {
	// 		fmt.Printf("WARNING: Failed to parse item as song (skipped): %+v\n", item)
	// 		continue
	// 	}
	// 	songs = append(songs, song)
	// }

	// return songs, votes, nil
}

func (db *DB) UpdateSong(leagueName, themeId, userId, songUrl, submissionId, spotifyTrackId string) error {
	song := MxtpItem{
		PK:             fmt.Sprintf("league#%v#theme#%v", leagueName, themeId),
		SK:             fmt.Sprintf("song#%v", userId),
		UserId:         userId,
		SongUrl:        songUrl,
		SubmissionId:   submissionId,
		SpotifyTrackId: spotifyTrackId,
	}

	return db.table.Put(song).Run()
}

func (db *DB) UpdateVotes(leagueName, themeId, userId string, submissionIds []string) error {
	song := MxtpItem{
		PK:            fmt.Sprintf("league#%v#theme#%v", leagueName, themeId),
		SK:            fmt.Sprintf("votes#%v", userId),
		UserId:        userId,
		SubmissionIds: submissionIds,
	}

	return db.table.Put(song).Run()
}

func (db *DB) GetSpotifyToken(userId string) (*oauth2.Token, error) {
	var item MxtpItem
	err := db.table.Get("PK", fmt.Sprintf("secret#%v", userId)).
		Range("SK", dynamo.Equal, "spotify").
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
	tokenItem := MxtpItem{
		PK:           fmt.Sprintf("secret#%v", userId),
		SK:           "spotify",
		AccessToken:  token.AccessToken,
		TokenType:    token.TokenType,
		RefreshToken: token.RefreshToken,
		Expiry:       token.Expiry,
	}

	return db.table.Put(tokenItem).Run()
}

// func main() {
// 	db, err := New()
// 	if err != nil {
// 		panic(err)
// 	}

// 	res1, err := db.GetLeague("devetry")
// 	if err != nil {
// 		panic(err)
// 	}
// 	fmt.Printf("%+v\n", res1)

// 	res2, err := db.GetSong("devetry", "2020-05-18", "ted@devetry.com")
// 	if err != nil {
// 		panic(err)
// 	}
// 	fmt.Printf("%+v\n", res2)

// 	items, err := db.GetThemeItems("devetry", "2020-05-04")
// 	if err != nil {
// 		panic(err)
// 	}
// 	fmt.Printf("%+v\n", items)

// 	err = db.UpdateSong("devetry", "2020-05-18", "ted@devetry.com", "http://www.youtube.com/hello", uuid.New().String())
// 	if err != nil {
// 		panic(err)
// 	}

// 	res2, err = db.GetSong("devetry", "2020-05-18", "ted@devetry.com")
// 	if err != nil {
// 		panic(err)
// 	}
// 	fmt.Printf("%+v\n", res2)

// 	err = db.UpdateVotes("devetry", "2020-05-04", "ted@devetry.com", []string{"qwerty"})
// 	if err != nil {
// 		panic(err)
// 	}

// 	items, err = db.GetThemeItems("devetry", "2020-05-04")
// 	if err != nil {
// 		panic(err)
// 	}
// 	fmt.Printf("%+v\n", items)
// }
