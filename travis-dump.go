package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

/* Configuration */
const APIKey = "travis_api_key"
const RepoId = "repo_id"

type TravisUser struct {
	Type           string `json:"@type"`
	Href           string `json:"@href"`
	Representation string `json:"@representation"`
	ID             int    `json:"id"`
	Login          string `json:"login"`
}

type TravisBuild struct {
	Type           string `json:"@type"`
	Href           string `json:"@href"`
	Representation string `json:"@representation"`
	Permissions    struct {
		Read    bool `json:"read"`
		Cancel  bool `json:"cancel"`
		Restart bool `json:"restart"`
	} `json:"@permissions"`
	ID                int         `json:"id"`
	Number            string      `json:"number"`
	State             string      `json:"state"`
	Duration          int         `json:"duration"`
	EventType         string      `json:"event_type"`
	PreviousState     string      `json:"previous_state"`
	PullRequestTitle  string      `json:"pull_request_title"`
	PullRequestNumber interface{} `json:"pull_request_number"`
	StartedAt         time.Time   `json:"started_at"`
	FinishedAt        time.Time   `json:"finished_at"`
	Repository        struct {
		Type           string `json:"@type"`
		Href           string `json:"@href"`
		Representation string `json:"@representation"`
		ID             int    `json:"id"`
		Name           string `json:"name"`
		Slug           string `json:"slug"`
	} `json:"repository"`
	Branch struct {
		Type           string `json:"@type"`
		Href           string `json:"@href"`
		Representation string `json:"@representation"`
		Name           string `json:"name"`
	} `json:"branch"`
	Tag    interface{} `json:"tag"`
	Commit struct {
		Type           string    `json:"@type"`
		Representation string    `json:"@representation"`
		ID             int       `json:"id"`
		Sha            string    `json:"sha"`
		Ref            string    `json:"ref"`
		Message        string    `json:"message"`
		CompareURL     string    `json:"compare_url"`
		CommittedAt    time.Time `json:"committed_at"`
	} `json:"commit"`
	Jobs []struct {
		Type           string `json:"@type"`
		Href           string `json:"@href"`
		Representation string `json:"@representation"`
		ID             int    `json:"id"`
	} `json:"jobs"`
	Stages    []interface{} `json:"stages"`
	CreatedBy TravisUser    `json:"created_by"`
}

type TravisBuildResponse struct {
	Type           string `json:"@type"`
	Href           string `json:"@href"`
	Representation string `json:"@representation"`
	Pagination     struct {
		Limit   int  `json:"limit"`
		Offset  int  `json:"offset"`
		Count   int  `json:"count"`
		IsFirst bool `json:"is_first"`
		IsLast  bool `json:"is_last"`
		Next    struct {
			Href   string `json:"@href"`
			Offset int    `json:"offset"`
			Limit  int    `json:"limit"`
		} `json:"next"`
		Prev  interface{} `json:"prev"`
		First struct {
			Href   string `json:"@href"`
			Offset int    `json:"offset"`
			Limit  int    `json:"limit"`
		} `json:"first"`
		Last struct {
			Href   string `json:"@href"`
			Offset int    `json:"offset"`
			Limit  int    `json:"limit"`
		} `json:"last"`
	} `json:"@pagination"`
	Builds []TravisBuild `json:"builds"`
}

func main() {
	writer := createWriter()
	processAllBuilds(RepoId, 7200, writer)
	writer.Flush()
}

func processAllBuilds(repo string, nextOffset int, writer *csv.Writer) {
	limit := 100
	fetched := 0
	initOffset := nextOffset

	for {
		builds := getBuilds(repo, limit, nextOffset)
		fetched += len(builds.Builds)
		fmt.Println("Progress: ", fetched, " of: ", builds.Pagination.Count-initOffset)
		writeToCSV(builds.Builds, writer)
		if builds.Pagination.Next.Offset == 0 {
			break // Reached the end
		}
		nextOffset = builds.Pagination.Next.Offset
	}
}

func createWriter() *csv.Writer {
	file, _ := os.Create("travis-builds.csv")
	writer := csv.NewWriter(file)

	// Headers
	headers := []string{
		"ID",
		"Number",
		"State",
		"EventType",
		"RepositoryName",
		"BranchName",
		"PullRequestTitle",
		"StartedAt",
		"FinishedAt",
		"Duration",
		"CreatedByID",
		"CreatedByLogin"}

	writer.Write(headers)
	return writer
}

func writeToCSV(builds []TravisBuild, writer *csv.Writer) {
	// Content
	for _, build := range builds {
		line := []string{
			strconv.FormatInt(int64(build.ID), 10),
			build.Number,
			build.State,
			build.EventType,
			build.Repository.Name,
			build.Branch.Name,
			build.PullRequestTitle,
			build.StartedAt.Format(time.RFC3339),
			build.FinishedAt.Format(time.RFC3339),
			strconv.FormatInt(int64(build.Duration), 10),
			strconv.FormatInt(int64(build.CreatedBy.ID), 10),
			build.CreatedBy.Login}
		writer.Write(line)
	}
}

func getBuilds(repo string, limit int, offset int) TravisBuildResponse {
	var buildResponse TravisBuildResponse
	url := fmt.Sprintf("https://api.travis-ci.com/repo/%s/builds?limit=%d&offset=%d&sort_by=started_at:desc", repo, limit, offset)
	token := fmt.Sprintf("token %s", APIKey)

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		panic(err)
	}

	req.Header.Add("Travis-API-Version", "3")
	req.Header.Add("Authorization", token)
	resp, err := client.Do(req)

	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&buildResponse); err != nil {
		log.Println(err)
	}
	return buildResponse
}
