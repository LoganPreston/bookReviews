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

		if !strings.Contains(title, searchTitle) &&
			!strings.Contains(searchTitle, title) {
			//fmt.Printf("\n\t sorted out: %s, expected %s\n", title, searchTitle)
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

	inFile, err := os.Open("books.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer inFile.Close()

	outFile, err := os.Create("./booksOut.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer outFile.Close()

	writer := bufio.NewWriter(outFile)
	line := "Title|Author|ISBN|Review|Review Count|Book Count\n"
	if _, err := writer.WriteString(line); err != nil {
		fmt.Println(err.Error())
	}

	scanner := bufio.NewScanner(inFile)
	for scanner.Scan() {
		items := itemsInfo{}

		book := scanner.Text()
		bookInfo := strings.Split(book, "|")
		fmt.Printf("Checking %s by %s\n", bookInfo[0], bookInfo[1])

		url := getUrl(bookInfo[0], bookInfo[1])
		responseBytes, _ := getUrlInfo(url)

		if err := json.Unmarshal(responseBytes, &items); err != nil {
			fmt.Println(err.Error())
		}

		avgRating, numReviews, bookCount := getBookRating(items.Items, bookInfo[0])

		//get author, title. Default to input, else first with isbn
		title, author, isbn := bookInfo[0], []string{bookInfo[1]}, ""
		for i := 0; i < len(items.Items) && isbn == ""; i++ {
			title = items.Items[i].VolumeInfo.Title
			author = items.Items[i].VolumeInfo.Authors
			isbn = getIsbn(items.Items[i].VolumeInfo.Ids)
		}

		line := fmt.Sprintf("%s|%v|%s|%.2f|%d|%d\n",
			title, author, isbn, avgRating, numReviews, bookCount)
		_, err := writer.WriteString(line)
		if err != nil {
			fmt.Println(err.Error())
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	writer.Flush()
}
