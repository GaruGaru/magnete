package main

import (
	"github.com/gin-gonic/gin"
	"github.com/GaruGaru/magnete/providers"
	"os"
	"time"
	"fmt"
)

func main() {

	var timeoutDuration = getTimeoutEnv()

	var provider = providers.NewTorrentz("https://torrentz2.eu", timeoutDuration)

	r := gin.Default()

	r.GET("/probe", func(context *gin.Context) {
		context.String(200, "OK")
	})

	r.GET("/magnete", func(c *gin.Context) {

		query := c.Query("q")

		if query == "" {
			c.JSON(400, gin.H{
				"message": "error",
				"reason":  "Missing query param 'q'",
			})
			return
		}

		var results = provider.Get(query)
		c.JSON(200, results)

	})
	r.Run()

}
func getTimeoutEnv() time.Duration {
	var timeout = os.Getenv("TIMEOUT")
	if timeout != "" {
		var envTimeout, err = time.ParseDuration(timeout)
		if err == nil {
			fmt.Printf("Got timeout duration %s", envTimeout)
			return envTimeout
		} else {
			panic(err)
		}
	}
	return 5 * time.Second
}
