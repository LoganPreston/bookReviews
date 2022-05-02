package main

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"bookReviews/config"
)

type bookIsbnInfo struct {
	Title string   `xml:"title"`
	Link  string   `xml:"link"`
	Isbn  []string `xml:"isbn"`
}

type itemsInfo struct {
	Items []string `xml:"items"`
}

type reviewStruct struct {
	Name       string
	Isbn       string
	Stars      int
	NumReviews int
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
	fmt.Printf("%v\n",response)
	//read the response
	if responseData, err = ioutil.ReadAll(response.Body); err != nil {
		return []byte{}, err
	}
	fmt.Printf("%v\n",responseData)
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

	fmt.Printf("%v\n",books)
	//get isbn from librarything
	//libraryThingUrlBase := "http://librarything.com/api/thingTitle/"
	googleUrlBase := "https://www.googleapis.com/books/v1/volumes?q="

	//var isbn bookIsbnInfo
	var items itemsInfo
	for _, book := range books {
		url := googleUrlBase + book
		responseBytes, _ := getUrlInfo(url)

		if err := xml.Unmarshal(responseBytes, &items); err != nil{
			fmt.Println(err.Error())
		}
		fmt.Printf("%v\n", items)
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
