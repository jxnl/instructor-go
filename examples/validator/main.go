package main

import (
	"context"
	"fmt"
	"os"

	"github.com/instructor-ai/instructor-go/pkg/instructor"
	"github.com/instructor-ai/instructor-go/pkg/instructor/core"
	instructor_openai "github.com/instructor-ai/instructor-go/pkg/instructor/providers/openai"
	"github.com/sashabaranov/go-openai"
)

type User struct {
	FirstName      string     `json:"first_name"        jsonschema:"title=First Name,description=The first name of the user"            validate:"required"`
	LastName       string     `json:"last_name"         jsonschema:"title=Last Name,description=The last name of the user"              validate:"required"`
	Age            uint8      `json:"age"               jsonschema:"title=Age,description=The age of the user"                          validate:"gte=0,lte=130"`
	Email          string     `json:"email"             jsonschema:"title=Email,description=The email address of the user"              validate:"required,email"`
	Gender         string     `json:"gender"            jsonschema:"title=Gender,description=The gender of the user"                    validate:"oneof=male female prefer_not_to"`
	FavouriteColor string     `json:"favourite_color"   jsonschema:"title=Favourite Color,description=The favourite color of the user"  validate:"iscolor"`
	Addresses      []*Address `json:"addresses"         jsonschema:"title=Addresses,description=The addresses of the user"              validate:"required,dive,required"`
}

type Address struct {
	Street string `json:"street"    jsonschema:"title=Street,description=The street address"    validate:"required"`
	City   string `json:"city"      jsonschema:"title=City,description=The city"                validate:"required"`
	Planet string `json:"planet"    jsonschema:"title=Planet,description=The planet"            validate:"required"`
	Phone  string `json:"phone"     jsonschema:"title=Phone,description=The phone number"       validate:"required"`
}

func (u User) String() string {
	result := fmt.Sprintf("First Name: %s\nLast Name: %s\nAge: %d\nEmail: %s\nGender: %s\nFavourite Color: %s\nAddresses:\n",
		u.FirstName, u.LastName, u.Age, u.Email, u.Gender, u.FavouriteColor)
	for _, address := range u.Addresses {
		result += fmt.Sprintf("  %s\n", address)
	}
	return result
}

func (a Address) String() string {
	return fmt.Sprintf("Street: %s, City: %s, Planet: %s, Phone: %s", a.Street, a.City, a.Planet, a.Phone)
}

func main() {
	ctx := context.Background()

	client := instructor.FromOpenAI(
		openai.NewClient(os.Getenv("OPENAI_API_KEY")),
		instructor.WithMode(instructor.ModeJSON),
		instructor.WithMaxRetries(3),
		instructor.WithValidation(),
	)

	conversation := core.NewConversation()
	conversation.AddUserMessage("Meet Jane Doe: a 30-year-old adventurer who can be reached at janed@example.com. " +
		"Jane loves the vibrant hue of #FF5733. She resides in Metropolis at 456 Oak St, on the wonderful planet Earth. " +
		"To chat with her, dial (555) 555-1234. Jane also spends her weekends at her cottage located at 789 Pine St, " +
		"in Smallville, on the same planet. You can contact her there at (555) 555-5678.")

	var user User
	_, err := client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model:    openai.GPT4o,
			Messages: instructor_openai.ConversationToMessages(conversation),
		},
		&user,
	)
	if err != nil {
		panic(err)
	}

	fmt.Println(user)
	/*
		First Name: Jane
		Last Name: Doe
		Age: 30
		Email: janed@example.com
		Gender: female
		Favourite Color: #FF5733
		Addresses:
		  Street: 456 Oak St, City: Metropolis, Planet: Earth, Phone: (555) 555-1234
		  Street: 789 Pine St, City: Smallville, Planet: Earth, Phone: (555) 555-5678
	*/
}
