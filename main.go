//
// Copyright (C) 2022 YumeMichi
//
// SPDX-License-Identifier: Apache-2.0
//

package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/YumeMichi/mojim-lrc-downloader/config"
	"github.com/YumeMichi/mojim-lrc-downloader/utils"
	"github.com/anaskhan96/soup"
	"golang.org/x/net/proxy"
)

var (
	mSearchUrl = "https://mojim.com/song_name.html?t3"
	mSongUrl   = "https://mojim.com/twysong_id.htm"
	mLrcUrl    = "https://mojim.com/twthxsong_idx1.htm"
	client     http.Client
)

func init() {
	if !utils.PathExists(config.Conf.Lrc.LoadFileName) {
		_, err := os.Create(config.Conf.Lrc.LoadFileName)
		if err != nil {
			panic(err)
		}
	}

	if !utils.PathExists(config.Conf.Lrc.SaveFileName) {
		_, err := os.Create(config.Conf.Lrc.SaveFileName)
		if err != nil {
			panic(err)
		}
	}

	c, err := NewDialer()
	if err != nil {
		panic(err)
	}
	client = c
}

func NewDialer() (client http.Client, err error) {
	if config.Conf.Proxy.Enabled {
		if strings.ToUpper(config.Conf.Proxy.Protocol) != "SOCKS5" {
			fmt.Println("Only support SOCKS5 without authentication for now...")
			return
		}
		var dialer proxy.Dialer
		dialer, err = proxy.SOCKS5("tcp", config.Conf.Proxy.Ip+":"+config.Conf.Proxy.Port, nil, proxy.Direct)
		if err != nil {
			return
		}
		transport := &http.Transport{Dial: dialer.Dial}
		client.Transport = transport
	}
	return
}

func getSongIdList(songInfo string) (songIdList map[int]string) {
	// fmt.Println(songInfo)
	info := strings.Split(songInfo, ",")
	// fmt.Println(info)
	// fmt.Println(len(info))
	if len(info) != 3 {
		fmt.Println("Error line: ", songInfo)
		return
	}
	songUrl := strings.ReplaceAll(mSearchUrl, "song_name", url.QueryEscape(info[0]))
	req, err := http.NewRequest("GET", songUrl, nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer res.Body.Close()

	idx := 1
	list := make(map[int]string)
	doc := soup.HTMLParse(string(body))
	spans := doc.FindAll("span", "class", "mxsh_ss4")
	for _, span := range spans {
		as := span.FindAll("a")
		if len(as) == 0 {
			continue
		}
		for _, a := range as {
			title := a.Attrs()["title"]
			// fmt.Println(title)
			if strings.Contains(title, info[1]) {
				if info[2] != "" {
					id, err := strconv.Atoi(info[2])
					if err != nil {
						return
					}
					if idx == id {
						// fmt.Println(a.Attrs()["title"])
						list[id] = strings.ReplaceAll(strings.ReplaceAll(a.Attrs()["href"], "/twy", ""), ".htm", "")
					}
				} else {
					// fmt.Println(a.Attrs()["title"])
					list[idx] = strings.ReplaceAll(strings.ReplaceAll(a.Attrs()["href"], "/twy", ""), ".htm", "")
				}
			}
			idx++
		}
	}

	// fmt.Println(list)

	return list
}

func getLrcById(songId string) (lrc string) {
	lrcUrl := strings.ReplaceAll(mLrcUrl, "song_id", songId)
	req, err := http.NewRequest("GET", lrcUrl, nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer res.Body.Close()

	doc := soup.HTMLParse(string(body))
	script := doc.Find("script").HTML()
	// fmt.Println(script)

	regex := regexp.MustCompile("var swfmm = \"(.*?)\";")
	match := regex.FindSubmatch([]byte(script))
	if len(match) > 0 {
		content, err := url.QueryUnescape(strings.ReplaceAll(string(match[len(match)-1]), "_", "%"))
		if err != nil {
			fmt.Println(err)
			return
		}
		return content
	}
	return
}

func main() {
	inFile, err := os.OpenFile(config.Conf.Lrc.LoadFileName, os.O_RDONLY, 0777)
	if err != nil {
		panic(err)
	}
	defer inFile.Close()

	outFile, err := os.OpenFile(config.Conf.Lrc.SaveFileName, os.O_WRONLY|os.O_APPEND, 0777)
	if err != nil {
		panic(err)
	}
	defer outFile.Close()

	reader := bufio.NewReader(inFile)
	writer := bufio.NewWriter(outFile)
	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			panic(err)
		}

		if err == io.EOF {
			break
		}

		line = strings.TrimSpace(line)
		fmt.Printf("Pulling LRC for %s...\n", line)
		songIdList := getSongIdList(line)
		if len(songIdList) == 0 {
			fmt.Printf("NO LRC for %s\n", line)
		} else if len(songIdList) == 1 {
			for _, uid := range songIdList {
				lrc := getLrcById(uid)
				if lrc != "" {
					writer.WriteString(lrc)
					writer.Flush()
				}
			}
		} else {
			fmt.Printf("There are multiple LRCs for %s, please specify the ID of the one you need in lrc-in.txt.\n", line)
			fmt.Println("ID\tLink")
			for id, uid := range songIdList {
				songId := strconv.Itoa(id)
				fmt.Println(songId + "\t" + strings.ReplaceAll(mSongUrl, "song_id", uid))
			}
		}
		// fmt.Println(songIdList)
	}

	reader = bufio.NewReader(os.Stdin)
	fmt.Println("Press Enter to exit")
	_, err = reader.ReadString('\n')
	if err != nil {
		panic(err)
	}
}
