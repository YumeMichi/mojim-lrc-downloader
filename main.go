package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/manifoldco/promptui"
	"github.com/spf13/pflag"
)

var (
	serverAddr = "https://mojigeci.com"
	secretKey  = "wmn_api_secret_2024_v1"

	inputFlag  = pflag.StringP("input", "i", "input.txt", "读取文本文件")
	outputFlag = pflag.StringP("output", "o", "downloads", "保存目录")
	stripFlag  = pflag.BoolP("strip", "s", false, "去除歌词中的时间戳")
	assFlag    = pflag.BoolP("ass", "a", false, "生成ASS字幕文件")

	blackList = []string{
		"作詞", "作曲", "編曲",
	}

	httpClient = &http.Client{
		Timeout: 15 * time.Second,
	}

	lrcLineRegexp = regexp.MustCompile(`\[(\d{2}):(\d{2})\.(\d{2,3})\](.*)`)

	fileNameSanitizer = strings.NewReplacer(
		"/", "／",
		"\\", "＼",
		":", "：",
		"*", "＊",
		"?", "？",
		`"`, "＂",
		"<", "＜",
		">", "＞",
		"|", "｜",
		"\n", " ",
		"\r", " ",
	)
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

func (s SongData) ArtistDisplay() string {
	if len(s.Artist) == 0 {
		return "未知歌手"
	}
	return s.Artist[0]
}

func (s SongData) AlbumDisplay() string {
	if strings.TrimSpace(s.Album) == "" {
		return "未知专辑"
	}
	return s.Album
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

type LrcLine struct {
	Text  string
	Start int
	End   int
}

func generateApiSign(name, ts string) string {
	hash := sha256.Sum256([]byte(name + ts + secretKey))
	return hex.EncodeToString(hash[:])
}

func mojimRequest(ctx context.Context, api string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, api, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/145.0.0.0 Safari/537.36")

	resp, err := httpClient.Do(req)
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

func requestJSON[T any](ctx context.Context, api string, out *T) error {
	res, err := mojimRequest(ctx, api)
	if err != nil {
		return err
	}
	return json.Unmarshal(res, out)
}

func inBlackList(lrc string) bool {
	for _, s := range blackList {
		if strings.Contains(lrc, s) {
			return true
		}
	}
	return false
}

func splitLrc(lrc string) []string {
	lines := strings.Split(lrc, "\n")
	res := make([]string, 0, len(lines))
	for _, line := range lines {
		if !inBlackList(line) {
			res = append(res, line)
		}
	}
	return res
}

func stripLrc(lrc string) string {
	lines := splitLrc(lrc)
	plainLines := make([]string, 0, len(lines))
	for _, line := range lines {
		_, text, ok := strings.Cut(line, "]")
		if !ok || text == "" {
			continue
		}
		plainLines = append(plainLines, text)
	}
	return strings.Join(plainLines, "\n")
}

func formatTime(ms int) string {
	m := ms / 60000
	s := (ms % 60000) / 1000
	msRemain := ms % 1000
	return fmt.Sprintf("%02d:%02d.%03d", m, s, msRemain)
}

func parseLrcTime(min, sec, ms string) int {
	m, _ := strconv.Atoi(min)
	s, _ := strconv.Atoi(sec)
	milli, _ := strconv.Atoi(ms)

	if len(ms) == 2 {
		milli *= 10
	}

	return m*60*1000 + s*1000 + milli
}

func parseLrc(lrc string) []LrcLine {
	split := splitLrc(lrc)
	res := make([]LrcLine, 0, len(split))
	for _, line := range split {
		match := lrcLineRegexp.FindStringSubmatch(line)
		if match == nil {
			continue
		}

		t := parseLrcTime(match[1], match[2], match[3])
		start := max(t-200, 0) // 作为字幕而言，提前 200ms 显示更符合人眼习惯
		text := strings.TrimSpace(match[4])
		if text == "" {
			continue
		}

		if len(res) > 0 {
			res[len(res)-1].End = start
		}

		res = append(res, LrcLine{
			Text:  text,
			Start: start,
		})
	}

	return res
}

func CreateASS(lrc string) string {
	header := `[Script Info]
ScriptType: v4.00+

[V4+ Styles]
Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding
Style: Default,Arial,48,&H00FFFFFF,&H000000FF,&H00000000,&H00000000,0,0,0,0,100,100,0,0,1,2,2,2,10,10,10,1

[Events]
Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text
`
	const styleName = "Default"

	var builder strings.Builder
	builder.WriteString(header)

	for _, line := range parseLrc(lrc) {
		fmt.Fprintf(&builder, "Dialogue: 0,%s,%s,%s,,0,0,0,,%s\n",
			formatTime(line.Start), formatTime(line.End), styleName, line.Text)
	}

	return builder.String()
}

func nowTimestamp() string {
	return strconv.FormatInt(time.Now().UnixMilli(), 10)
}

func sanitizeFileName(name string) string {
	res := strings.TrimSpace(fileNameSanitizer.Replace(name))
	if res == "" {
		return "unknown"
	}
	return res
}

func readKeywords() ([]string, error) {
	if pflag.Lookup("input").Changed {
		data, err := os.ReadFile(*inputFlag)
		if err != nil {
			return nil, err
		}

		lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
		keywords := make([]string, 0, len(lines))
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			keywords = append(keywords, line)
		}
		return keywords, nil
	}

	kw := strings.TrimSpace(strings.Join(pflag.Args(), " "))
	if kw == "" {
		return nil, errors.New("未提供关键词，请传入参数或使用 -i/--input 文件")
	}
	return []string{kw}, nil
}

func searchSongs(ctx context.Context, keyword string) ([]SongData, error) {
	ts := nowTimestamp()
	sig := generateApiSign(keyword, ts)

	listParams := url.Values{
		"keyword":   {keyword},
		"timestamp": {ts},
		"signature": {sig},
		"page":      {"1"},
		"pageSize":  {"12"},
	}
	listAPI := serverAddr + "/api/search_lists?" + listParams.Encode()

	var listData SearchApiResp
	if err := requestJSON(ctx, listAPI, &listData); err != nil {
		return nil, err
	}
	if listData.Code != 200 {
		return nil, errors.New(listData.Msg)
	}
	return listData.Data.Data, nil
}

func fetchLyrics(ctx context.Context, keyword string, song SongData) (LrcData, error) {
	ts := nowTimestamp()
	sig := generateApiSign(keyword, ts)

	lrcParams := url.Values{
		"id":          {strconv.Itoa(song.ID)},
		"song_name":   {song.Name},
		"song_artist": {strings.Join(song.Artist, "、")},
		"song_cover":  {song.Cover},
		"keyword":     {keyword},
		"timestamp":   {ts},
		"signature":   {sig},
	}
	lrcAPI := serverAddr + "/api/get_lyrics_by_id?" + lrcParams.Encode()

	var lrc LrcApiResp
	if err := requestJSON(ctx, lrcAPI, &lrc); err != nil {
		return LrcData{}, err
	}
	if lrc.Code != 200 {
		return LrcData{}, errors.New(lrc.Msg)
	}
	return lrc.Data.Data, nil
}

func writeLyricsFiles(baseName string, lyrics LrcData) error {
	if *assFlag {
		jpAssFile := filepath.Join(*outputFlag, baseName+".ass")
		if err := os.WriteFile(jpAssFile, []byte(CreateASS(lyrics.Lyrics)), 0o644); err != nil {
			return err
		}

		chtAssFile := filepath.Join(*outputFlag, baseName+".cht.ass")
		if err := os.WriteFile(chtAssFile, []byte(CreateASS(lyrics.Tlyric)), 0o644); err != nil {
			return err
		}
	}

	jpLyrics := lyrics.Lyrics
	chtLyrics := lyrics.Tlyric
	if *stripFlag {
		jpLyrics = stripLrc(jpLyrics)
		chtLyrics = stripLrc(chtLyrics)
	}

	jpLrcFile := filepath.Join(*outputFlag, baseName+".lrc")
	if err := os.WriteFile(jpLrcFile, []byte(jpLyrics), 0o644); err != nil {
		return err
	}

	chtLrcFile := filepath.Join(*outputFlag, baseName+".cht.lrc")
	if err := os.WriteFile(chtLrcFile, []byte(chtLyrics), 0o644); err != nil {
		return err
	}

	return nil
}

func processKeyword(ctx context.Context, keyword string) error {
	songs, err := searchSongs(ctx, keyword)
	if err != nil {
		return err
	}
	if len(songs) == 0 {
		return errors.New("未找到匹配歌曲")
	}

	prompt := promptui.Select{
		Label: "选择需要下载的歌词",
		Items: songs,
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . }}",
			Active:   "👉 {{ .Name }} ({{ .AlbumDisplay }}) - {{ .ArtistDisplay }}",
			Inactive: "  {{ .Name }} ({{ .AlbumDisplay }}) - {{ .ArtistDisplay }}",
			Selected: "✔ {{ .Name }} ({{ .AlbumDisplay }}) - {{ .ArtistDisplay }}",
			Help:     "使用方向键选择 (↑ ↓ ← →)，回车确认。",
		},
	}

	selected, _, err := prompt.Run()
	if err != nil {
		return err
	}

	lyrics, err := fetchLyrics(ctx, keyword, songs[selected])
	if err != nil {
		return err
	}

	baseName := sanitizeFileName(fmt.Sprintf("%d %s - %s", songs[selected].ID, lyrics.Artist, lyrics.Name))
	return writeLyricsFiles(baseName, lyrics)
}

func main() {
	pflag.Parse()

	keywords, err := readKeywords()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if len(keywords) == 0 {
		fmt.Fprintln(os.Stderr, "未读取到有效关键词")
		os.Exit(1)
	}

	if err := os.MkdirAll(*outputFlag, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	ctx := context.Background()
	for _, kw := range keywords {
		if err := processKeyword(ctx, kw); err != nil {
			fmt.Fprintf(os.Stderr, "%s 下载失败: %v\n", kw, err)
			continue
		}
		fmt.Println(kw + " 下载完成！")
	}
}
