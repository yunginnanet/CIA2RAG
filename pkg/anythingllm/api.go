package anythingllm

const (
	DefaultEndpoint = "http://localhost:3001/api"
)

type Config struct {
	Endpoint string
	Key      string
}

func NewConfig() *Config {
	return &Config{
		Endpoint: DefaultEndpoint,
	}
}

func (c *Config) WithEndpoint(endpoint string) *Config {
	c.Endpoint = endpoint
	return c
}

func (c *Config) WithAPIKey(key string) *Config {
	c.Key = key
	return c
}
