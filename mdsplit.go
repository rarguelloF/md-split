package mdsplit

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/russross/blackfriday/v2"
)

const (
	MaxGithubCommentSize = 65536
)

type wrapper struct {
	begin string
	end   string
}

type chunk struct {
	content  string
	wrappers []*wrapper
}

// SplitGithubComment is an alias of MarkdownSplit using MaxGithubCommentSize.
func SplitGithubComment(text, sep string) ([]string, bool) {
	return MarkdownSplit(text, MaxGithubCommentSize, sep)
}

// MarkdownSplit tries to perform a markdown split based on max length and a separator string,
// preserving markdown syntax on the chunked splits as much as possible.
// If it's not possible, it fallbacks to simple split method.
//
// Returns the text splits and a bool informing if it was able to do markdown split successfully or not.
func MarkdownSplit(text string, max int, sep string) ([]string, bool) {
	// If we're under the limit then no need to split.
	if len(text) <= max {
		return []string{text}, true
	}

	// If we can't fit the separator string in then this doesn't make sense.
	if max <= len(sep) {
		return nil, false
	}

	var chunks []*chunk
	baseTitle := ""
	titleLen := 0
	titleSuffixFmt := " (%d/%s)\n\n"
	canSplit := true

	var htmlWrappers []*wrapper

	md := blackfriday.New(blackfriday.WithExtensions(blackfriday.Strikethrough))
	rootNode := md.Parse([]byte(text))

	rootNode.Walk(func(node *blackfriday.Node, entering bool) blackfriday.WalkStatus {
		switch node.Type {
		case blackfriday.List:
			// TODO: change when lists are actually implemented
			canSplit = false
			return blackfriday.Terminate
		}

		if node.Literal == nil {
			return blackfriday.GoToNext
		}

		contents := string(node.Literal)
		var wrappers []*wrapper

		parent := node.Parent
		for parent != nil {
			switch parent.Type {
			case blackfriday.Del:
				wrappers = append(wrappers, &wrapper{begin: "~~", end: "~~"})

			case blackfriday.Emph:
				wrappers = append(wrappers, &wrapper{begin: "_", end: "_"})

			case blackfriday.Strong:
				wrappers = append(wrappers, &wrapper{begin: "**", end: "**"})

			case blackfriday.Heading:
				heading := strings.Repeat("#", parent.Level)

				if baseTitle == "" && len(chunks) == 0 {
					baseTitle = fmt.Sprintf("%s %s", heading, contents)

					// give extra 10 characters to the title, just in case the totalComments grow too much
					titleLen = len(baseTitle) + len(titleSuffixFmt) + 10

					return blackfriday.GoToNext
				}

				wrappers = append(wrappers, &wrapper{begin: heading + " ", end: "\n\n"})

			case blackfriday.Link:
				linkData := parent.LinkData

				var sb strings.Builder
				sb.WriteString("](")

				linkDest, linkTitle := string(linkData.Destination), string(linkData.Title)
				if linkDest != "" {
					sb.WriteString(linkDest)
				}

				if linkTitle != "" {
					sb.WriteString(fmt.Sprintf(" \"%s\"", linkTitle))
				}

				sb.WriteString(")")

				begin, end := "[", sb.String()
				wrappers = append(wrappers, &wrapper{begin: begin, end: end})
			}

			parent = parent.Parent
		}

		switch node.Type {
		case blackfriday.Code:
			begin, end := "```\n", "\n```"

			lineBreakIdx := strings.Index(contents, "\n")
			if lineBreakIdx != -1 {
				prefix := contents[:lineBreakIdx+1]
				contents = strings.TrimLeft(contents, prefix)
				begin = "```" + prefix
			}

			// remove latest linebreak from code
			contents = strings.TrimRight(contents, "\n")
			wrappers = append(wrappers, &wrapper{begin: begin, end: end})

		case blackfriday.HTMLSpan:
			if isHTMLOpeningTag(contents) {
				// close automatically, even if tag wasn't closed in original text
				htmlWrappers = append(htmlWrappers, &wrapper{contents, getHTMLClosingTag(contents)})
				contents = ""
			} else {
				// check if it's closing the last opened tag, if not, it's badly constructed html
				if len(htmlWrappers) > 0 && contents == htmlWrappers[len(htmlWrappers)-1].end {
					htmlWrappers = htmlWrappers[:len(htmlWrappers)-1]
					contents = ""
				}
			}
		}

		// add pending htmlWrappers to current wrappers, in case there are any
		wrappers = append(wrappers, htmlWrappers...)

		wLen := 0
		for _, w := range wrappers {
			wLen += len(w.begin) + len(w.end)
		}

		sepLen := len(sep)

		// sum the length of the extra added contents, apart from the text contents
		extraLen := wLen + titleLen + sepLen

		if extraLen >= max {
			// we don't have enough space to do this, so just perform a simple text split
			canSplit = false
			return blackfriday.Terminate
		}

		chunkLen := max - extraLen
		chunks = append(chunks, buildChunks(contents, chunkLen, wrappers)...)

		return blackfriday.GoToNext
	})

	if !canSplit {
		return SimpleSplit(text, max, sep), false
	}

	return chunksAsStr(chunks, max, baseTitle, titleSuffixFmt), true
}

