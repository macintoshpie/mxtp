package main

import (
	"github.com/macintoshpie/mxtp-fx/mxtpdb"
	"github.com/zmb3/spotify"
)

const redirectURI = "https://www.mxtp.xyz/.netlify/functions/jockey/callback"

var Auth = spotify.NewAuthenticator(redirectURI, spotify.ScopePlaylistModifyPublic)

func NewClient(db *mxtpdb.DB, userId string) (*spotify.Client, error) {
	tok, err := db.GetSpotifyToken(userId)
	if err != nil {
		return nil, err
	}

	client := Auth.NewClient(tok)

	return &client, nil
}
