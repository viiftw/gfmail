package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Get create a get request
func Get(query, url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("User-Agent", "GoodBoy")
	req.Header.Add("Accept", "*/*")
	req.Header.Add("Cache-Control", "no-cache")
	// req.Header.Add("Accept-Encoding", "gzip, deflate")
	req.Header.Add("Connection", "keep-alive")
	req.Header.Add("cache-control", "no-cache")

	req.URL.RawQuery = query
	trt := &http.Transport{
		TLSClientConfig:    &tls.Config{InsecureSkipVerify: true},
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: true,
	}
	client := &http.Client{Transport: trt}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func main() {
	username := flag.String("u", "", "Username want to find mails")

	flag.Parse()
	if *username == "" {
		fmt.Println("-u username is required")
		os.Exit(1)
	}

	fmt.Println(*username)
	user := *username
	findFromNPM(user)
	findFromRecentCommits(user)
	findFromRecentActivity(user)
}
func findFromNPM(user string) {
	fmt.Println("-----Emails on npm-----")
	userURL := "https://registry.npmjs.org/-/user/org.couchdb.user:" + user

	q := url.Values{}
	byteBody, err := Get(q.Encode(), userURL)
	if err != nil {
		fmt.Println("Error when do get request to npm api")
		return
	}

	// regular expression pattern
	regE := regexp.MustCompile(`"(email)":"([^"]+)"`)
	fmt.Println(regE.FindAllString(string(byteBody), -1))
}

func findFromRecentCommits(user string) {
	fmt.Println("-----Emails from recent commits-----")
	userURL := "https://api.github.com/users/" + user + "/events"
	byteBody, err := Get("", userURL)
	if err != nil {
		fmt.Println("Error when do get request to github api")
		return
	}

	// regular expression pattern
	reg := regexp.MustCompile(`"(email)":"([^"]+)"`)
	stringArr := unique(reg.FindAllString(string(byteBody), -1))
	for i := 0; i < len(stringArr); i++ {
		tmp := strings.ReplaceAll(stringArr[i], `"email":"`, "")
		fmt.Println(tmp[:len(tmp)-1])
	}
}

func unique(stringSlice []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range stringSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

func getAllRepos(user string) ([]string, error) {
	userURL := "https://api.github.com/users/" + user + "/repos"
	q := url.Values{}
	q.Add("type", "owner")
	q.Add("sort", "updated")
	byteBody, err := Get(q.Encode(), userURL)
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}
	json.Unmarshal(byteBody, &results)
	var allRepo []string
	for _, result := range results {
		allRepo = append(allRepo, result["name"].(string))
	}
	return allRepo, nil
}

func findFromRecentActivity(user string) {
	fmt.Println("-----Emails from owned-repo recent activity-----")

	repos, err := getAllRepos(user)
	if err != nil {
		fmt.Println("Error when get owned-repo recent activity")
		return
	}
	for _, repo := range repos {
		go findFromRepo(user, repo)
	}
	wg.Wait()
	for i := 0; i < len(emailFromRepos); i++ {
		fmt.Println(emailFromRepos[i])
	}
}

var wg sync.WaitGroup
var emailFromRepos []string
var mutex = &sync.Mutex{}

func findFromRepo(user, repo string) {
	wg.Add(1)
	defer wg.Done()
	userURL := "https://api.github.com/repos/" + user + "/" + repo + "/commits"
	byteBody, err := Get("", userURL)
	if err != nil {
		// fmt.Println("Error when get details owned-repo recent activity")
		return
	}

	// regular expression pattern
	reg := regexp.MustCompile(`"(email)":"([^"]+)"`)
	stringArr := unique(reg.FindAllString(string(byteBody), -1))
	for i := 0; i < len(stringArr); i++ {
		tmp := strings.ReplaceAll(stringArr[i], `"email":"`, "")
		mutex.Lock()
		emailFromRepos = appendIfMissing(emailFromRepos, tmp[:len(tmp)-1])
		mutex.Unlock()
	}
}

// It's simple and obvious and will be fast for small lists.
func appendIfMissing(slice []string, i string) []string {
	for _, ele := range slice {
		if ele == i {
			return slice
		}
	}
	return append(slice, i)
}
