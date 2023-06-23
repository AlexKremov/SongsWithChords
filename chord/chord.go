package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
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
	filePath := "../some_songs.json"

	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("Ошибка при открытии файла:", err)
		return
	}
	defer file.Close()

	var songs []Song
	err = json.NewDecoder(file).Decode(&songs)
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

		body, err := io.ReadAll(resp.Body)
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

				songBody, err := io.ReadAll(songResp.Body)
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

	outputFilePath := "some_songs.json"
	outputFile, err := os.Create(outputFilePath)
	if err != nil {
		fmt.Println("Ошибка при создании файла:", err)
		return
	}
	defer outputFile.Close()

	encoder := json.NewEncoder(outputFile)
	encoder.SetIndent("", "    ")
	err = encoder.Encode(songsWithChords)
	if err != nil {
		fmt.Println("Ошибка при записи в файл:", err)
		return
	}

	fmt.Println("Аккорды сохранены в файл", outputFilePath)
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
		line = strings.TrimSpace(line)
		if regexp.MustCompile(`^[A-Za-z0-9\s]+$`).MatchString(line) {
			chords[index+1] = strings.Fields(line)
			index++
		}
	}

	return chords
}
