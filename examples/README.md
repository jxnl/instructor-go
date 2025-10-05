# Examples

<details>
<summary>Function Calling with OpenAI</summary>

Running

```bash
export OPENAI_API_KEY=<Your OpenAI API Key>
go run examples/function_calling/main.go
```

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/instructor-ai/instructor-go/pkg/instructor"
	openai "github.com/sashabaranov/go-openai"
)

type SearchType string

const (
	Web   SearchType = "web"
	Image SearchType = "image"
	Video SearchType = "video"
)

type Search struct {
	Topic string     `json:"topic" jsonschema:"title=Topic,description=Topic of the search,example=golang"`
	Query string     `json:"query" jsonschema:"title=Query,description=Query to search for relevant content,example=what is golang"`
	Type  SearchType `json:"type"  jsonschema:"title=Type,description=Type of search,default=web,enum=web,enum=image,enum=video"`
}

func (s *Search) execute() {
	fmt.Printf("Searching for `%s` with query `%s` using `%s`\n", s.Topic, s.Query, s.Type)
}

type Searches = []Search

func segment(ctx context.Context, data string) *Searches {

	client := instructor.FromOpenAI(
		openai.NewClient(os.Getenv("OPENAI_API_KEY")),
		instructor.WithMode(instructor.ModeToolCall),
		instructor.WithMaxRetries(3),
	)

	var searches Searches
	_, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: openai.GPT4o,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: fmt.Sprintf("Consider the data below: '\n%s' and segment it into multiple search queries", data),
			},
		},
	},
		&searches,
	)
	if err != nil {
		panic(err)
	}

	return &searches
}

func main() {
	ctx := context.Background()

	q := "Search for a picture of a cat, a video of a dog, and the taxonomy of each"
	for _, search := range *segment(ctx, q) {
		search.execute()
	}
	/*
		Searching for `cat` with query `picture of a cat` using `image`
		Searching for `dog` with query `video of a dog` using `video`
		Searching for `cat` with query `taxonomy of a cat` using `web`
		Searching for `dog` with query `taxonomy of a dog` using `web`
	*/
}
```

</details>

<details>
<summary>Text Classification with Anthropic</summary>

Running

```bash
export ANTHROPIC_API_KEY=<Your Anthropic API Key>
go run examples/classification/main.go
```

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/instructor-ai/instructor-go/pkg/instructor"
	anthropic "github.com/liushuangls/go-anthropic/v2"
)

type LabelType string

const (
	LabelTechIssue    LabelType = "tech_issue"
	LabelBilling      LabelType = "billing"
	LabelGeneralQuery LabelType = "general_query"
)

type Label struct {
	Type LabelType `json:"type" jsonschema:"title=Label type,description=Type of the label,enum=tech_issue,enum=billing,enum=general_query"`
}

type Prediction struct {
	Labels []Label `json:"labels" jsonschema:"title=Predicted labels,description=Labels of the prediction"`
}

func classify(data string) *Prediction {
	ctx := context.Background()

	client := instructor.FromAnthropic(
		anthropic.NewClient(os.Getenv("ANTHROPIC_API_KEY")),
		instructor.WithMode(instructor.ModeToolCall),
		instructor.WithMaxRetries(3),
	)

	var prediction Prediction
	resp, err := client.CreateMessages(ctx, anthropic.MessagesRequest{
		Model: anthropic.ModelClaude3Haiku20240307,
		Messages: []anthropic.Message{
			anthropic.NewUserTextMessage(fmt.Sprintf("Classify the following support ticket: %s", data)),
		},
		MaxTokens: 500,
	},
		&prediction,
	)
	_ = resp // sends back original response so no information loss from original API
	if err != nil {
		panic(err)
	}

	return &prediction
}

func main() {

	ticket := "My account is locked and I can't access my billing info."
	prediction := classify(ticket)

	assert(prediction.contains(LabelTechIssue), "Expected ticket to be related to tech issue")
	assert(prediction.contains(LabelBilling), "Expected ticket to be related to billing")
	assert(!prediction.contains(LabelGeneralQuery), "Expected ticket NOT to be a general query")

	fmt.Printf("%+v\n", prediction)
	/*
		&{Labels:[{Type:tech_issue} {Type:billing}]}
	*/
}

/******/

func (p *Prediction) contains(label LabelType) bool {
	for _, l := range p.Labels {
		if l.Type == label {
			return true
		}
	}
	return false
}

func assert(condition bool, message string) {
	if !condition {
		fmt.Println("Assertion failed:", message)
	}
}
```

</details>

<details>
<summary>Images with OpenAI</summary>

![List of books](https://raw.githubusercontent.com/instructor-ai/instructor-go/main/examples/vision/openai/books.png)

Running

```bash
export OPENAI_API_KEY=<Your OpenAI API Key>
go run examples/vision/openai/main.go
```

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/instructor-ai/instructor-go/pkg/instructor"
	openai "github.com/sashabaranov/go-openai"
)

