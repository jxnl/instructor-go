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
	Name  string  `json:"name"     jsonschema:"title=Item Name,description=The name of the item,example=Apple,example=Banana"`
	Price float64 `json:"price"    jsonschema:"title=Item Price,description=The price of the item in dollars,example=1.99,example=2.50"`
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
			Model: openai.GPT4o,
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
		// source: https://templates.mediamodifier.com/645124ff36ed2f5227cbf871/supermarket-receipt-template.jpg
		"https://raw.githubusercontent.com/instructor-ai/instructor-go/main/examples/vision/receipt/supermarket-receipt-template.jpg",
		// source: https://ocr.space/Content/Images/receipt-ocr-original.jpg
		"https://raw.githubusercontent.com/instructor-ai/instructor-go/main/examples/vision/receipt/receipt-ocr-original.jpg",
	}

	for _, url := range urls {
		receipt, err := extract(ctx, client, url)
		fmt.Printf("Receipt:\n%s\n", receipt)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}
		fmt.Println()
		fmt.Println("--------------------------------")
		fmt.Println()
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