// SimpleSplit performs a simple split based on max length and a separator string.
func SimpleSplit(text string, max int, sep string) []string {
	// If we're under the limit then no need to split.
	if len(text) <= max {
		return []string{text}
	}

	// If we can't fit the separator string in then this doesn't make sense.
	if max <= len(sep) {
		return nil
	}

	var chunks []string

	maxSize := max - len(sep)
	numChunks := int(math.Ceil(float64(len(text)) / float64(maxSize)))

	for i := 0; i < numChunks; i++ {
		upTo := min(len(text), (i+1)*maxSize)
		portion := text[i*maxSize : upTo]
		if i < numChunks-1 {
			portion += sep
		}
		chunks = append(chunks, portion)
	}

	return chunks
}

func isHTMLOpeningTag(tag string) bool {
	if strings.HasPrefix(tag, "</") {
		return false
	}
	return true
}

func getHTMLClosingTag(open string) string {
	return strings.Replace(open, "<", "</", 1)
}

func buildChunks(contents string, chunkLen int, wrappers []*wrapper) []*chunk {
	var result []*chunk

	for contents != "" {
		c := &chunk{}
		c.wrappers = wrappers

		if len(contents) <= chunkLen {
			c.content = contents
			contents = ""
		} else {
			c.content = contents[0:chunkLen]
			contents = contents[chunkLen:]
		}

		result = append(result, c)
	}

	return result
}

func chunksAsStr(chunks []*chunk, max int, baseTitle, titleSuffixFmt string) []string {
	titleTotalID := fmt.Sprintf("<%s>", uuid.New().String())

	var result []string
	curChunk := 1

	for _, cm := range chunks {
		cmStr := ""

		for _, w := range cm.wrappers {
			cmStr = w.begin + cmStr
		}

		cmStr = cmStr + cm.content

		for _, w := range cm.wrappers {
			cmStr = cmStr + w.end
		}

		if len(result) > 0 {
			prev := result[len(result)-1]

			if len(prev)+len(cmStr) <= max {
				result[len(result)-1] += cmStr
				continue
			}
		}

		if baseTitle != "" {
			title := baseTitle + fmt.Sprintf(titleSuffixFmt, curChunk, titleTotalID)
			cmStr = title + cmStr
		}

		result = append(result, cmStr)
		curChunk += 1
	}

	totalStr := strconv.Itoa(len(result))

	for i := 0; i < len(result); i++ {
		result[i] = strings.Replace(result[i], titleTotalID, totalStr, 1)
	}

	return result
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
