package main

import (
	"time"

	"github.com/macintoshpie/mxtp-fx/mxtpdb"
	"github.com/zmb3/spotify"
)

const redirectURI = "https://www.mxtp.xyz/.netlify/functions/jockey/callback"

var auth = spotify.NewAuthenticator(redirectURI, spotify.ScopePlaylistModifyPublic)

func NewClient(db *mxtpdb.DB, userId string) (*spotify.Client, error) {
	tok, err := db.GetSpotifyToken(userId)
	if err != nil {
		return nil, err
	}
	client := auth.NewClient(tok)

	if m, _ := time.ParseDuration("5m30s"); time.Until(tok.Expiry) < m {
		newToken, _ := client.Token()
		err := db.UpdateSpotifyToken(newToken, userId)
		if err != nil {
			return nil, err
		}
	}
	return &client, nil
}
