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

var redirectURI = os.Getenv("REDIRECT_URI") // TODO: make configurable?

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

func (s *Spotify) Authenticate() chan (bool) {
	srv := &http.Server{Addr: fmt.Sprintf(":%s", os.Getenv("SERVER_PORT"))}

	handler, channel := s.completeAuth(srv)
	http.HandleFunc("/spotify/callback", handler)

	go srv.ListenAndServe()

	return channel
}

func (s *Spotify) Search(song Song) []Result {
	results, err := s.client.Search(context.Background(), fmt.Sprintf("track:%s artist:%s", song.Title, song.Artist), spotify.SearchTypeTrack)
	if err != nil {
		log.Println(err.Error())
		return nil
	}

	if results.Tracks == nil || results.Tracks.Tracks == nil || len(results.Tracks.Tracks) == 0 {
		return nil
	}

	return utils.Map(results.Tracks.Tracks, func(track spotify.FullTrack) Result {
		return Result{
			Title:   track.Name,
			Artists: artistsToString(track.Artists),
			URI:     string(track.URI),
		}
	})
}

func (s *Spotify) PlaySong(songUri SongURI) GenericResult {
	uri := spotify.URI(songUri.URI)

	s.client.PlayOpt(context.Background(), &spotify.PlayOptions{
		URIs: []spotify.URI{uri},
	})

	return GenericResult{
		Status: "Success",
	}
}

func (s *Spotify) QueueSong(songID SongID) GenericResult {
	// Trim prefix in case the LLM passes the full URI
	id := spotify.ID(strings.Replace(songID.ID, "spotify:track:", "", 1))

	s.client.QueueSong(context.Background(), id)

	return GenericResult{
		Status: "Success",
	}
}

// func (s *Spotify) SearchAndPlaySongs(songs []Song) {
// 	for _, song := range songs {
// 		if id, ok := s.searchSong(song); ok {
// 			s.client.QueueSong(context.Background(), *id)
// 		}
// 	}
// }

func (s *Spotify) getQueue() []Song {
	queue, _ := s.client.GetQueue(context.Background())
	if queue == nil || len(queue.Items) == 0 {
		return nil
	}

	var songs []Song
	for _, song := range queue.Items {
		songs = append(songs, Song{
			Title:  song.Name,
			Artist: artistsToString(song.Artists),
		})
	}

	return songs
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

func artistsToString(artists []spotify.SimpleArtist) string {
	artistNames := make([]string, len(artists))
	for i, artist := range artists {
		artistNames[i] = artist.Name
	}

	return strings.Join(artistNames, ", ")
}

// TODO: look at combining into a single GetQueue call
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
		currentSong = &Song{
			Title:  curr.Item.Name,
			Artist: artistsToString(curr.Item.Artists),
		}
	}

	return PlayerState{
		CurrentSong: currentSong,
		Playing:     curr.Playing,
		Queue:       s.getQueue(),
	}
}

func (s *Spotify) completeAuth(srv *http.Server) (func(w http.ResponseWriter, r *http.Request), chan (bool)) {
	auth := spotifyauth.New(
		spotifyauth.WithClientID(os.Getenv("SPOTIFY_ID")),
		spotifyauth.WithClientSecret(os.Getenv("SPOTIFY_SECRET")),
		spotifyauth.WithRedirectURL(redirectURI),
		spotifyauth.WithScopes(spotifyauth.ScopeUserReadCurrentlyPlaying, spotifyauth.ScopeUserReadPlaybackState, spotifyauth.ScopeUserModifyPlaybackState),
	)
	channel := make(chan bool)

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
		channel <- true

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "<script>window.close('','_parent','')</script>")

		time.AfterFunc(time.Second*5, func() {
			srv.Shutdown(context.Background())
		})
	}, channel
}
