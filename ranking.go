package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/net/html"
)

const (
	brenschProfile = "https://play.battlesnake.com/profile/brensch" // you're the only one who matters
)

type CompetitionResult struct {
	Name  string
	Score int
	Rank  int
}

func GetCompetitionResults() ([]CompetitionResult, error) {
	// Replace brenschProfile with the actual URL
	resp, err := http.Get(brenschProfile)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve URL: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse the HTML
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	var results []CompetitionResult

	// Traverse the DOM tree to find competition blocks
	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" {
			if hasClasses(n, []string{"card", "p-1", "text-white"}) {
				result := CompetitionResult{}
				extractCompetitionDetails(n, &result)
				results = append(results, result)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}

	traverse(doc)

	return results, nil
}

func extractCompetitionDetails(n *html.Node, result *CompetitionResult) {
	var f func(*html.Node)
	f = func(node *html.Node) {
		if node.Type == html.ElementNode {
			if node.Data == "h4" && hasClasses(node, []string{"text-center", "text-lg", "font-bold", "uppercase"}) {
				result.Name = strings.TrimSpace(getNodeText(node))
			} else if node.Data == "p" {
				if hasClasses(node, []string{"text-4xl", "text-center", "font-bold"}) || hasClasses(node, []string{"text-2xl", "text-center", "font-bold"}) {
					scoreStr := strings.TrimSpace(getNodeText(node))
					scoreStr = strings.ReplaceAll(scoreStr, ",", "")
					if scoreStr != "--" {
						score, err := strconv.Atoi(scoreStr)
						if err == nil {
							result.Score = score
						}
					}
				} else if hasClasses(node, []string{"text-lg", "text-center", "text-sm"}) {
					rankStr := extractRank(node)
					if rankStr != "" {
						rank, err := strconv.Atoi(rankStr)
						if err == nil {
							result.Rank = rank
						}
					}
				}
			}
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
}

func getAttr(n *html.Node, attrName string) string {
	for _, attr := range n.Attr {
		if attr.Key == attrName {
			return attr.Val
		}
	}
	return ""
}

func hasClasses(n *html.Node, requiredClasses []string) bool {
	classAttr := getAttr(n, "class")
	classes := strings.Fields(classAttr)
	classMap := make(map[string]bool)
	for _, class := range classes {
		classMap[class] = true
	}
	for _, required := range requiredClasses {
		if !classMap[required] {
			return false
		}
	}
	return true
}

func getNodeText(n *html.Node) string {
	var buf bytes.Buffer
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.TextNode {
			buf.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	return buf.String()
}

func extractRank(n *html.Node) string {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "big" {
			rankStr := getNodeText(c)
			rankStr = strings.TrimFunc(rankStr, func(r rune) bool {
				return !unicode.IsDigit(r)
			})
			return rankStr
		}
	}
	return ""
}
