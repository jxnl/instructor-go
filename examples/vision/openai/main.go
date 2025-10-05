package main

import (
	"context"
	"fmt"
	"os"

	"github.com/instructor-ai/instructor-go/pkg/instructor"
	"github.com/instructor-ai/instructor-go/pkg/instructor/core"
	"github.com/instructor-ai/instructor-go/pkg/instructor/providers/openai"
	openaiLib "github.com/sashabaranov/go-openai"
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
		openaiLib.NewClient(os.Getenv("OPENAI_API_KEY")),
		instructor.WithMode(instructor.ModeJSON),
		instructor.WithMaxRetries(3),
	)

	url := "https://raw.githubusercontent.com/instructor-ai/instructor-go/main/examples/vision/openai/books.png"

	conversation := core.NewConversation()
	conversation.AddUserMessageWithImageURLs("Extract book catelog from the image", url)

	var bookCatalog BookCatalog
	_, err := client.CreateChatCompletion(ctx, openaiLib.ChatCompletionRequest{
		Model:    openaiLib.GPT4o,
		Messages: openai.ConversationToMessages(conversation),
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
