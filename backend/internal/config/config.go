package config

import "github.com/spf13/viper"

type Config struct {
	Port          string
	GitHubToken   string
	OpenAIAPIKey  string
	OpenAIModel   string
	OpenAIBaseURL string
}

func Load() (Config, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	v.SetDefault("PORT", "8080")
	v.SetDefault("OPENAI_BASE_URL", "https://api.openai.com/v1")
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return Config{}, err
		}
	}

	return Config{
		Port:          v.GetString("PORT"),
		GitHubToken:   v.GetString("GITHUB_TOKEN"),
		OpenAIAPIKey:  v.GetString("OPENAI_API_KEY"),
		OpenAIModel:   v.GetString("OPENAI_MODEL"),
		OpenAIBaseURL: v.GetString("OPENAI_BASE_URL"),
	}, nil
}