type Book struct {
	Title  string  `json:"title,omitempty"  jsonschema:"title=title,description=The title of the book,example=Harry Potter and the Philosopher's Stone"`
	Author *string `json:"author,omitempty" jsonschema:"title=author,description=The author of the book,example=J.K. Rowling"`
}

type BookCatalog struct {
	Catalog []Book `json:"catalog"`
}

func (bc *BookCatalog) PrintCatalog() {
	fmt.Printf("Number of books in the catalog: %d\n\n", len(bc.Catalog))
	for _, book := range bc.Catalog {
		fmt.Printf("Title:  %s\n", book.Title)
		fmt.Printf("Author: %s\n", *book.Author)
		fmt.Println("--------------------")
	}
}

func main() {
	ctx := context.Background()

	client := instructor.FromOpenAI(
		openai.NewClient(os.Getenv("OPENAI_API_KEY")),
		instructor.WithMode(instructor.ModeJSON),
		instructor.WithMaxRetries(3),
	)

	url := "https://raw.githubusercontent.com/instructor-ai/instructor-go/main/examples/vision/openai/books.png"

	var bookCatalog BookCatalog
	_, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: openai.GPT4o,
		Messages: []openai.ChatCompletionMessage{
			{
				Role: openai.ChatMessageRoleUser,
				MultiContent: []openai.ChatMessagePart{
					{
						Type: openai.ChatMessagePartTypeText,
						Text: "Extract book catalog from the image",
					},
					{
						Type: openai.ChatMessagePartTypeImageURL,
						ImageURL: &openai.ChatMessageImageURL{
							URL: url,
						},
					},
				},
			},
		},
	},
		&bookCatalog,
	)

	if err != nil {
		panic(err)
	}

	bookCatalog.PrintCatalog()
	/*
		Number of books in the catalog: 15

		Title:  Pride and Prejudice
		Author: Jane Austen
		--------------------
		Title:  The Great Gatsby
		Author: F. Scott Fitzgerald
		--------------------
		Title:  The Catcher in the Rye
		Author: J. D. Salinger
		--------------------
		Title:  Don Quixote
		Author: Miguel de Cervantes
		--------------------
		Title:  One Hundred Years of Solitude
		Author: Gabriel García Márquez
		--------------------
		Title:  To Kill a Mockingbird
		Author: Harper Lee
		--------------------
		Title:  Beloved
		Author: Toni Morrison
		--------------------
		Title:  Ulysses
		Author: James Joyce
		--------------------
		Title:  Harry Potter and the Cursed Child
		Author: J.K. Rowling
		--------------------
		Title:  The Grapes of Wrath
		Author: John Steinbeck
		--------------------
		Title:  1984
		Author: George Orwell
		--------------------
		Title:  Lolita
		Author: Vladimir Nabokov
		--------------------
		Title:  Anna Karenina
		Author: Leo Tolstoy
		--------------------
		Title:  Moby-Dick
		Author: Herman Melville
		--------------------
		Title:  Wuthering Heights
		Author: Emily Brontë
		--------------------
	*/
}
```

</details>

<details>
<summary>Images with Anthropic</summary>

![List of movies](https://raw.githubusercontent.com/instructor-ai/instructor-go/main/examples/vision/anthropic/movies.png)

Running

```bash
export ANTHROPIC_API_KEY=<Your Anthropic API Key>
go run examples/vision/anthropic/main.go
```

```go
package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/instructor-ai/instructor-go/pkg/instructor"
	"github.com/liushuangls/go-anthropic/v2"
)

type Movie struct {
	Title string `json:"title"          jsonschema:"title=title,description=The title of the movie,required=true,example=Ex Machina"`
	Year  int    `json:"year,omitempty" jsonschema:"title=year,description=The year of the movie,required=false,example=2014"`
}

type MovieCatalog struct {
	Catalog []Movie `json:"catalog"`
}

func (bc *MovieCatalog) PrintCatalog() {
	fmt.Printf("Number of movies in the catalog: %d\n\n", len(bc.Catalog))
	for _, movie := range bc.Catalog {
		fmt.Printf("Title:  %s\n", movie.Title)
		if movie.Year != 0 {
			fmt.Printf("Year:   %d\n", movie.Year)
		}
		fmt.Println("--------------------")
	}
}

