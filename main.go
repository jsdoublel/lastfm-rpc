package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"slices"
	"syscall"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/adrg/xdg"
	"github.com/jsdoublel/rich-go/client"
)

const (
	DiscordAppID = "1521900099993600020"
	PollInterval = 15 * time.Second
)

var (
	discordRPC DiscordRPC
	config     Config

	configPath    = path.Join(xdg.ConfigHome, "lastfm-rpc.toml")
	defaultConfig = []byte("username=\"user name here\"\napi_key=\"api key here\"")
)

type DiscordRPC struct {
	cancel func()
}

type Config struct {
	Username string `toml:"username"`
	ApiKey   string `toml:"api_key"`
}

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

func CurTrackURL() string {
	if config.ApiKey == "" || config.Username == "" {
		panic("missing config")
	}
	return fmt.Sprintf("http://ws.audioscrobbler.com/2.0/?method=user.getrecenttracks&user=%s&api_key=%s&format=json&limit=1", config.Username, config.ApiKey)
}

func Usage() {
	fmt.Printf("Usage: %s <username>\n", os.Args[0])
}

func getConfig() error {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := os.WriteFile(configPath, []byte(defaultConfig), 0o644); err != nil {
			panic("could not write default config")
		}
		fmt.Printf("Config file created at %s. Please fill in username/api key.", configPath)
		return errors.New("config does not exist")
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Printf("error, could not read config file, %v", err)
	}
	if slices.Equal(data, defaultConfig) {
		return errors.New("config is the default config (please add username/api key)")
	}
	_, err = toml.Decode(string(data), &config)
	return err
}

func main() {
	if err := getConfig(); err != nil {
		log.Fatalf("could not load config, %v", err)
	}
	if err := runRPC(); err != nil {
		log.Fatalf("lastfm-rpc failed, error: %s\n", err)
	}
}

func runRPC() error {
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, syscall.SIGABRT, os.Interrupt)

	ticker := time.NewTicker(PollInterval)
	defer ticker.Stop()

	updateRPC() // avoid delay for first update
	for {
		select {
		case <-ticker.C:
			updateRPC()
		case <-sc:
			log.Print("shutting down")
			return nil
		}

	}
}

// Updates Discord RPC based on current track being played.
func updateRPC() {
	track, err := getCurrentTrack()
	if err != nil {
		log.Printf("error fetching current track, %v", err)
		return
	}
	if track == nil { // we're not playing anything
		if discordRPC.cancel != nil {
			log.Print("last.fm not playing, disconnecting Discord RPC")
			discordRPC.cancel()
		}
		return
	}
	if discordRPC.cancel == nil {
		log.Print("attempting to connect to Discord")
		ctx, cancel := context.WithCancel(context.Background())
		discordRPC = DiscordRPC{cancel: cancel}
		go func() {
			defer cancel()
			err := client.Login(DiscordAppID)
			if err != nil {
				log.Printf("failed to connect to discord with error %v", err)
				return
			}
			defer client.Logout()
			log.Print("successfully connected to Discord")
			if err := setRPC(track); err != nil {
				log.Printf("failed to update Discord RPC %v", err)
			}
			<-ctx.Done()
		}()
		return
	}
	if err := setRPC(track); err != nil {
		log.Printf("failed to update Discord RPC %v", err)
	}
}

func setRPC(track *Track) error {
	return client.SetActivity(client.Activity{
		Type:              client.ActivityTypeListening,
		StatusDisplayType: client.StatusDisplayTypeDetails,
		Details:           track.Name,
		State:             track.Artist.Text,
		LargeImage:        getAlbumArtURL(*track),
		LargeText:         track.Album.Text,
		SmallImage:        "lastfm-icon",
		SmallText:         "last.fm",
	})
}

func getCurrentTrack() (*Track, error) {
	resp, err := http.Get(CurTrackURL())
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
