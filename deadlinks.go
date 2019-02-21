package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var checkedurl map[string]bool // Creates blacklist map
var nakedurl string            // Original url for the website
var urlfound int               // Total number of url found
var endfile *os.File           // Text file to print in

func checkFatal(e error) { // In case of really bad error
	if e != nil {
		log.Fatal(e)
	}
}

func removeDuplicates(elements []string) []string { // To simply remove duplicate elements from a string slice
	encountered := map[string]bool{} // Use map to record duplicates as we find them.
	result := []string{}

	for v := range elements {
		if encountered[elements[v]] != true { // Record this element as an encountered element.
			encountered[elements[v]] = true // Append to result slice.
			result = append(result, elements[v])
		}
	}
	return result // Return the new slice.
}

func getStatus(url string) (status int, statElapsed time.Duration) { // Get server response (or not) and returns it
	statStart := time.Now()
	resp, err := http.Get(url)
	if err != nil {
		return
	}
	status = resp.StatusCode
	statElapsed = time.Since(statStart)
	defer resp.Body.Close()
	return
}

func fetchurl(urlOrigine string) (urlexternal, urlinternal []string) { // Get 2 slices of all url present on the page sorted by intern & extern

	rhref, err := regexp.Compile(`href="(.+?)"`) // Regexp for all href
	checkFatal(err)
	rsrc, err := regexp.Compile(`src="(.+?)"`) // Regexp for all src
	checkFatal(err)

	resp, err := http.Get(urlOrigine) // Get html code
	checkFatal(err)
	bodyb, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		endfile.WriteString(urlOrigine + " -> Unable to read body\n")
		return
	}
	body := string(bodyb)
	defer resp.Body.Close()

	// Getting & trimming the things we don't need in found url
	hrefList := rhref.FindAllString(body, -1)
	for i, href := range hrefList {
		hrefList[i] = href[6 : len(href)-1]
	}
	srcList := rsrc.FindAllString(body, -1)
	for i, src := range srcList {
		srcList[i] = src[5 : len(src)-1]
	}

	urlall := append(hrefList, srcList...) // Mashing the url lists together to sort them another way

	// Sorting url by external/internal
	for _, url := range urlall {
		if strings.HasPrefix(url, "http") {
			urlexternal = append(urlexternal, url)
		} else if len(url) < 2083 && !strings.HasPrefix(url, "#") { // Max length of an url is 2,083 char & checking anchor is useless
			if strings.HasSuffix(url, "/") {
				url = strings.TrimSuffix(url, "/") // Trailing slash are not welcome in this house
			}
			urlinternal = append(urlinternal, nakedurl+strings.TrimPrefix(url, "/"))
		}
	}
	return
}

func printfile(url string) { // Check if url is ok or not
	stat, time := getStatus(url)
	if stat == 200 {
		endfile.WriteString("\t" + url + " -> OK (" + time.String() + ")\n") // File input for found url
	} else if stat == 0 {
		endfile.WriteString(url + " -> No server response\n")
	} else {
		endfile.WriteString(url + " -> Error " + strconv.Itoa(stat) + " (" + time.String() + ")\n") // File input for returned errors
	}
	urlfound++
	return
}

func main() {
	mainStart := time.Now()

	urlbase := os.Args[1] // Get the url input

	stripurl, err := regexp.Compile(`https?://.*?/`)
	checkFatal(err)
	nakedurl = stripurl.FindString(urlbase) // Strip url from any trail to get to the origin

	ioutil.WriteFile("result.txt", []byte(""), 0666)                        // Emptying the result file || creating it w/ Satan's help
	endfile, err = os.OpenFile("result.txt", os.O_APPEND|os.O_WRONLY, 0666) // Open result file in write only mode
	checkFatal(err)
	defer endfile.Close()

	checkedurl = make(map[string]bool)

	var exturl, inturl []string       // Create 2 string slices to store urls
	inturl = append(inturl, nakedurl) // Adding the first link to check

	endfile.WriteString("/// INTERNAL CHECK ///\n\n")

	for len(inturl) != 0 { // While there is still urls to check
		if checkedurl[inturl[0]] { // If url has been checked
			inturl = append(inturl[:0], inturl[1:]...) // Delete from list
		} else { // If url has not been checked
			printfile(inturl[0])
			checkedurl[inturl[0]] = true // Storing url in map for later checks
			urlStatus, _ := getStatus(inturl[0])

			if urlStatus == 200 {
				newExturl, newInturl := fetchurl(inturl[0]) // Get new urls
				exturl = append(exturl, newExturl...)       // Add external urls to the list
				inturl = append(inturl[:0], inturl[1:]...)  // Delete current url
				inturl = append(inturl, newInturl...)       // Append newfound internal urls
			} else {
				inturl = append(inturl[:0], inturl[1:]...)
			}
		}
	}

	endfile.WriteString("\n\n")

	endfile.WriteString("/// EXTERNAL CHECK ///\n\n")

	// Lastly, verifying each url leading outside the website
	exturl = removeDuplicates(exturl)
	for _, url := range exturl {
		printfile(url)
	}

	endfile.WriteString("\n\n")
	endfile.WriteString("/// URLS FOUND ///\n\n" + strconv.Itoa(urlfound))

	mainElapsed := time.Since(mainStart)
	endfile.WriteString(" in " + mainElapsed.String())
}