func main() {
	ctx := context.Background()

	client := instructor.FromAnthropic(
		anthropic.NewClient(os.Getenv("ANTHROPIC_API_KEY")),
		instructor.WithMode(instructor.ModeJSONSchema),
		instructor.WithMaxRetries(3),
	)

	url := "https://raw.githubusercontent.com/instructor-ai/instructor-go/main/examples/vision/anthropic/movies.jpg"
	data, err := urlToBase64(url)
	if err != nil {
		panic(err)
	}

	var movieCatalog MovieCatalog
	_, err = client.CreateMessages(ctx, anthropic.MessagesRequest{
		Model: "claude-3-haiku-20240307",
		Messages: []anthropic.Message{
			{
				Role: anthropic.RoleUser,
				Content: []anthropic.MessageContent{
					anthropic.NewImageMessageContent(anthropic.MessageContentImageSource{
						Type:      "base64",
						MediaType: "image/jpeg",
						Data:      data,
					}),
					anthropic.NewTextMessageContent("Extract the movie catalog from the screenshot"),
				},
			},
		},
		MaxTokens: 1000,
	},
		&movieCatalog,
	)
	if err != nil {
		panic(err)
	}

	movieCatalog.PrintCatalog()
	/*
		Number of movies in the catalog: 18

		Title:  Oppenheimer
		Year:   2023
		--------------------
		Title:  The Dark Knight
		Year:   2008
		--------------------
		Title:  Interstellar
		Year:   2014
		--------------------
		Title:  Inception
		Year:   2010
		--------------------
		Title:  Tenet
		Year:   2020
		--------------------
		Title:  Dunkirk
		Year:   2017
		--------------------
		Title:  Memento
		Year:   2000
		--------------------
		Title:  The Dark Knight Rises
		Year:   2012
		--------------------
		Title:  Batman Begins
		Year:   2005
		--------------------
		Title:  The Prestige
		Year:   2006
		--------------------
		Title:  Insomnia
		Year:   2002
		--------------------
		Title:  Following
		Year:   1998
		--------------------
		Title:  Man of Steel
		Year:   2013
		--------------------
		Title:  Transcendence
		Year:   2014
		--------------------
		Title:  Justice League
		Year:   2017
		--------------------
		Title:  Batman v Superman: Dawn of Justice
		Year:   2016
		--------------------
		Title:  Ending the Knight
		Year:   2016
		--------------------
		Title:  Larceny
		--------------------
	*/
}

/*
 * Image utilties
 */

func urlToBase64(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(data), nil
}
```

</details>

<details>
<summary>Streaming with OpenAI</summary>

Running

```bash
export OPENAI_API_KEY=<Your OpenAI API Key>
go run examples/streaming/openai/main.go
```

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/instructor-ai/instructor-go/pkg/instructor"
	openai "github.com/sashabaranov/go-openai"
)

type Product struct {
	ID   string `json:"product_id"   jsonschema:"title=Product ID,description=ID of the product,required=True"`
	Name string `json:"product_name" jsonschema:"title=Product Name,description=Name of the product,required=True"`
}

func (p *Product) String() string {
	return fmt.Sprintf("Product [ID: %s, Name: %s]", p.ID, p.Name)
}

type Recommendation struct {
	Product
	Reason string `json:"reason" jsonschema:"title=Recommendation Reason,description=Reason for the product recommendation"`
}

func (r *Recommendation) String() string {
	return fmt.Sprintf(`
Recommendation [
    %s
    Reason [%s]
]`, r.Product.String(), r.Reason)
}

func main() {
	ctx := context.Background()

	client := instructor.FromOpenAI(
		openai.NewClient(os.Getenv("OPENAI_API_KEY")),
		instructor.WithMode(instructor.ModeJSON),
	)

	profileData := `
Customer ID: 12345
Recent Purchases: [Laptop, Wireless Headphones, Smart Watch]
Frequently Browsed Categories: [Electronics, Books, Fitness Equipment]
Product Ratings: {Laptop: 5 stars, Wireless Headphones: 4 stars}
Recent Search History: [best budget laptops 2023, latest sci-fi books, yoga mats]
Preferred Brands: [Apple, AllBirds, Bench]
Responses to Previous Recommendations: {Philips: Not Interested, Adidas: Not Interested}
Loyalty Program Status: Gold Member
Average Monthly Spend: $500
Preferred Shopping Times: Weekend Evenings
...
`

	products := []Product{
		{ID: "1", Name: "Sony WH-1000XM4 Wireless Headphones - Noise-canceling, long battery life"},
		{ID: "2", Name: "Apple Watch Series 7 - Advanced fitness tracking, seamless integration with Apple ecosystem"},
		{ID: "3", Name: "Kindle Oasis - Premium e-reader with adjustable warm light"},
		{ID: "4", Name: "AllBirds Wool Runners - Comfortable, eco-friendly sneakers"},
		{ID: "5", Name: "Manduka PRO Yoga Mat - High-quality, durable, eco-friendly"},
		{ID: "6", Name: "Bench Hooded Jacket - Stylish, durable, suitable for outdoor activities"},
		{ID: "7", Name: "Apple MacBook Air (2023) - Latest model, high performance, portable"},
		{ID: "8", Name: "GoPro HERO9 Black - 5K video, waterproof, for action photography"},
		{ID: "9", Name: "Nespresso Vertuo Next Coffee Machine - Quality coffee, easy to use, compact design"},
		{ID: "10", Name: "Project Hail Mary by Andy Weir - Latest sci-fi book from a renowned author"},
	}

	productList := ""
	for _, product := range products {
		productList += product.String() + "\n"
	}

	recommendationChan, err := client.CreateChatCompletionStream(ctx, openai.ChatCompletionRequest{
		Model: openai.GPT4o20240513,
		Messages: []openai.ChatCompletionMessage{
			{
				Role: openai.ChatMessageRoleSystem,
				Content: fmt.Sprintf(`
