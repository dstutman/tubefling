package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rylio/ytdl"
	"github.com/tsdtsdtsd/identicon"
	"html/template"
	"image/png"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"
)

type Channel struct {
	Id        string     `xml:"channelId"`
	Name      string     `xml:"title"`
	URL       ChannelURL `xml:"link"`
	Published time.Time  `xml:"published"`
	Videos    []Video    `xml:"entry"`
}

type ChannelURL struct {
	Text string `xml:"href,attr"`
}

type Video struct {
	Id           string       `xml:"videoId"`
	Name         string       `xml:"title"`
	Published    time.Time    `xml:"published"`
	ThumbnailURL ThumbnailURL `xml:"group>thumbnail"`
	Description  string       `xml:"group>description"`
}

type ThumbnailURL struct {
	Text string `xml:"url,attr"`
}

type FeedTemplateContext struct {
	Channel       Channel
	ServerBaseURL string
}

func main() {
	staticDirectory := os.Getenv("STATIC_DIR")
	if len(staticDirectory) == 0 {
		staticDirectory = ""
	}
	tempDirectory := os.Getenv("TEMP_DIR")
	if len(tempDirectory) == 0 {
		tempDirectory = ""
	}
	strings.Split(os.Getenv("AUTHORIZED_TOKENS"), ",")
	server := echo.New()
	server.Use(middleware.Logger())
	server.Use(middleware.Recover())
	server.Use(middleware.Static(staticDirectory))
	feedTemplate, err := template.New("feed").Parse(`
			<rss version="2.0" xmlns:itunes="http://www.itunes.com/DTDs/Podcast-1.0.dtd" xmlns:media="http://search.yahoo.com/mrss/">
				<channel>
					<title>{{.Channel.Name}}</title>
					<link>{{.Channel.URL.Text}}</link>
					<image>
						<url>{{.ServerBaseURL}}/{{.Channel.Id}}.png</url>
						<title>{{.Channel.Name}}</title>
						<link>{{.Channel.URL.Text}}</link>
					</image>
					<language>en-us</language>
					<copyright>{{.Channel.Name}}</copyright>
					<lastBuildDate>{{.Channel.Published}}</lastBuildDate>
					<itunes:image href="{{.ServerBaseURL}}/{{.Channel.Id}}.png"/>
					{{range .Channel.Videos}}
					<item>
						<title>{{.Name}}</title>
						<description>{{.Description}}</description>
						<itunes:summary>{{.Description}}</itunes:summary>
						<itunes:image href="{{.ThumbnailURL.Text}}"/>
						<guid>{{$.ServerBaseURL}}/video/{{.Id}}.mp3</guid>
						<link>{{$.ServerBaseURL}}/video/{{.Id}}.mp3</link>
						<enclosure url="{{$.ServerBaseURL}}/video/{{.Id}}.mp3" type="audio/mpeg"/>
						<pubDate>{{.Published}}</pubDate>
					</item>
					{{end}}
				</channel>
			</rss>`)
	if err != nil {
		server.Logger.Fatal(err)
	}
	server.GET("/channel/:channelId", buildChannelRoute(server.Logger, feedTemplate, staticDirectory))
	server.GET("/video/:videoId", buildVideoRoute(server.Logger, staticDirectory, tempDirectory))
	server.Logger.Fatal(server.StartServer(&http.Server{Addr: ":1323", ReadTimeout: 5 * time.Minute}))
}

func buildChannelRoute(logger echo.Logger, feedTemplate *template.Template, staticDirectory string) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Get channel data and unmarshal
		channelId := strings.TrimSuffix(c.Param("channelId"), ".xml")
		resp, err := http.Get(fmt.Sprintf("https://www.youtube.com/feeds/videos.xml?channel_id=%s", channelId))
		defer resp.Body.Close()
		if err != nil {
			logger.Error(err)
			return c.String(http.StatusServiceUnavailable, "Could not retrieve channel data")
		}
		// Ignore, fallthrough error handling to the parse
		xmlBytes, _ := ioutil.ReadAll(resp.Body)
		var channelData Channel
		err = xml.Unmarshal(xmlBytes, &channelData)
		if err != nil {
			logger.Error(err)
			return c.String(http.StatusInternalServerError, "Could not parse channel data")
		}
		iconFileName := fmt.Sprintf("%s.png", channelData.Id)
		// Create static assets if they don't exist
		if _, err := os.Stat(path.Join(staticDirectory, iconFileName)); os.IsNotExist(err) {
			icon, err := identicon.New(channelData.Id, &identicon.Options{ImageSize: 128})
			if err != nil {
				logger.Error(err)
				return c.String(http.StatusInternalServerError, "Could not generate channel icon")
			}
			handle, err := os.Create(path.Join(staticDirectory, iconFileName))
			if err != nil {
				logger.Error(err)
				return c.String(http.StatusInternalServerError, "Could not create channel icon file")
			}
			defer handle.Close()
			err = png.Encode(handle, icon)
			if err != nil {
				logger.Error(err)
				return c.String(http.StatusInternalServerError, "Could not encode channel icon png")
			}
		}
		var response bytes.Buffer
		baseUrl := c.Request().Host
		feedTemplate.Execute(&response, FeedTemplateContext{Channel: channelData, ServerBaseURL: "http://" + baseUrl})
		return c.XMLBlob(http.StatusOK, response.Bytes())
	}
}

func buildVideoRoute(logger echo.Logger, staticDirectory string, temporaryDirectory string) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Get channel data and unmarshal
		videoId := strings.TrimSuffix(c.Param("videoId"), ".mp3")
		videoFileName := fmt.Sprintf("%s.mp4", videoId)
		audioFileName := fmt.Sprintf("%s.mp3", videoId)
		// Create static assets if they don't exist
		if _, err := os.Stat(path.Join(staticDirectory, audioFileName)); os.IsNotExist(err) {
			video, err := ytdl.GetVideoInfo(fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoId))
			if err != nil {
				logger.Error(err)
				return c.String(http.StatusServiceUnavailable, "Could not retrieve video data")
			}
			touchHandle, _ := os.Create(path.Join(staticDirectory, audioFileName))
			touchHandle.Close()
			videoHandle, err := os.Create(path.Join(staticDirectory, videoFileName))
			if err != nil {
				logger.Error(err)
				return c.String(http.StatusInternalServerError, "Could not open video file")
			}
			err = video.Download(video.Formats.Best(ytdl.FormatAudioBitrateKey)[0], videoHandle)
			if err != nil {
				logger.Error(err)
				return c.String(http.StatusServiceUnavailable, "Could not download video")
			}
			videoHandle.Close()
			err = exec.Command("ffmpeg", "-i",  videoFileName, audioFileName, "-y").Run()
			if err != nil {
				print(err)
			}
		}
		baseUrl := c.Request().Host
		return c.Redirect(http.StatusTemporaryRedirect, "http://" + baseUrl + "/" + audioFileName)
	}
}
