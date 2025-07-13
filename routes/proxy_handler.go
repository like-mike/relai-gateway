package routes

import (
	"io"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/like-mike/relai-gateway/provider"
)

// ProxyHandler proxies any undefined HTTP route to the selected provider API.
func ProxyHandler(c *gin.Context) {
	method := c.Request.Method
	path := c.Request.URL.Path
	query := c.Request.URL.RawQuery
	target := path
	if query != "" {
		target += "?" + query
	}

	config := provider.NewProxyConfigFromEnv("openai")
	req, err := http.NewRequest(method, config.BaseURL+target, c.Request.Body)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to create proxy request")
		return
	}
	for k, v := range c.Request.Header {
		for _, vv := range v {
			req.Header.Add(k, vv)
		}
	}
	if config.AuthHeader != "" {
		if token := os.Getenv("LLM_API_KEY"); token != "" {
			req.Header.Set(config.AuthHeader, "Bearer "+token)
		}
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.String(http.StatusBadGateway, "failed to reach provider")
		return
	}
	defer resp.Body.Close()
	for hk, hv := range resp.Header {
		for _, v := range hv {
			c.Writer.Header().Add(hk, v)
		}
	}
	c.Status(resp.StatusCode)
	_, err = io.Copy(c.Writer, resp.Body)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to stream provider response")
		return
	}
}
