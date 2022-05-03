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

func main() {
	//Just go get it from google, need to format the OAuth/api key to get access to query
	if err := config.ReadConfig(); err != nil {
		fmt.Println(err.Error())
		return
	}
	googleKey := config.Key
	books := readBooks()

	//fmt.Printf("%v\n", books)
	//get isbn from librarything
	//libraryThingUrlBase := "http://librarything.com/api/thingTitle/"
	googleUrlBase := "https://www.googleapis.com/books/v1/volumes?q=intitle:"

	//var isbn bookIsbnInfo
	var (
		items itemsInfo
		avgRating float64
		numReviews int
		title, isbn, url string
	)

	for _, book := range books {
		items = itemsInfo{}
		bookInfo := strings.Split(book, "|")
		book = strings.Replace(bookInfo[0], " ", "%20", -1)
		//book = "\"" + book + "\""
		//TODO need to refine...
		//check length of title maybe?
		url = googleUrlBase + book
		if len(bookInfo) > 1 {
			author := strings.Replace(bookInfo[1], " ", "%20", -1)
			//author = "\"" + author + "\""
			url += "+inauthor:" + author
		}
		url += "&langRestrict=\"en\""
		url += "&key=" + googleKey
		fmt.Printf("%s\n", url)
		responseBytes, _ := getUrlInfo(url)
		//fmt.Printf("%v\n", responseBytes)
		if err := json.Unmarshal(responseBytes, &items); err != nil {
			fmt.Println(err.Error())
		}
		//fmt.Printf("%v\n", items)
		avgRating, numReviews = 0.0, 0
		isbn, title = "", ""
		for _, item := range items.Items {
			if item.VolumeInfo.Language != "en" {
				continue
			}
			itemRating := item.VolumeInfo.AverageRating
			itemNumReviews := item.VolumeInfo.RatingsCount
			if avgRating > 0 {
				avgRating = (avgRating*float64(numReviews) +
					itemRating*float64(itemNumReviews)) /
					float64(numReviews+itemNumReviews)
			} else {
				avgRating = itemRating
			}

			numReviews += item.VolumeInfo.RatingsCount
			title = item.VolumeInfo.Title
			bookAuthor := item.VolumeInfo.Authors
			fmt.Printf("%s, %v\n",title, bookAuthor)
			if len(isbn) == 0 {
				for _, id := range item.VolumeInfo.Ids {
					if id.TypeName == "ISBN_13" {
						isbn = id.Identifier
						break
					}
				}
			}
		}
		fmt.Printf("Title: %s\n\tISBN: %s\n\t Review: %.2f"+
			"\n\t Review Count: %d\n\n", title, isbn,
			avgRating, numReviews)
		/*
			url := libraryThingUrlBase + book
			fmt.Printf("%s\n",url)
			responseBytes, _ := getUrlInfo(url)
			if len(responseBytes) == 0 {
				continue
			}
			fmt.Printf("%v\n",responseBytes)
			if err := xml.Unmarshal(responseBytes, &isbn); err != nil {
				log.Fatal(err)
			}
			fmt.Printf("%#v\n", isbn)
		*/
	}

	//use isbn to get reviews from google books

}
