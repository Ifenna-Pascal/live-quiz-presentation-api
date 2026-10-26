package v1

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ifenna-pascal/interactive-presentation/internal/handler/respond"
	"github.com/ifenna-pascal/interactive-presentation/internal/module/realtime"
	"github.com/sirupsen/logrus"
)

// keepAlive is how often a comment line is sent so proxies keep the stream open.
const keepAlive = 15 * time.Second

type RealtimeHandler interface {
	StreamHandler(c *gin.Context)
}

type realtimeHandler struct {
	logger  *logrus.Logger
	useCase realtime.Usecase
}

func NewRealtimeHandler(logger *logrus.Logger, useCase realtime.Usecase) RealtimeHandler {
	return realtimeHandler{logger: logger, useCase: useCase}
}

// StreamHandler godoc
//
//	@Summary		Stream live updates (Server-Sent Events)
//	@Description	Open with the browser's EventSource. Event names: `ready`, `presentation`, `players`, `vote` (data: {"poll_id"}), `chat` (data: a MessageResponse), `deleted`, `resync`. Except for chat, an event means "fetch this again".
//	@Tags			realtime
//	@Produce		text/event-stream
//	@Param			presentation_id	path	string	true	"Presentation ID"	Format(uuid)
//	@Success		200				"An endless event stream"
//	@Failure		400				{object}	model.ErrorResponse	"Invalid id"
//	@Failure		404				{object}	model.ErrorResponse	"Presentation not found"
//	@Router			/presentations/{presentation_id}/events [get]
func (server realtimeHandler) StreamHandler(c *gin.Context) {
	presentationID, ok := respond.UUIDParam(c, "presentation_id")
	if !ok {
		return
	}
	stream, cancel, err := server.useCase.Subscribe(c.Request.Context(), presentationID)
	if err != nil {
		respond.Error(c, server.logger, err)
		return
	}
	defer cancel()

	header := c.Writer.Header()
	header.Set("Content-Type", "text/event-stream")
	header.Set("Cache-Control", "no-cache")
	header.Set("Connection", "keep-alive")
	header.Set("X-Accel-Buffering", "no") // nginx: do not buffer
	c.Status(http.StatusOK)

	w := c.Writer
	// Ask the browser to wait 2s before reconnecting after a drop.
	fmt.Fprint(w, "retry: 2000\n\n")
	writeEvent(w, "ready", nil)
	w.Flush()

	ping := time.NewTicker(keepAlive)
	defer ping.Stop()
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-ping.C:
			fmt.Fprint(w, ": ping\n\n")
			w.Flush()
		case e, open := <-stream:
			if !open {
				return
			}
			writeEvent(w, e.Type, e.Data)
			w.Flush()
		}
	}
}

func writeEvent(w gin.ResponseWriter, eventType string, data json.RawMessage) {
	if len(data) == 0 {
		data = json.RawMessage("{}")
	}
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, data)
}
