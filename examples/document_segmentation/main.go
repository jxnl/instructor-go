package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	cohere "github.com/cohere-ai/cohere-go/v2"
	cohereclient "github.com/cohere-ai/cohere-go/v2/client"
	"github.com/jxnl/instructor-go/pkg/instructor"
	"github.com/jxnl/instructor-go/pkg/instructor/core"
	instructor_cohere "github.com/jxnl/instructor-go/pkg/instructor/providers/cohere"
)

type Section struct {
	Title      string `json:"title"         jsonschema:"description=main topic of this section of the document"`
	StartIndex int    `json:"start_index"   jsonschema:"description=line number where the section begins"`
	EndIndex   int    `json:"end_index"     jsonschema:"description=line number where the section ends"`
}

type StructuredDocument struct {
	Sections []Section `json:"sections" jsonschema:"description=a list of sections of the document"`
}

type Segment struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	Start   int    `json:"start"`
	End     int    `json:"end"`
}

func (s Segment) String() string {
	return fmt.Sprintf("Title: %s\nContent:\n%s\nStart: %d\nEnd: %d\n",
		s.Title, s.Content, s.Start, s.End)
}

func (sd *StructuredDocument) PrettyPrint() string {
	s, err := json.MarshalIndent(sd, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(s)
}

func main() {
	ctx := context.Background()

	client := instructor.FromCohere(
		cohereclient.NewClient(cohereclient.WithToken(os.Getenv("COHERE_API_KEY"))),
		instructor.WithMode(instructor.ModeToolCall),
		instructor.WithMaxRetries(3),
	)

	/*
	 *	Document is downloaded from a tutorial on Transformers from Sebastian Raschka: https://sebastianraschka.com/blog/2023/self-attention-from-scratch.html
	 *	Downloaded and scraped via `trafilatura`: https://github.com/adbar/trafilatura
	 */
	doc, err := os.ReadFile("examples/document_segmentation/self-attention-from-scratch.txt")
	if err != nil {
		panic(err)
	}

	getStructuredDocument := func(docWithLines string) *StructuredDocument {
		conversation := core.NewConversation(`
You are a world class educator working on organizing your lecture notes.
Read the document below and extract a StructuredDocument object from it where each section of the document is centered around a single concept/topic that can be taught in one lesson.
Each line of the document is marked with its line number in square brackets (e.g. [1], [2], [3], etc). Use the line numbers to indicate section start and end.
`)
		conversation.AddUserMessage(docWithLines)

		preamble, chatHistory := instructor_cohere.ConversationToMessages(conversation)
		req := &cohere.ChatRequest{
			Model:       toPtr("command-r-plus"),
			ChatHistory: chatHistory,
		}
		if preamble != "" {
			req.Preamble = &preamble
		}

		var structuredDoc StructuredDocument
		_, err := client.Chat(ctx, req, &structuredDoc)
		if err != nil {
			panic(err)
		}
		return &structuredDoc
	}

	documentWithLineNumbers, line2text := docWithLines(string(doc))
	structuredDoc := getStructuredDocument(documentWithLineNumbers)
	segments := getSectionsText(structuredDoc, line2text)

	println(segments[0].String())
	/*
		Title: Introduction to Self-Attention
		Content:
		Understanding and Coding the Self-Attention Mechanism of Large Language Models From Scratch
		In this article, we are going to understand how self-attention works from scratch. This means we will code it ourselves one step at a time.
		Since its introduction via the original transformer paper (Attention Is All You Need), self-attention has become a cornerstone of many state-of-the-art deep learning models, particularly in the field of Natural Language Processing (NLP). Since self-attention is now everywhere, it’s important to understand how it works.
		Self-Attention
		The concept of “attention” in deep learning has its roots in the effort to improve Recurrent Neural Networks (RNNs) for handling longer sequences or sentences. For instance, consider translating a sentence from one language to another. Translating a sentence word-by-word does not work effectively.
		To overcome this issue, attention mechanisms were introduced to give access to all sequence elements at each time step. The key is to be selective and determine which words are most important in a specific context. In 2017, the transformer architecture introduced a standalone self-attention mechanism, eliminating the need for RNNs altogether.
		(For brevity, and to keep the article focused on the technical self-attention details, and I am skipping parts of the motivation, but my Machine Learning with PyTorch and Scikit-Learn book has some additional details in Chapter 16 if you are interested.)
		We can think of self-attention as a mechanism that enhances the information content of an input embedding by including information about the input’s context. In other words, the self-attention mechanism enables the model to weigh the importance of different elements in an input sequence and dynamically adjust their influence on the output. This is especially important for language processing tasks, where the meaning of a word can change based on its context within a sentence or document.
		Note that there are many variants of self-attention. A particular focus has been on making self-attention more efficient. However, most papers still implement the original scaled-dot product attention mechanism discussed in this paper since it usually results in superior accuracy and because self-attention is rarely a computational bottleneck for most companies training large-scale transformers.
		Start: 0
		End: 9
	*/
}

/*
 * Preprocessing utilties
 */

func toPtr[T any](val T) *T {
	return &val
}

func docWithLines(document string) (string, map[int]string) {
	documentLines := strings.Split(document, "\n")
	documentWithLineNumbers := ""
	line2text := make(map[int]string)
	for i, line := range documentLines {
		documentWithLineNumbers += fmt.Sprintf("[%d] %s\n", i, line)
		line2text[i] = line
	}
	return documentWithLineNumbers, line2text
}

func getSectionsText(structuredDoc *StructuredDocument, line2text map[int]string) []Segment {
	var segments []Segment
	for _, s := range structuredDoc.Sections {
		var contents []string
		for lineID := s.StartIndex; lineID < s.EndIndex; lineID++ {
			if line, exists := line2text[lineID]; exists {
				contents = append(contents, line)
			}
		}
		segment := Segment{
			Title:   s.Title,
			Content: strings.Join(contents, "\n"),
			Start:   s.StartIndex,
			End:     s.EndIndex,
		}
		segments = append(segments, segment)
	}
	return segments
}
