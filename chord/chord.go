package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"net/http"
	"net/url"

	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type Song struct {
	Title  string `json:"title"`
	Lyrics string `json:"lyrics"`
}

type SongWithChords struct {
	Title  string           `json:"title"`
	Lyrics string           `json:"lyrics"`
	Chords map[int][]string `json:"chords"`
}

func main() {

	fileContent, err := ioutil.ReadFile("../some_songs.json")
	if err != nil {
		fmt.Println("Ошибка при чтении файла:", err)
		return
	}

	var songs []Song
	err = json.Unmarshal(fileContent, &songs)
	if err != nil {
		fmt.Println("Ошибка при разборе JSON:", err)
		return
	}

	var songsWithChords []SongWithChords

	for _, song := range songs {
		searchURL := "https://holychords.pro/search"
		query := url.QueryEscape(song.Title)
		searchURL += "?name=" + query

		req, err := http.NewRequest("GET", searchURL, nil)
		if err != nil {
			fmt.Println("Ошибка при создании запроса:", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "curl/7.64.1")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("Ошибка при выполнении запроса:", err)
			return
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("Ошибка при чтении ответа:", err)
			return
		}
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
		if err != nil {
			fmt.Println("Ошибка при парсинге HTML:", err)
			return
		}

		entries := doc.Find("#entries")
		if entries.Length() > 0 {

			link := entries.Find("a:contains('" + song.Title + "')")
			if link.Length() > 0 {
				href, _ := link.Attr("href")

				songURL := "https://holychords.pro" + href
				songReq, err := http.NewRequest("GET", songURL, nil)
				if err != nil {
					fmt.Println("Ошибка при создании запроса:", err)
					return
				}

				songReq.Header.Set("Content-Type", "application/json")
				songReq.Header.Set("User-Agent", "curl/7.64.1")

				songResp, err := client.Do(songReq)
				if err != nil {
					fmt.Println("Ошибка при выполнении запроса:", err)
					return
				}

				songBody, err := ioutil.ReadAll(songResp.Body)
				if err != nil {
					fmt.Println("Ошибка при чтении ответа:", err)
					return
				}

				songDoc, err := goquery.NewDocumentFromReader(strings.NewReader(string(songBody)))
				if err != nil {
					fmt.Println("Ошибка при парсинге HTML:", err)
					return
				}

				musicText := songDoc.Find("#music_text")
				if musicText.Length() > 0 {

					h, _ := songDoc.Find("#music_text").Html()

					qwe := deleteWords(h)

					qq := deleteBr(qwe)

					chords := extractChords(qq)

					songWithChords := SongWithChords{
						Title:  song.Title,
						Lyrics: song.Lyrics,
						Chords: chords,
					}
					songsWithChords = append(songsWithChords, songWithChords)
				} else {
					fmt.Println("Текст песни не найден")
				}
			} else {
				fmt.Println("Песня не найдена:", song.Title)
			}
		} else {
			fmt.Println("Блок 'entries' не найден на странице")
		}
	}

	songsJSON, err := json.MarshalIndent(songsWithChords, "", "    ")
	if err != nil {
		fmt.Println("Ошибка при преобразовании в JSON:", err)
		return
	}

	err = ioutil.WriteFile("some_songs.json", songsJSON, 0644)
	if err != nil {
		fmt.Println("Ошибка при сохранении в файл:", err)
		return
	}

	fmt.Println("Аккорды сохранены в файл some_songs.json")
}

func deleteWords(str string) string {
	re := regexp.MustCompile(`\d+\s+куплет:|Припев:`)
	str = re.ReplaceAllString(str, "")
	str = strings.TrimPrefix(str, "<br/>")
	str = strings.ReplaceAll(str, "<br/><br/>", "<br/>")

	return str
}

func deleteBr(str string) []string {
	re := regexp.MustCompile(`<br/>(\s*<br/>)+`)
	str = re.ReplaceAllString(str, "<br/>")

	substrings := strings.Split(str, "<br/>")

	for i := range substrings {
		substrings[i] = strings.TrimSpace(substrings[i])
	}

	return substrings
}

func extractChords(lines []string) map[int][]string {
	chords := make(map[int][]string)
	index := 0
	for _, line := range lines {
		isChords := regexp.MustCompile(`^[A-Za-z\s]+$`).MatchString(line)
		if isChords {
			chords[index] = strings.Fields(line)
			index++
		}
	}
	return chords
}