Generate the product recommendations from the product list based on the customer profile.
Return in order of highest recommended first.
Product list:
%s`, productList),
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: fmt.Sprintf("User profile:\n%s", profileData),
			},
		},
		Stream: true,
	},
		*new(Recommendation),
	)
	if err != nil {
		panic(err)
	}

	for instance := range recommendationChan {
		recommendation, _ := instance.(*Recommendation)
		println(recommendation.String())
	}
	/*
		Recommendation [
		    Product [ID: 7, Name: Apple MacBook Air (2023) - Latest model, high performance, portable]
		    Reason [As you have recently searched for budget laptops of 2023 and previously purchased a laptop, we believe the latest Apple MacBook Air will meet your high-performance requirements. Additionally, Apple is one of your preferred brands.]
		]

		Recommendation [
		    Product [ID: 2, Name: Apple Watch Series 7 - Advanced fitness tracking, seamless integration with Apple ecosystem]
		    Reason [Based on your recent purchase history which includes a smart watch and your preference for Apple products, we recommend the Apple Watch Series 7 for its advanced fitness tracking features.]
		]

		Recommendation [
		    Product [ID: 10, Name: Project Hail Mary by Andy Weir - Latest sci-fi book from a renowned author]
		    Reason [Given your recent search for the latest sci-fi books and frequent browsing in the Books category, 'Project Hail Mary' by Andy Weir may interest you.]
		]

		Recommendation [
		    Product [ID: 5, Name: Manduka PRO Yoga Mat - High-quality, durable, eco-friendly]
		    Reason [Since you recently searched for yoga mats and frequently browse fitness equipment, we recommend the Manduka PRO Yoga Mat to support your fitness activities.]
		]

		Recommendation [
		    Product [ID: 4, Name: AllBirds Wool Runners - Comfortable, eco-friendly sneakers]
		    Reason [Considering your preference for the AllBirds brand and your frequent browsing in fitness categories, the AllBirds Wool Runners would be a great fit for your lifestyle.]
		]
	*/
}
```

</details>

<details>
<summary>Document Segmentation with Cohere</summary>

Running

```bash
export COHERE_API_KEY=<Your Cohere API Key>
go run examples/document_segmentation/main.go
```

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	cohere "github.com/cohere-ai/cohere-go/v2"
	cohereclient "github.com/cohere-ai/cohere-go/v2/client"
	"github.com/instructor-ai/instructor-go/pkg/instructor"
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
		var structuredDoc StructuredDocument
		_, err := client.Chat(ctx, &cohere.ChatRequest{
			Model: toPtr("command-r-plus"),
			Preamble: toPtr(`
You are a world class educator working on organizing your lecture notes.
Read the document below and extract a StructuredDocument object from it where each section of the document is centered around a single concept/topic that can be taught in one lesson.
Each line of the document is marked with its line number in square brackets (e.g. [1], [2], [3], etc). Use the line numbers to indicate section start and end.
`),
			Message: docWithLines,
		},
			&structuredDoc,
		)
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
```

</details>

</details>

<details>
<summary>Streaming with Cohere</summary>

Running

```bash
export COHERE_API_KEY=<Your Cohere API Key>
go run examples/streaming/cohere/main.go
```

```go
package main

import (
	"context"
	"fmt"
	"os"

	cohere "github.com/cohere-ai/cohere-go/v2"
	cohereclient "github.com/cohere-ai/cohere-go/v2/client"
	"github.com/instructor-ai/instructor-go/pkg/instructor"
)

type HistoricalFact struct {
	Decade      string `json:"decade"       jsonschema:"title=Decade of the Fact,description=Decade when the fact occurred"`
	Topic       string `json:"topic"        jsonschema:"title=Topic of the Fact,description=General category or topic of the fact"`
	Description string `json:"description"  jsonschema:"title=Description of the Fact,description=Description or details of the fact"`
}

func (hf HistoricalFact) String() string {
	return fmt.Sprintf(`
Decade:         %s
Topic:          %s
Description:    %s`, hf.Decade, hf.Topic, hf.Description)
}

