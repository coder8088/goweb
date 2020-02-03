package goweb

import (
	"net/http"
	"sync"
	"bytes"
	"io/ioutil"
	"io"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/tedux/goweb/xerr"
)

const (
	requestId = "server.request_id"

	logFields         = "log.fields"
	logFieldRequestId = "request_id"
	logFieldCode      = "code"
	logFieldStatus    = "status"
	logFieldError     = "error"
)

func Gin(register func(gin.IRouter)) http.Handler {
	engine := gin.New()
	engine.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		Output: logrus.StandardLogger().WriterLevel(logrus.DebugLevel),
	}))
	engine.Use(gin.Recovery())
	engine.Use(generateRequestId())
	engine.Use(accessLog())

	register(engine)
	return engine
}

func generateRequestId() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		info := GetRequestInfo(ctx.Request.Context())
		ctx.Set(requestId, info.String())
		ctx.Next()
	}
}

func accessLog() gin.HandlerFunc {
	pool := sync.Pool{
		New: func() interface{} { return &bytes.Buffer{} },
	}

	return func(ctx *gin.Context) {
		buffer := pool.Get().(*bytes.Buffer)
		defer pool.Put(buffer)
		buffer.Reset()

		reader := ctx.Request.Body
		defer func() { _ = reader.Close() }()

		ctx.Request.Body = ioutil.NopCloser(io.TeeReader(reader, buffer))
		startTime := time.Now()

		ctx.Set(logFields, map[string]interface{}{
			logFieldRequestId: ctx.GetString(requestId),
			logFieldStatus:    xerr.CodeToStr(xerr.OK),
			logFieldCode:      http.StatusOK,
			logFieldError:     "",
		})

		ctx.Next()

		fields := ctx.GetStringMap(logFields)
		message := fmt.Sprintf("[%v] %v(%v) - %v %v - %v",
			time.Since(startTime).Truncate(time.Millisecond),
			fields[logFieldStatus],
			fields[logFieldCode],
			ctx.Request.Method,
			ctx.Request.URL,
			buffer.String())

		logger := logrus.WithFields(fields)
		if fields[logFieldCode] == http.StatusOK {
			logger.Info(message)
		} else {
			logger.Warn(message)
		}
	}
}

func RenderError(ctx *gin.Context, op xerr.Op, err error) {
	requestId := ctx.GetString(requestId)
	resp := xerr.Http(requestId, xerr.E(op, err))

	fields := ctx.GetStringMap(logFields)
	fields[logFieldStatus] = resp.Error.Status
	fields[logFieldCode] = resp.Error.Code
	fields[logFieldError] = resp.Error.Message

	ctx.JSON(resp.Error.Code, resp)
}

func NotFound(name, id string) error {
	return xerr.E(xerr.NotFound, &xerr.NotFoundError{Type: name, ID: id})
}
