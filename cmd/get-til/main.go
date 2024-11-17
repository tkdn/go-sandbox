package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
)

var (
	ErrEsaEndpointRequired  = errors.New("ESA_ENDPOINT is requied")
	ErrEsaAuthTokenRequired = errors.New("ESA_AUTH_TOKEN is requied")
	ErrEsaTeamURLRequired   = errors.New("ESA_TEAM_URL is requied")
)

var logger = &Logger{slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{}))}

type Logger struct {
	logger *slog.Logger
}

func (l *Logger) Info(log string) { l.logger.Info(log) }
func (l *Logger) Error(err error) { l.logger.Error(err.Error()) }

func main() {
	esaEndpoint := os.Getenv("ESA_ENDPOINT")
	if esaEndpoint == "" {
		logger.Error(ErrEsaEndpointRequired)
		return
	}
	authToken := os.Getenv("ESA_AUTH_TOKEN")
	if authToken == "" {
		logger.Error(ErrEsaAuthTokenRequired)
		return
	}
	esaTeamURL := os.Getenv("ESA_TEAM_URL")
	if esaTeamURL == "" {
		logger.Error(ErrEsaTeamURLRequired)
		return
	}

	res, err := requestEsa(esaEndpoint, authToken)
	if err != nil {
		logger.Error(err)
		return
	}

	file, err := os.Create("./out/tils.txt")
	if err != nil {
		logger.Error(err)
		return
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, p := range res.Posts {
		reader := strings.NewReader(p.BodyMarkdown)
		if err := pipeline(reader, writer, p.EsaTitle, esaTeamURL); err != nil {
			logger.Error(err)
			return
		}
	}
	if err := writer.Flush(); err != nil {
		logger.Error(err)
		return
	}
	logger.Info("completed: ファイルに書き込みました")
}

type esaRes struct {
	Posts []struct {
		BodyMarkdown string `json:"body_md"`
		EsaTitle     string `json:"full_name"`
	} `json:"posts"`
}

func requestEsa(endpoint, authToken string) (*esaRes, error) {
	reqURL := buildRequestURL(endpoint)
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+authToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var res esaRes
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return &res, nil
}

var (
	queryKeyVal = map[string]string{
		"in":      "日報",
		"user":    "tkdn",
		"sort":    "created-asc",
		"created": ">2024-04-22",
	}
	postPerPage = "100"
)

func buildRequestURL(endpoint string) string {
	u, _ := url.Parse(endpoint)
	qa := make([]string, 0, len(queryKeyVal))
	for k, v := range queryKeyVal {
		qa = append(qa, k+":"+v)
	}
	qs := strings.Join(qa, " ")
	query := url.Values{}
	query.Add("q", qs)
	query.Add("per_page", postPerPage)
	u.RawQuery = query.Encode()
	return u.String()
}

func extractTILSection(input io.Reader, output io.Writer, title string) error {
	tilHeading := "# わかったこと"
	collect := false

	scanner := bufio.NewScanner(input)
	writer := bufio.NewWriter(output)
	defer writer.Flush()

	fmt.Fprintln(writer, "- "+title)
	for scanner.Scan() {
		line := scanner.Text()
		// ignore blank space
		if strings.TrimSpace(line) == "" {
			continue
		}
		// detect not til heading. checke til section had been already collected
		if strings.HasPrefix(line, "#") && line != tilHeading {
			if collect {
				break
			}
		}
		if line == tilHeading {
			collect = true
			continue
		}
		if collect {
			fmt.Fprintln(writer, "  "+line)
		}
	}
	return scanner.Err()
}

func convertToConsenseFormat(input io.Reader, output io.Writer, esaTeamURL string) error {
	scanner := bufio.NewScanner(input)
	writer := bufio.NewWriter(output)
	defer writer.Flush()

	prevLine := "- TIL"
	indentLevel := 1
	for scanner.Scan() {
		currentLine := scanner.Text()
		trimedCurrentLine := strings.TrimLeft(currentLine, "- ")
		currentSpaces := len(currentLine) - len(trimedCurrentLine)
		trimedPrevLine := strings.TrimLeft(prevLine, "- ")
		prevSpaces := len(prevLine) - len(trimedPrevLine)

		if currentSpaces > 0 {
			if currentSpaces > prevSpaces {
				indentLevel++
			} else if currentSpaces < prevSpaces {
				indentLevel--
			}
		} else {
			indentLevel = 1
		}
		consensedLine := mdLinkToCosenseLink(trimedCurrentLine, esaTeamURL)
		indent := strings.Repeat("\t", indentLevel)
		fmt.Fprintln(writer, indent+consensedLine)
		prevLine = currentLine
	}
	return scanner.Err()
}

func mdLinkToCosenseLink(markdown, esaTeamURL string) string {
	re := regexp.MustCompile(`\[(.*?)\]\((.*?)\)`)

	line := re.ReplaceAllStringFunc(markdown, func(match string) string {
		matches := re.FindStringSubmatch(match)
		if len(match) > 2 {
			text := matches[1]
			link := matches[2]
			if !strings.HasPrefix(link, "http") {
				link = esaTeamURL + link
			}
			return fmt.Sprintf("[%s %s]", text, link)
		}
		return match
	})
	return line
}

func pipeline(input io.Reader, output io.Writer, title, esaTeamURL string) error {
	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()
		if err := extractTILSection(input, pw, title); err != nil {
			pw.CloseWithError(err)
		}
	}()
	if err := convertToConsenseFormat(pr, output, esaTeamURL); err != nil {
		return err
	}
	return nil
}
