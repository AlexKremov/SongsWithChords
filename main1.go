package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

type Song struct {
	Title    string `json:"title"`
	Category string `json:"category"`
	Number   int    `json:"number"`
	Lyrics   string `json:"lyrics"`
}

type Songs []Song

func (s Songs) Len() int {
	return len(s)
}

func (s Songs) Less(i, j int) bool {
	return s[i].Number < s[j].Number
}

func (s Songs) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func getSong(url string, wg *sync.WaitGroup, songs *Songs, sem chan struct{}) {
	defer wg.Done()
	defer func() { <-sem }()

	fmt.Println(url)
	doc, err := goquery.NewDocument(url)
	if err != nil {
		fmt.Println("Error getting song:", err)
		return
	}

	title := doc.Find("h1").Text()
	lyricsElement := doc.Find("#music_text")

	// Удаление ненужных тегов
	lyricsElement.Find("sup, br").Each(func(i int, s *goquery.Selection) {
		s.Remove()
	})

	// Удаление строк с "Куплет" и "Припев"
	cleanedLyrics := strings.TrimSpace(lyricsElement.Text())
	lines := strings.Split(cleanedLyrics, "\n")
	cleanedLines := []string{}

	for _, line := range lines {
		if !strings.Contains(line, "Куплет") && !strings.Contains(line, "Припев") {
			cleanedLines = append(cleanedLines, line)
		}
	}

	cleanedLyrics = strings.Join(cleanedLines, "\\n")

	collectionStr := doc.Find("#song-info tr").First().Find("td").Text()
	collectionNo := 0
	if matched := regexp.MustCompile(`(\d+)`).FindStringSubmatch(collectionStr); len(matched) > 0 {
		collectionNo, _ = strconv.Atoi(matched[1])
	}

	categoryStr := doc.Find("#song-info tr").Last().Find("td").Text()

	// Замена новых строк на \n
	cleanedLyrics = strings.Replace(cleanedLyrics, "\n", "\\n", -1)

	song := Song{
		Title:    strings.TrimSpace(title),
		Lyrics:   strings.TrimSpace(cleanedLyrics),
		Number:   collectionNo,
		Category: categoryStr,
	}

	// Добавление песни в общий массив
	*songs = append(*songs, song)
}

func main() {
	var wg sync.WaitGroup
	songs := Songs{}
	pageUrl := "https://hvalite.com/pesni"
	urur := ""
	sem := make(chan struct{}, 5)

	for {
		doc, err := goquery.NewDocument(pageUrl)
		if err != nil {
			panic(err)
		}

		doc.Find(".list-view h6 a").Each(func(i int, s *goquery.Selection) {
			songUrl, exists := s.Attr("href")
			urur = songUrl

			if exists {
				wg.Add(1)
				sem <- struct{}{}
				// song, err := getSong("https://hvalite.com" + songUrl)
				go getSong("https://hvalite.com"+songUrl, &wg, &songs, sem)
				// songs = append(songs, song)
			}
		})

		// Проверяем наличие ссылки на следующую страницу
		nextPageLink := doc.Find("li.next a").First()

		if nextPageLink.Length() == 0 || strings.Contains(urur, "-30") {
			// Если ссылки на следующую страницу нет, выходим из цикла
			break
		} else {
			// Иначе, получаем URL следующей страницы и продолжаем парсинг
			pageUrl, _ = nextPageLink.Attr("href")
			pageUrl = "https://hvalite.com" + pageUrl
		}
	}
	wg.Wait()
	sort.Sort(songs)
	file, _ := json.MarshalIndent(songs, "", " ")
	_ = os.WriteFile("some_songs.json", file, 0644)
}