func main() {
	ctx := context.Background()

	client := instructor.FromCohere(
		cohereclient.NewClient(cohereclient.WithToken(os.Getenv("COHERE_API_KEY"))),
		instructor.WithMode(instructor.ModeJSON),
		instructor.WithMaxRetries(3),
	)

	hfStream, err := client.ChatStream(ctx, &cohere.ChatStreamRequest{
		Model:     toPtr("command-r-plus"),
		Message:   "Tell me about the history of artificial intelligence up to year 2000",
		MaxTokens: toPtr(2500),
	},
		*new(HistoricalFact),
	)
	if err != nil {
		panic(err)
	}

	for instance := range hfStream {
		hf := instance.(*HistoricalFact)
		println(hf.String())
	}
	/*
	   Decade:         1950s
	   Topic:          Birth of AI
	   Description:    The term 'Artificial Intelligence' is coined by John McCarthy at the Dartmouth Conference in 1956, considered the birth of AI as a field. Early research focuses on areas like problem solving, search algorithms, and logic.

	   Decade:         1960s
	   Topic:          Expert Systems and LISP
	   Description:    The language LISP is developed, which becomes widely used in AI applications. Research also leads to the development of expert systems, which emulate human decision-making abilities in specific domains.

	   Decade:         1970s
	   Topic:          AI Winter
	   Description:    AI experiences its first 'winter', a period of reduced funding and interest due to unmet expectations. Despite this, research continues in areas like knowledge representation and natural language processing.

	   Decade:         1980s
	   Topic:          Machine Learning and Neural Networks
	   Description:    The field of machine learning emerges, with a focus on developing algorithms that can learn from data. Neural networks, inspired by the structure of biological brains, gain traction during this decade.

	   Decade:         1990s
	   Topic:          AI in Practice
	   Description:    AI starts to find practical applications in various industries. Speech recognition, image processing, and expert systems are used in fields like healthcare, finance, and manufacturing.
	*/
}

func toPtr[T any](val T) *T {
	return &val
}
```

</details>

<details>
<summary>Local, Self-Hosted Models with Ollama (via OpenAI API Support)</summary>

Running

```bash
go run examples/ollama/main.go
```

```go
package main

import (
	"context"
	"fmt"

	"github.com/instructor-ai/instructor-go/pkg/instructor"
	openai "github.com/sashabaranov/go-openai"
)

type Character struct {
	Name string   `json:"name" jsonschema:"title=the name,description=The name of the character"`
	Age  int      `json:"age"  jsonschema:"title=the age,description=The age of the character"`
	Fact []string `json:"fact" jsonschema:"title=facts,description=A list of facts about the character"`
}

func (c *Character) String() string {
	facts := ""
	for i, fact := range c.Fact {
		facts += fmt.Sprintf("  %d. %s\n", i+1, fact)
	}
	return fmt.Sprintf(`
Name: %s
Age: %d
Facts:
%s
`,
		c.Name, c.Age, facts)
}

func main() {
	ctx := context.Background()

	config := openai.DefaultConfig("ollama")
	config.BaseURL = "http://localhost:11434/v1"

	client := instructor.FromOpenAI(
		openai.NewClientWithConfig(config),
		instructor.WithMode(instructor.ModeJSON),
		instructor.WithMaxRetries(3),
	)

	var character Character
	_, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: "llama3",
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: "Tell me about the Hal 9000",
			},
		},
	},
		&character,
	)
	if err != nil {
		panic(err)
	}

	println(character.String())
	/*
	   Name: Hal
	   Age: 0
	   Facts:
	     1. Viciously intelligent artificial intelligence system
	     2. Main computer on board Discovery One spacecraft
	     3. Killed David Bowman to preserve its own existence and maintain control of the ship
	     4. Famous line: 'Dave, stop. Stop. Will you stop? Dave?'
	*/
}
```
</details>

<details>

<summary>Receipt Item Extraction from Image (using OpenAI GPT-4o)</summary>

<p align="center">
  <img src="https://raw.githubusercontent.com/instructor-ai/instructor-go/main/examples/vision/receipt/supermarket-receipt-template.jpg" alt="Receipt 1" width="45%">
  <img src="https://raw.githubusercontent.com/instructor-ai/instructor-go/main/examples/vision/receipt/receipt-ocr-original.jpg" alt="Receipt 2" width="45%">
</p>

Running

```bash
go run examples/vision/receipt/main.go
```

```go
package main

import (
	"context"
	"fmt"
	"math"
	"os"

	"github.com/instructor-ai/instructor-go/pkg/instructor"
	openai "github.com/sashabaranov/go-openai"
)

type Item struct {
	Name     string  `json:"name"     jsonschema:"title=Item Name,description=The name of the item,example=Apple,example=Banana"`
	Price    float64 `json:"price"    jsonschema:"title=Item Price,description=The price of the item in dollars,example=1.99,example=2.50"`
}

func (i Item) String() string {
	return fmt.Sprintf("  Item: %s, Price: $%.2f", i.Name, i.Price)
}

type Receipt struct {
	Items []Item  `json:"items" jsonschema:"title=Receipt Items,description=The list of items in the receipt"`
	Total float64 `json:"total" jsonschema:"title=Receipt Total,description=The total cost of all items in the receipt,example=10.99,example=25.50"`
}

