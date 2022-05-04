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

func getBookRating(items []item, searchTitle string) (float64, int, int) {
	var (
		avgRating  float64
		numReviews int
		bookCount  int
	)

	searchTitle = strings.ToLower(searchTitle)
	searchTitle = strings.Replace(searchTitle, ",", "", -1)
	searchTitle = strings.Replace(searchTitle, "-", " ", -1)

	for _, item := range items {
		//if we know the language, make sure it's english
		lang := item.VolumeInfo.Language
		if len(lang) > 0 && lang != "en" {
			continue
		}

		title := strings.ToLower(item.VolumeInfo.Title)
		title = strings.Replace(title, ",", "", -1)
		title = strings.Replace(title, "-", " ", -1)

		if !strings.Contains(title, searchTitle) &&
			!strings.Contains(searchTitle, title) {
			//fmt.Printf("this title was sorted out: %s because it didn't match %s\n", title, searchTitle)
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

func main() {

	if err := config.ReadConfig(); err != nil {
		fmt.Println(err.Error())
		return
	}

	file, err := os.Create("./booksOut.txt")
	if err != nil {
		fmt.Println(err.Error())
	}
	writer := bufio.NewWriter(file)

	line := "Title|Author|ISBN|Review|Review Count|Book Count\n"
	if _, err := writer.WriteString(line); err != nil {
		fmt.Println(err.Error())
	}

	books := readBooks()

	for _, book := range books {

		items := itemsInfo{}
		bookInfo := strings.Split(book, "|")
		url := getUrl(bookInfo[0], bookInfo[1])
		//fmt.Printf("%s\n", url)
		fmt.Printf("Checking into %s by %s\n", bookInfo[0], bookInfo[1])
		responseBytes, _ := getUrlInfo(url)

		//fmt.Printf("%v\n", responseBytes)
		if err := json.Unmarshal(responseBytes, &items); err != nil {
			fmt.Println(err.Error())
		}
		//fmt.Printf("%v\n", items)
		avgRating, numReviews, bookCount := getBookRating(items.Items, bookInfo[0])

		//get author, title. Default to input, else first with isbn
		title, author, isbn := bookInfo[0], []string{bookInfo[1]}, ""
		for i := 0; i < len(items.Items) && isbn == ""; i++ {
			title = items.Items[i].VolumeInfo.Title
			author = items.Items[i].VolumeInfo.Authors
			isbn = getIsbn(items.Items[i].VolumeInfo.Ids)
			//fmt.Printf("%d, %d, %s\n", len(items.Items), i, isbn)
		}

		line := fmt.Sprintf("%s|%v|%s|%.2f|%d|%d\n", title, author, isbn,
			avgRating, numReviews, bookCount)
		_, err := writer.WriteString(line)
		if err != nil {
			fmt.Println(err.Error())
		}
	}
	writer.Flush()

	//use isbn to get reviews from google books

}
