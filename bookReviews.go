package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"bookReviews/config"
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

func readBooks() []string {
	books := make([]string, 0, 100)

	file, err := os.Open("books.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		books = append(books, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	return books
}

func getUrl(book, author string) string {
	//get isbn from librarything?
	//libraryThingUrlBase := "http://librarything.com/api/thingTitle/"

	url := "https://www.googleapis.com/books/v1/volumes?q="
	book = strings.Replace(book, " ", "%20", -1)
	//book = "\"" + book + "\""
	url += "intitle:" + book

	if len(author) > 0 {
		author = strings.Replace(author, " ", "%20", -1)
		//author = "\"" + author + "\""
		url += "+inauthor:" + author
	}

	url += "&langRestrict=\"en\""
	url += "&key=" + config.Key
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
	//fmt.Printf("%v\n", response)
	//read the response
	if responseData, err = ioutil.ReadAll(response.Body); err != nil {
		return []byte{}, err
	}
	//fmt.Printf("%v\n",responseData)
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

func getBookInfo(items []item) (float64, int, string, []string, string) {
	var (
		avgRating   float64
		numReviews  int
		title, isbn string
		author      []string
	)

	for _, item := range items {
		//if we know the language, make sure it's english
		lang := item.VolumeInfo.Language
		if len(lang) > 0 && lang != "en" {
			continue
		}

		itemRating := item.VolumeInfo.AverageRating
		itemNumReviews := item.VolumeInfo.RatingsCount
		avgRating = getWeightedAvg(avgRating, itemRating, numReviews,
			itemNumReviews)
		numReviews += item.VolumeInfo.RatingsCount

		/*
			title = item.VolumeInfo.Title
			author = item.VolumeInfo.Authors
			fmt.Printf("%s, %v\n", title, author)
		*/
		if len(isbn) == 0 {
			title = item.VolumeInfo.Title
			author = item.VolumeInfo.Authors
			isbn = getIsbn(item.VolumeInfo.Ids)
		}
	}
	return avgRating, numReviews, title, author, isbn
}

func main() {
	var (
		items           itemsInfo
		url             string
		bookInfo, books []string
		responseBytes   []byte
	)

	if err := config.ReadConfig(); err != nil {
		fmt.Println(err.Error())
		return
	}

	books = readBooks()

	for _, book := range books {
		items = itemsInfo{}
		bookInfo = strings.Split(book, "|")
		url = getUrl(bookInfo[0], bookInfo[1])
		responseBytes, _ = getUrlInfo(url)

		//fmt.Printf("%v\n", responseBytes)
		if err := json.Unmarshal(responseBytes, &items); err != nil {
			fmt.Println(err.Error())
		}
		//fmt.Printf("%v\n", items)
		avgRating, numReviews, title, author, isbn := getBookInfo(items.Items)
		fmt.Printf("Title: %s\n\tAuthor: %s\n\t"+
			"ISBN: %s\n\t Review: %.2f\n\t"+
			"Review Count: %d\n\n", title, author, isbn,
			avgRating, numReviews)
	}

	//use isbn to get reviews from google books

}