func (r Receipt) String() string {
	var result string
	for _, item := range r.Items {
		result += item.String() + "\n"
	}
	result += fmt.Sprintf("Total: $%.2f", r.Total)
	return result
}

func (r *Receipt) Validate() error {
	calculatedTotal := 0.0
	for _, item := range r.Items {
		calculatedTotal += item.Price
	}

	calculatedTotal = math.Round(calculatedTotal*10) / 10
	expectedTotal := math.Round(r.Total*10) / 10

	if calculatedTotal != expectedTotal {
		return fmt.Errorf("total %f does not match the sum of item prices %f", r.Total, calculatedTotal)
	}
	return nil
}

func extract(ctx context.Context, client *instructor.InstructorOpenAI, url string) (*Receipt, error) {

	var receipt Receipt
	_, err := client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model:       openai.GPT4o,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: `Analyze the image and return the items (include tax and coupons as their own items) in the receipt and the total amount.`,
				},
				{
					Role: openai.ChatMessageRoleUser,
					MultiContent: []openai.ChatMessagePart{
						{
							Type: openai.ChatMessagePartTypeImageURL,
							ImageURL: &openai.ChatMessageImageURL{
								URL: url,
							},
						},
					},
				},
			},
		},
		&receipt,
	)
	if err != nil {
		return nil, err
	}

	if err := receipt.Validate(); err != nil {
		return &receipt, err
	}

	return &receipt, nil
}

func main() {
	ctx := context.Background()

	client := instructor.FromOpenAI(
		openai.NewClient(os.Getenv("OPENAI_API_KEY")),
		instructor.WithMode(instructor.ModeJSON),
		instructor.WithMaxRetries(3),
	)

	urls := []string{
		"https://templates.mediamodifier.com/645124ff36ed2f5227cbf871/supermarket-receipt-template.jpg",
		"https://ocr.space/Content/Images/receipt-ocr-original.jpg",
	}

	for _, url := range urls {
		receipt, err := extract(ctx, client, url)
		fmt.Printf("Receipt:\n%s\n", receipt)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}
        fmt.Println("\n--------------------------------\n")
	}
/*
Receipt:
  Item: Lorem ipsum, Price: $9.20
  Item: Lorem ipsum dolor sit, Price: $19.20
  Item: Lorem ipsum dolor sit amet, Price: $15.00
  Item: Lorem ipsum, Price: $15.00
  Item: Lorem ipsum, Price: $15.00
  Item: Lorem ipsum dolor sit, Price: $15.00
  Item: Lorem ipsum, Price: $19.20
Total: $107.60

--------------------------------

Receipt:
  Item: PET TOY, Price: $1.97
  Item: FLOPPY PUPPY, Price: $1.97
  Item: SSSUPREME S, Price: $4.97
  Item: 2.5 SQUEAK, Price: $5.92
  Item: MUNCHY DMBEL, Price: $3.77
  Item: DOG TREAT, Price: $2.92
  Item: PED PCH 1, Price: $0.50
  Item: PED PCH 1, Price: $0.50
  Item: HNYMD SMORES, Price: $3.98
  Item: FRENCH DRSNG, Price: $1.98
  Item: 3 ORANGES, Price: $5.47
  Item: BABY CARROTS, Price: $1.48
  Item: COLLARDS, Price: $1.24
  Item: CALZONE, Price: $2.50
  Item: MM RVW MNT, Price: $19.77
  Item: STKOBRLPLIABL, Price: $1.97
  Item: STKOBRLPLIABL, Price: $1.97
  Item: STKO SUNFLWR, Price: $0.97
  Item: STKO SUNFLWR, Price: $0.97
  Item: STKO SUNFLWR, Price: $0.97
  Item: STKO SUNFLWR, Price: $0.97
  Item: BLING BEADS, Price: $0.97
  Item: GREAT VALUE, Price: $9.97
  Item: LIPTON, Price: $4.44
  Item: DRY DOG, Price: $12.44
  Item: COUPON 2310652, Price: $-1.00
  Item: TAX, Price: $4.59
Total: $98.21
*/
}
```

</details>

<details>

<summary>Task Ticket Creator from Transcript - OpenAI Structured Outputs (Strict JSON Mode)</summary>

Running

```bash
export OPENAI_API_KEY=<Your OpenAI API Key>
go run examples/auto_ticketer/main.go
```

```go
/*
 *  Original example in Python: https://github.com/jxnl/instructor/blob/11125a7c831a26e2a4deaef4129f2b4845a7e079/examples/auto-ticketer/run.py
 */

package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/instructor-ai/instructor-go/pkg/instructor"
	openai "github.com/sashabaranov/go-openai"
)

type PriorityEnum string

const (
	High   PriorityEnum = "High"
	Medium PriorityEnum = "Medium"
	Low    PriorityEnum = "Low"
)

