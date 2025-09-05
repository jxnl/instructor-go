package core

type Provider = string

const (
	ProviderOpenAI    Provider = "OpenAI"
	ProviderAnthropic Provider = "Anthropic"
	ProviderCohere    Provider = "Cohere"
	ProviderGoogle    Provider = "Google"
)
