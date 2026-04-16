package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jxnl/instructor-go/pkg/instructor"
	"github.com/jxnl/instructor-go/pkg/instructor/core"
	instructor_openai "github.com/jxnl/instructor-go/pkg/instructor/providers/openai"
	"github.com/sashabaranov/go-openai"
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

	conversation := core.NewConversation(fmt.Sprintf(`
Generate the product recommendations from the product list based on the customer profile.
Return in order of highest recommended first.
Product list:
%s`, productList))
	conversation.AddUserMessage(fmt.Sprintf("User profile:\n%s", profileData))

	recommendationChan, err := client.CreateChatCompletionStream(ctx, openai.ChatCompletionRequest{
		Model:    openai.GPT4o20240513,
		Messages: instructor_openai.ConversationToMessages(conversation),
		Stream:   true,
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