type Subtask struct {
	ID   int    `json:"id"      jsonschema:"title=unique identifier for the subtask,description=Unique identifier for the subtask"`
	Name string `json:"name"    jsonschema:"title=name of the subtask,description=Informative title of the subtask"`
}

type Ticket struct {
	ID           int          `json:"id"                        jsonschema:"title=unique identifier for the ticket,description=Unique identifier for the ticket"`
	Name         string       `json:"name"                      jsonschema:"title=name of the task,description=Title of the task"`
	Description  string       `json:"description"               jsonschema:"title=description of the task,description=Detailed description of the task"`
	Priority     PriorityEnum `json:"priority"                  jsonschema:"title=priority level,description=Priority level"`
	Assignees    []string     `json:"assignees"                 jsonschema:"title=list of users assigned to the task,description=List of users assigned to the task"`
	Subtasks     []Subtask    `json:"subtasks"        jsonschema:"title=list of subtasks associated with the main task,description=List of subtasks associated with the main task"`
	Dependencies []int        `json:"dependencies"    jsonschema:"title=list of ticket IDs that this ticket depends on,description=List of ticket IDs that this ticket depends on"`
}

type ActionItems struct {
	Tickets []Ticket `json:"tickets"`
}

func (ai ActionItems) String() string {
	var sb strings.Builder

	for _, ticket := range ai.Tickets {
		sb.WriteString(fmt.Sprintf("Ticket ID: %d\n", ticket.ID))
		sb.WriteString(fmt.Sprintf("  Name: %s\n", ticket.Name))
		sb.WriteString(fmt.Sprintf("  Description: %s\n", ticket.Description))
		sb.WriteString(fmt.Sprintf("  Priority: %s\n", ticket.Priority))
		sb.WriteString(fmt.Sprintf("  Assignees: %s\n", strings.Join(ticket.Assignees, ", ")))

		if len(ticket.Subtasks) > 0 {
			sb.WriteString("  Subtasks:\n")
			for _, subtask := range ticket.Subtasks {
				sb.WriteString(fmt.Sprintf("    - Subtask ID: %d, Name: %s\n", subtask.ID, subtask.Name))
			}
		}

		if len(ticket.Dependencies) > 0 {
			sb.WriteString(fmt.Sprintf("  Dependencies: %v\n", ticket.Dependencies))
		}

		sb.WriteString("\n")
	}

	return sb.String()
}

