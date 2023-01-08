package caramelmail

import (
	"github.com/adjust/rmq/v5"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"net/http"
	"time"
)

var (
	e *echo.Echo

	connection rmq.Connection

	singleQueue rmq.Queue
	bulkQueue   rmq.Queue
)

func Run() {
	errChan := make(chan error, 1)
	var err error

	connection, err = rmq.OpenConnection("rmq", "tcp", "localhost:6379", 1, errChan)
	if err != nil {
		panic(err)
	}

	singleQueue, _ = connection.OpenQueue("singleQueue")
	bulkQueue, _ = connection.OpenQueue("bulkQueue")
	_ = singleQueue.StartConsuming(10, time.Second)
	_ = bulkQueue.StartConsuming(10, time.Second)

	e = echo.New()
	e.Logger.SetLevel(log.DEBUG)
	e.Pre(middleware.RemoveTrailingSlash())
	e.Use(middleware.Logger())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
		AllowMethods: []string{echo.GET, echo.HEAD, echo.PUT, echo.PATCH, echo.POST, echo.DELETE},
	}))
	e.GET("/", index)

	e.POST("/send/single", addSingleMail)
	e.POST("/send/bulk", addBulkMail)

	e.Logger.Fatal(e.Start(":8080"))
}

func index(c echo.Context) error {
	return c.String(http.StatusOK, "Hello, World!")
}
