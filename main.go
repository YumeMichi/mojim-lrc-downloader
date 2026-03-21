package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/manifoldco/promptui"
)

var (
	ServerAddr = "https://mojigeci.com"
	SecretKey  = "wmn_api_secret_2024_v1"

	DownloadPath = "downloads"
)

type SearchApiResp struct {
	Code int           `json:"code"`
	Msg  string        `json:"msg"`
	Data SearchApiData `json:"data"`
}

type SearchApiData struct {
	Data  []SongData `json:"data"`
	Cache int        `json:"cache"`
}

type SongData struct {
	ID     int      `json:"id"`
	Name   string   `json:"name"`
	Artist []string `json:"artist"`
	Cover  string   `json:"cover"`
	Album  string   `json:"album"`
}

type LrcApiResp struct {
	Code int        `json:"code"`
	Msg  string     `json:"msg"`
	Data LrcApiData `json:"data"`
}

type LrcApiData struct {
	Code         string  `json:"code"`
	Data         LrcData `json:"data"`
	Cache        int     `json:"cache"`
	AudioExpired int     `json:"audio_expired"`
	Source       string  `json:"source"`
}

type LrcData struct {
	Name     string `json:"name"`
	Artist   string `json:"artist"`
	Cover    string `json:"cover"`
	AudioURL string `json:"audio_url"`
	Lyrics   string `json:"lyrics"`
	Tlyric   string `json:"tlyric"`
}

func sign(name, ts string) string {
	hash := sha256.Sum256([]byte(name + ts + SecretKey))
	return hex.EncodeToString(hash[:])
}

func request(api string) ([]byte, error) {
	req, err := http.NewRequest("GET", api, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/145.0.0.0 Safari/537.36")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http error: %d", resp.StatusCode)
	}

	res, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func main() {
	kw := strings.Join(os.Args[1:], " ")
	ts := strconv.Itoa(int(time.Now().UnixMilli()))
	sig := sign(kw, ts)

	listParams := url.Values{
		"keyword":   {kw},
		"timestamp": {ts},
		"signature": {sig},
		"page":      {"1"},
		"pageSize":  {"12"},
	}
	listQuery := listParams.Encode()
	listApi := ServerAddr + "/api/search_lists?" + listQuery

	res, err := request(listApi)
	if err != nil {
		panic(err)
	}

	var listData SearchApiResp
	if err := json.Unmarshal(res, &listData); err != nil {
		panic(err)
	}

	if listData.Code != 200 {
		panic(listData.Msg)
	}

	prompt := promptui.Select{
		Label: "选择需要下载的歌词",
		Items: listData.Data.Data,
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . }}",
			Active:   "👉 {{ .Name }} ({{ .Album }}) - {{ index .Artist 0 }}",
			Inactive: "  {{ .Name }} ({{ .Album }}) - {{ index .Artist 0 }}",
			Selected: "✔ {{ .Name }} ({{ .Album }}) - {{ index .Artist 0 }}",
			Help:     "使用方向键选择 (↑ ↓ ← →)，回车确认。",
		},
	}

	selected, _, err := prompt.Run()
	if err != nil {
		return
	}

	song := listData.Data.Data[selected]

	ts = strconv.Itoa(int(time.Now().UnixMilli()))
	sig = sign(kw, ts)

	lrcParams := url.Values{
		"id":          {strconv.Itoa(song.ID)},
		"song_name":   {song.Name},
		"song_artist": {strings.Join(song.Artist, "、")},
		"song_cover":  {song.Cover},
		"keyword":     {kw},
		"timestamp":   {ts},
		"signature":   {sig},
	}
	lrcQuery := lrcParams.Encode()
	lrcApi := ServerAddr + "/api/get_lyrics_by_id?" + lrcQuery

	res, err = request(lrcApi)
	if err != nil {
		panic(err)
	}

	var lrc LrcApiResp
	if err := json.Unmarshal(res, &lrc); err != nil {
		panic(err)
	}

	if lrc.Code != 200 {
		panic(lrc.Msg)
	}

	fileName := fmt.Sprintf("%d %s - %s", song.ID, lrc.Data.Data.Artist, lrc.Data.Data.Name)
	jpLrcFile := filepath.Join(DownloadPath, fileName+".lrc")
	chtLrcFile := filepath.Join(DownloadPath, fileName+".cht.lrc")

	if err := os.MkdirAll(DownloadPath, 0755); err != nil {
		panic(err)
	}
	if err := os.WriteFile(jpLrcFile, []byte(lrc.Data.Data.Lyrics), 0644); err != nil {
		panic(err)
	}
	if err := os.WriteFile(chtLrcFile, []byte(lrc.Data.Data.Tlyric), 0644); err != nil {
		panic(err)
	}

	fmt.Println("下载完成！")
}