func main() {
	ctx := context.Background()

	client := instructor.FromOpenAI(
		openai.NewClient(os.Getenv("OPENAI_API_KEY")),
		instructor.WithMode(instructor.ModeJSONStrict),
		instructor.WithMaxRetries(0),
	)

	transcript := `
Alice: Hey team, we have several critical tasks we need to tackle for the upcoming release. First, we need to work on improving the authentication system. It's a top priority.

Bob: Got it, Alice. I can take the lead on the authentication improvements. Are there any specific areas you want me to focus on?

Alice: Good question, Bob. We need both a front-end revamp and back-end optimization. So basically, two sub-tasks.

Carol: I can help with the front-end part of the authentication system.

Bob: Great, Carol. I'll handle the back-end optimization then.

Alice: Perfect. Now, after the authentication system is improved, we have to integrate it with our new billing system. That's a medium priority task.

Carol: Is the new billing system already in place?

Alice: No, it's actually another task. So it's a dependency for the integration task. Bob, can you also handle the billing system?

Bob: Sure, but I'll need to complete the back-end optimization of the authentication system first, so it's dependent on that.

Alice: Understood. Lastly, we also need to update our user documentation to reflect all these changes. It's a low-priority task but still important.

Carol: I can take that on once the front-end changes for the authentication system are done. So, it would be dependent on that.

Alice: Sounds like a plan. Let's get these tasks modeled out and get started.
`

	var actionItems ActionItems
	_, err := client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model:       openai.GPT4oMini20240718,
			Temperature: .2,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "The following is a transcript of a meeting between a manager and their team. The manager is assigning tasks to their team members and creating action items for them to complete.",
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: fmt.Sprintf("Create the action items for the following transcript: %s", transcript),
				},
			},
		},
		&actionItems,
	)
	if err != nil {
		panic(err)
	}

	println(actionItems.String())
	/*
		Ticket ID: 1
		  Name: Improve Authentication System
		  Description: Revamp the front-end and optimize the back-end of the authentication system.
		  Priority: high
		  Assignees: Bob, Carol
		  Subtasks:
		    - Subtask ID: 1, Name: Front-end Revamp
		    - Subtask ID: 2, Name: Back-end Optimization

		Ticket ID: 2
		  Name: Integrate Authentication with New Billing System
		  Description: Integrate the improved authentication system with the new billing system.
		  Priority: medium
		  Assignees: Bob
		  Dependencies: [1]

		Ticket ID: 3
		  Name: Update User Documentation
		  Description: Update the user documentation to reflect changes made to the authentication system.
		  Priority: low
		  Assignees: Carol
		  Dependencies: [1]
	*/
}
```

</details>

<summary>Gemini Multimodel Example</summary>

Running

```bash
export GOOGLE_API_KEY="your-api-key-here"
go run examples/gemini/main.go
```

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/instructor-ai/instructor-go/pkg/instructor"
	"google.golang.org/genai"
)

type ImageInfo struct {
	Description string   `json:"description" jsonschema:"title=Description,description=What is visible in the image"`
	Objects     []string `json:"objects" jsonschema:"title=Objects,description=List of objects detected in the image"`
	Colors      []string `json:"colors" jsonschema:"title=Colors,description=Main colors present in the image"`
	Mood        string   `json:"mood" jsonschema:"title=Mood,description=Overall mood of the image,enum=bright,enum=dark,enum=cheerful,enum=melancholic"`
}

func main() {
	ctx := context.Background()

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  os.Getenv("GOOGLE_API_KEY"),
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to create Gemini client: %v", err))
	}

	instructorClient := instructor.FromGemini(
		client,
		instructor.WithMode(instructor.ModeJSON),
		instructor.WithMaxRetries(3),
	)

	imageBytes, err := os.ReadFile("examples/gemini/sample.jpg")
	if err != nil {
		fmt.Printf("Error reading image: %v\n", err)
		fmt.Println("Please add a sample image at examples/gemini/sample.jpg")
		return
	}

	contents := []*genai.Content{
		{
			Parts: []*genai.Part{
				{Text: "Analyze this image and provide structured information about what you see."},
				{InlineData: &genai.Blob{
					Data:     imageBytes,
					MIMEType: "image/jpeg",
				}},
			},
			Role: "user",
		},
	}

	request := instructor.GeminiRequest{
		Model:    "gemini-2.5-flash",
		Contents: contents,
	}

	var imageInfo ImageInfo
	_, err = instructorClient.CreateChatCompletion(ctx, request, &imageInfo)
	if err != nil {
		fmt.Printf("Error analyzing image: %v\n", err)
		return
	}

	fmt.Println("📸 Image Analysis Results:")
	fmt.Printf("Description: %s\n", imageInfo.Description)
	fmt.Printf("Mood: %s\n", imageInfo.Mood)
	fmt.Printf("Objects: %v\n", imageInfo.Objects)
	fmt.Printf("Colors: %v\n", imageInfo.Colors)
	/*
		📸 Image Analysis Results:
		Description: The image features three cartoon gopher characters, the mascot for the Go programming language, arranged horizontally against a light blue background. Each gopher is light blue with large white eyes and black pupils, a small light brown or beige snout with visible teeth, and small limbs. The left gopher is angled to its right, showing one eye; the center gopher faces directly forward, showing both eyes; and the right gopher is angled to its left, showing one eye. The art style is simple and friendly with a slight textured or crayon-like appearance.
		Mood: cheerful
		Objects: [gopher cartoon character mascot]
		Colors: [light blue blue white black beige]
	*/
}

```

</details>

<details>
<summary>Union Types - Basic Usage</summary>

Union types allow the LLM to choose between multiple structured response types, perfect for handling different notification types.

Running

```bash
export OPENAI_API_KEY=<Your OpenAI API Key>
go run examples/union/main.go
```

See the full example at [`examples/union/main.go`](union/main.go)

Output:
```
Example 1: Extracting email notification
------------------------------------------------------------
Result type: EmailNotification
Email notification:
  To: john@example.com
  Subject: Meeting Tomorrow
  Body: Don't forget our meeting at 2pm
Tokens: 145
```

</details>

<details>
<summary>Agent Pattern with Union Types</summary>

Build agents that can select and execute different tools using union types. This example shows an agent that can search, lookup, and finish tasks.

Running

```bash
export OPENAI_API_KEY=<Your OpenAI API Key>
go run examples/agent/main.go
```

See the full example at [`examples/agent/main.go`](agent/main.go)

Output:
```
Goal: Search for information about Go programming language, then lookup details about interfaces, and provide a summary.
================================================================================

--- Turn 1 ---
Tokens used: 234 (input: 187, output: 47)
Tool: search("Go programming language")
Result: Search results for: Go programming language
- Result 1: Go is a programming language
- Result 2: Go has excellent concurrency support

--- Turn 2 ---
Tokens used: 321 (input: 298, output: 23)
Tool: lookup("interfaces")
Result: Details for 'interfaces':
Go's interface system allows for flexible and composable designs. Interfaces define behavior contracts that types can implement.

--- Turn 3 ---
Tokens used: 398 (input: 365, output: 33)
Tool: finish
Final Answer: Go is a modern programming language with strong concurrency support and a flexible interface system that enables composable design patterns.
================================================================================

Final Answer:
Go is a modern programming language with strong concurrency support and a flexible interface system that enables composable design patterns.
```

</details>
