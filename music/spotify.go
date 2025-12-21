package music

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/dkaps125/juke/utils"
	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
)

const redirectURI = "http://127.0.0.1:8080/callback" // TODO: make configurable?

type Spotify struct {
	client *spotify.Client
}

var (
	ch    = make(chan *spotify.Client)
	state = "abc123" // TODO: make random
)

func NewSpotify() *Spotify {
	return &Spotify{}
}

func (s *Spotify) Authenticate() {
	srv := &http.Server{Addr: ":8080"}

	http.HandleFunc("/callback", s.completeAuth(srv))

	srv.ListenAndServe()
}

func (s *Spotify) SearchAndPlaySong(song Song) {
	results, err := s.client.Search(context.Background(), fmt.Sprintf("%s %s", song.Title, song.Artist), spotify.SearchTypeTrack)
	if err != nil {
		log.Println(err.Error())
		return
	}

	toPlay := results.Tracks.Tracks[0].URI

	s.client.PlayOpt(context.Background(), &spotify.PlayOptions{
		URIs: []spotify.URI{toPlay},
	})
}

func (s Spotify) Pause() {
	s.client.Pause(context.Background())
}

func (s Spotify) Play() {
	s.client.Play(context.Background())
}

func (s Spotify) Next() {
	s.client.Next(context.Background())
}

func (s Spotify) Previous() {
	s.client.Previous(context.Background())
}

func (s Spotify) CurrentState() PlayerState {
	if s.client == nil {
		return PlayerState{
			CurrentSong: nil,
			Playing:     false,
		}
	}

	curr, _ := s.client.PlayerCurrentlyPlaying(context.Background())

	if curr == nil || curr.Item == nil {
		return PlayerState{
			CurrentSong: nil,
			Playing:     false,
		}
	}

	var currentSong *Song = nil
	if curr.Item != nil {
		artistNames := make([]string, len(curr.Item.Artists))
		for i, artist := range curr.Item.Artists {
			artistNames[i] = artist.Name
		}

		currentSong = &Song{
			Title:  curr.Item.Name,
			Artist: strings.Join(artistNames, ", "),
		}
	}

	return PlayerState{
		CurrentSong: currentSong,
		Playing:     curr.Playing,
	}
}

func (s *Spotify) completeAuth(srv *http.Server) func(w http.ResponseWriter, r *http.Request) {
	auth := spotifyauth.New(
		spotifyauth.WithClientID(os.Getenv("SPOTIFY_ID")),
		spotifyauth.WithClientSecret(os.Getenv("SPOTIFY_SECRET")),
		spotifyauth.WithRedirectURL(redirectURI),
		spotifyauth.WithScopes(spotifyauth.ScopeUserReadCurrentlyPlaying, spotifyauth.ScopeUserReadPlaybackState, spotifyauth.ScopeUserModifyPlaybackState),
	)

	// In the background, fetch the auth URL and open it in the browser
	go func() {
		url := auth.AuthURL(state)

		utils.OpenURL(url)
	}()

	return func(w http.ResponseWriter, r *http.Request) {
		tok, err := auth.Token(r.Context(), state, r)
		if err != nil {
			http.Error(w, "Couldn't get token", http.StatusForbidden)
			log.Fatal(err)
		}
		if st := r.FormValue("state"); st != state {
			http.NotFound(w, r)
			log.Fatalf("State mismatch: %s != %s\n", st, state)
		}
		// use the token to get an authenticated client
		s.client = spotify.New(auth.Client(r.Context(), tok))

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "<script>window.close('','_parent','')</script>")

		time.AfterFunc(time.Second*5, func() {
			srv.Shutdown(context.Background())
		})
	}
}
