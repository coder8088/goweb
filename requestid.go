package goweb

import (
	"fmt"
	"context"

	"github.com/rs/xid"
	"go.opencensus.io/trace"
)

type RequestInfo struct {
	RootId string
	Id string
}

func (info *RequestInfo) String() string {
	return fmt.Sprintf("%s-%s", info.RootId, info.Id)
}

func GetRequestInfo(ctx context.Context) (info *RequestInfo) {
	info = &RequestInfo{}
	if span := trace.FromContext(ctx); span != nil {
		spanContext := span.SpanContext()
		info.RootId = spanContext.TraceID.String()
		info.Id = spanContext.SpanID.String()
	} else {
		id := xid.New().String()
		info.RootId = id
		info.Id = id
	}
	return
}
