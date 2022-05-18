package main

import (
	"bookReviews/config"

	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type itemsInfo struct {
	Items []item `json:"items"`
}

type item struct {
	VolumeInfo volumeInfo `json:"volumeInfo"`
}

type volumeInfo struct {
	Title         string       `json:"title"`
	Authors       []string     `json:"authors"`
	Ids           []identifier `json:"industryIdentifiers"`
	AverageRating float64      `json:"averageRating"`
	RatingsCount  int          `json:"ratingsCount"`
	PageCount     int          `json:"pageCount"`
	Language      string       `json:"language"`
}

type identifier struct {
	TypeName   string `json:"type"`
	Identifier string `json:"identifier"`
}

func getUrl(book, author string) string {

	url := "https://www.googleapis.com/books/v1/volumes?q="
	book = strings.Replace(book, " ", "%20", -1)
	url += "intitle:" + book

	if len(author) > 0 {
		author = strings.Replace(author, " ", "%20", -1)
		url += "+inauthor:" + author
	}

	url += "&langRestrict=\"en\""
	//url += "&key=" + config.Key
	//fmt.Printf("%s\n", url)
	return url
}

func getUrlInfo(url string) ([]byte, error) {
	var (
		response     *http.Response
		responseData []byte
		err          error
	)
	//get the initial info
	if response, err = http.Get(url); err != nil {
		return []byte{}, err
	}

	if response.StatusCode != 200 {
		fmt.Printf("\tStatus Code %d received, URL: %s\n",
			response.StatusCode, url)
	}

	//read the response
	if responseData, err = ioutil.ReadAll(response.Body); err != nil {
		return []byte{}, err
	}

	return responseData, nil
}

func getIsbn(Ids []identifier) string {
	var isbn string
	for _, id := range Ids {
		if id.TypeName == "ISBN_13" {
			isbn = id.Identifier
			break
		}
	}

	return isbn
}

func getWeightedAvg(valOne, valTwo float64, countOne, countTwo int) (rating float64) {
	if countOne+countTwo == 0 {
		return
	}
	rating = valOne*float64(countOne) + valTwo*float64(countTwo)
	rating /= float64(countOne + countTwo)
	return
}

func getBookRating(items []item, searchTitle string) (float64, int, int) {
	var (
		avgRating  float64
		numReviews int
		bookCount  int
	)

	searchTitle = strings.ToLower(searchTitle)
	searchTitle = strings.Replace(searchTitle, ",", "", -1)
	searchTitle = strings.Replace(searchTitle, "'", "", -1)
	searchTitle = strings.Replace(searchTitle, "-", " ", -1)

	for _, item := range items {
		//if we know the language, make sure it's english
		lang := item.VolumeInfo.Language
		if len(lang) > 0 && lang != "en" {
			continue
		}

		title := strings.ToLower(item.VolumeInfo.Title)
		title = strings.Replace(title, ",", "", -1)
		title = strings.Replace(title, "'", "", -1)
		title = strings.Replace(title, "-", " ", -1)

		lenSearchTitle := len(searchTitle)
		lenTitle := len(title)

		if lenTitle < lenSearchTitle &&
			!strings.Contains(title, searchTitle) {
			continue
		}
		if lenSearchTitle < lenTitle &&
			!strings.Contains(searchTitle, title) {
			continue
		}

		itemRating := item.VolumeInfo.AverageRating
		itemNumReviews := item.VolumeInfo.RatingsCount
		avgRating = getWeightedAvg(avgRating, itemRating, numReviews,
			itemNumReviews)
		numReviews += item.VolumeInfo.RatingsCount
		bookCount++

		//title := item.VolumeInfo.Title
		//author := item.VolumeInfo.Authors
		//fmt.Printf("%s, %v\n", title, author)

	}
	return avgRating, numReviews, bookCount
}

func processBook(book string, ch chan string, wg *sync.WaitGroup) {

	defer wg.Done()

	items := itemsInfo{}
	bookInfo := strings.Split(book, "|")
	fmt.Printf("Checking %s by %s\n", bookInfo[0], bookInfo[1])

	url := getUrl(bookInfo[0], bookInfo[1])
	responseBytes, _ := getUrlInfo(url)
	if err := json.Unmarshal(responseBytes, &items); err != nil {
		fmt.Println(err.Error())
	}

	avgRating, numReviews, bookCount :=
		getBookRating(items.Items, bookInfo[0])

	//get author, title. Default to input, else first with isbn
	title, author, isbn := bookInfo[0], []string{bookInfo[1]}, ""
	for i := 0; i < len(items.Items) && isbn == ""; i++ {
		title = items.Items[i].VolumeInfo.Title
		author = items.Items[i].VolumeInfo.Authors
		isbn = getIsbn(items.Items[i].VolumeInfo.Ids)
	}

	ch <- fmt.Sprintf("%s|%v|%s|%.2f|%d|%d\n",
		title, author, isbn, avgRating, numReviews, bookCount)
	return
}

func writeReviews(ch chan string, wg *sync.WaitGroup) {

	defer wg.Done()

	outFile, err := os.Create("./booksOut.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer outFile.Close()

	writer := bufio.NewWriter(outFile)
	for line := range ch {
		if _, err := writer.WriteString(line); err != nil {
			fmt.Println(err.Error())
		}
	}
	writer.Flush()
	return
}

func main() {

	var writeWg, readWg sync.WaitGroup

	if err := config.ReadConfig(); err != nil {
		fmt.Println(err.Error())
		return
	}

	inFile, err := os.Open("books.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer inFile.Close()

	ch := make(chan string)
	readWg.Add(1)
	go writeReviews(ch, &readWg)
	ch <- "Title|Author|ISBN|Review|Review Count|Book Count\n"

	scanner := bufio.NewScanner(inFile)
	for scanner.Scan() {
		book := scanner.Text()
		writeWg.Add(1)
		go processBook(book, ch, &writeWg)
		time.Sleep(100 * time.Millisecond) //try to avoid 429s
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	//wait for the writers to the channel 
	writeWg.Wait()

	//wait for the reader to finish up and output cleanly
	close(ch)
	readWg.Wait()
}
