package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jsdoublel/rich-go/client"
)

const (
	DiscordAppID = "1521900099993600020"
	PollInterval = 15 * time.Second
)

var APIKey = ""

type LastFmResponse struct {
	RecentTracks struct {
		Track []Track `json:"track"`
	} `json:"recenttracks"`
}

type Track struct {
	Name   string `json:"name"`
	Artist struct {
		Text string `json:"#text"`
	} `json:"artist"`
	Album struct {
		Text string `json:"#text"`
	} `json:"album"`
	Image []struct {
		Size string `json:"size"`
		Text string `json:"#text"`
	} `json:"image"`
	Attr *struct {
		NowPlaying string `json:"nowplaying"`
	} `json:"@attr,omitempty"`
}

func CurTrackURL(username string) string {
	if APIKey == "" {
		panic("No API Key")
	}
	return fmt.Sprintf("/2.0/?method=user.getrecenttracks&user=%s&api_key=%s&format=json&limit=1", username, APIKey)
}

func Usage() {
	fmt.Printf("Usage: %s <username>\n", os.Args[0])
}

// Parses command line argument(s) (i.e., username)
func parseArgs() string {
	if len(os.Args) != 2 {
		Usage()
		os.Exit(1)
	}
	return os.Args[1]
}

func main() {
	apiKey, set := os.LookupEnv("LASTFM_API_KEY")
	if !set {
		log.Fatalln("Environmental variable LASTFM_API_KEY must be set. Stopping!")
	}
	APIKey = apiKey
	if err := runRPC(parseArgs()); err != nil {
		log.Fatalf("lastfm-rpc failed, error: %s\n", err)
	}
}

func runRPC(username string) error {
	err := client.Login(DiscordAppID)
	if err != nil {
		return err
	}
	defer client.Logout()
	log.Printf("connected to Discord RPC")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, syscall.SIGABRT, os.Interrupt)

	ticker := time.NewTicker(PollInterval)
	defer ticker.Stop()

	updateRPC(username) // avoid delay for first update
	for {
		select {
		case <-ticker.C:
			updateRPC(username)
		case <-sc:
			log.Print("shutting down")
			return nil
		}

	}
}

// Updates Discord RPC based on current track being played.
func updateRPC(username string) {
	track, err := getCurrentTrack(username)
	if err != nil {
		log.Printf("error fetching current track, %v", err)
		return
	}
	if track == nil {
		if err := client.SetActivity(client.Activity{}); err != nil {
			log.Printf("error clearing Discord RPC, %v", err)
		}
		return
	}
	err = client.SetActivity(client.Activity{
		Type:              client.ActivityTypeListening,
		StatusDisplayType: client.StatusDisplayTypeDetails,
		Details:           track.Name,
		State:             track.Artist.Text,
		LargeImage:        getAlbumArtURL(*track),
		LargeText:         track.Album.Text,
		SmallImage:        "lastfm-icon",
		SmallText:         "last.fm",
	})
	if err != nil {
		log.Printf("error setting Discord RPC to track %s, %v", track.Name, err)
		return
	}
}

func getCurrentTrack(username string) (*Track, error) {
	url := fmt.Sprintf("http://ws.audioscrobbler.com/2.0/?method=user.getrecenttracks&user=%s&api_key=%s&format=json&limit=1", username, APIKey)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("last.fm API returned status %d", resp.StatusCode)
	}
	var data LastFmResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	if len(data.RecentTracks.Track) == 0 {
		return nil, nil
	}
	latestTrack := data.RecentTracks.Track[0]
	if latestTrack.Attr != nil && latestTrack.Attr.NowPlaying == "true" {
		return &latestTrack, nil
	}
	return nil, nil
}

func getAlbumArtURL(track Track) (artworkURL string) {
	for _, img := range track.Image {
		artworkURL = img.Text
		if img.Size == "extralarge" && img.Text != "" {
			break
		}
	}
	return
}
