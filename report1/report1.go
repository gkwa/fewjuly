package report1

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"text/template"

	"golang.org/x/net/html"
)

var scratch = "."

type Record struct {
	Content string
}

func RunReport1() error {
	url := "https://news.ycombinator.com/item?id=39232976"
	cacheFilePath := "data.txt"

	var err error

	err = fetchDataAndCache(url, cacheFilePath)
	if err != nil {
		return fmt.Errorf("error fetching data: %v", err)
	}

	err = ensureScratchDirectory()
	if err != nil {
		return fmt.Errorf("error ensuring scratch directory: %v", err)
	}

	err = buildTemplates()
	if err != nil {
		return fmt.Errorf("error building templates: %v", err)
	}

	return nil
}

func filterRecords(records []Record, pattern string) []Record {
	var filteredRecords []Record

	regex := regexp.MustCompile("(?i)" + pattern)

	for _, r := range records {
		if regex.MatchString(r.Content) {
			filteredRecords = append(filteredRecords, r)
		}
	}

	return filteredRecords
}

func ensureScratchDirectory() error {
	_, err := os.Stat(scratch)
	if os.IsNotExist(err) {
		err := os.Mkdir(scratch, os.ModePerm)
		if err != nil {
			return fmt.Errorf("error creating 'scratch' directory: %v", err)
		}
	}
	return nil
}

func buildTemplates() error {
	fname := "data.txt"
	records, err := parse(fname)
	if err != nil {
		return err
	}

	pattern := "cue|cuelang"

	filteredRecords := filterRecords(records, pattern)
	records = filteredRecords

	batchSize := 75

	for i := 0; i < len(records); i += batchSize {
		end := i + batchSize
		if end > len(records) {
			end = len(records)
		}
		batch := records[i:end]

		tmpl := `
In this request, please do not reply with code or code blocks.
I'm asking that you summarize many people's opinions and
experiences in a few paragraphs..

This article is about Pkl, but please find mention of CUE
language in the article.

Please summarize the pros and cons for the use of CUE.

For your reference, CUE and Cuelang are synonyms for the
same language.

{{range .}}
{{.Content}}
{{end}}
`
		t := template.Must(template.New("batchTemplate").Parse(tmpl))

		fileName := fmt.Sprintf("%s/result_%02d.txt", scratch, i/batchSize+1)
		outputFile, err := os.Create(fileName)
		if err != nil {
			return fmt.Errorf("error creating output file: %v", err)
		}
		defer outputFile.Close()

		err = t.Execute(outputFile, batch)
		if err != nil {
			return fmt.Errorf("error executing template: %v", err)
		}
	}

	return nil
}

func parse(fname string) ([]Record, error) {
	file, err := os.Open(fname)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()

	var records []Record
	var currentRecord strings.Builder
	inRecord := false

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()

		if strings.TrimSpace(line) == "reply" {
			if inRecord {
				record := Record{Content: currentRecord.String()}
				records = append(records, record)
				currentRecord.Reset()
			}
			inRecord = true
		} else {
			if inRecord {
				currentRecord.WriteString(line)
				currentRecord.WriteString("\n")
			}
		}
	}

	if currentRecord.Len() > 0 {
		record := Record{Content: currentRecord.String()}
		records = append(records, record)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning file: %v", err)
	}

	fmt.Printf("Number of records in the file: %d\n", len(records))

	return records, nil
}

func fetchDataAndCache(url, cacheFilePath string) error {
	_, err := os.Stat(cacheFilePath)
	if err == nil {
		return fmt.Errorf("data already cached in %s", cacheFilePath)
	}

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("error fetching data from %s: %v", url, err)
	}
	defer resp.Body.Close()

	file, err := os.Create(cacheFilePath)
	if err != nil {
		return fmt.Errorf("error creating cache file: %v", err)
	}
	defer file.Close()

	textContent, err := parseHTML(resp.Body)
	if err != nil {
		return fmt.Errorf("error parsing HTML: %v", err)
	}

	_, err = file.WriteString(textContent)
	if err != nil {
		return fmt.Errorf("error writing data to %s: %v", cacheFilePath, err)
	}

	fmt.Println("Data fetched and cached as text in", cacheFilePath)
	return nil
}

func parseHTML(r io.Reader) (string, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return "", err
	}

	var textContentBuilder strings.Builder
	var extractText func(*html.Node)

	extractText = func(n *html.Node) {
		if n.Type == html.TextNode {
			textContentBuilder.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extractText(c)
		}
	}

	extractText(doc)
	return textContentBuilder.String(), nil
}
