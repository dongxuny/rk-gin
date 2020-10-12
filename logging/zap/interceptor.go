package rk_gin_inter_logging

import (
	"github.com/gin-gonic/gin"
	"github.com/rookie-ninja/rk-query"
	"go.uber.org/zap"
	"strconv"
	"strings"
	"time"
)

var RKEventFactory *rk_query.EventFactory

// RkGinZap returns a gin.HandlerFunc (middleware) that logs requests using uber-go/zap.
func RkGinZap(factory *rk_query.EventFactory, opts ...Option) gin.HandlerFunc {
	// We will populate Noop Zap logger if factory is nil
	if factory == nil {
		factory = rk_query.NewEventFactory()
	}

	RKEventFactory = factory

	opt := MergeOpt(opts)

	return func(ctx *gin.Context) {
		// start timer
		event := RKEventFactory.CreateEvent()
		event.SetStartTime(time.Now())
		ctx.Set(RKEventKey, event)

		incomingRequestIds := GetRequestIdsFromHeader(ctx.Request.Header)

		fields := []zap.Field{
			realm, region, az, domain, appVersion, localIP,
			zap.String("api.path", ctx.Request.URL.Path),
			zap.String("api.method", ctx.Request.Method),
			zap.String("api.query", ctx.Request.URL.RawQuery),
			zap.String("api.protocol", ctx.Request.Proto),
			zap.String("user_agent", ctx.Request.UserAgent()),
			zap.Strings("incoming_request_ids", incomingRequestIds),
			zap.Time("start_time", event.GetStartTime()),
		}

		remoteAddressSet := getRemoteAddressSet(ctx)
		fields = append(fields, remoteAddressSet...)
		event.SetRemoteAddr(remoteAddressSet[0].String + ":" + remoteAddressSet[1].String)
		event.SetOperation(ctx.Request.Method + "-" + ctx.Request.URL.Path)

		// handle rest of interceptors
		ctx.Next()

		endTime := time.Now()
		elapsed := endTime.Sub(event.GetStartTime())

		if opt.enableLogging() {
			// handle errors
			if len(ctx.Errors) > 0 {
				event.AddErr(ctx.Errors.Last().Err)
			}

			event.SetResCode(strconv.Itoa(ctx.Writer.Status()))
			outgoingRequestIds := GetRequestIdsFromHeader(ctx.Writer.Header())
			fields = append(fields,
				zap.Int("res_code", ctx.Writer.Status()),
				zap.Time("end_time", endTime),
				zap.Int64("elapsed_ms", elapsed.Milliseconds()),
				zap.Strings("outgoing_request_id", outgoingRequestIds),
			)

			event.AddFields(fields...)
			if len(event.GetEventId()) < 1 {
				ids := append(incomingRequestIds, outgoingRequestIds...)

				if len(ids) > 0 {
					event.SetEventId(strings.Join(ids, ","))
				}
			}

			event.SetEndTime(endTime)
			event.WriteLog()
		}

		if opt.enableMetrics() {
			getServerDurationMetrics(ctx).Observe(float64(elapsed.Nanoseconds() / 1e6))
			if len(ctx.Errors) > 0 {
				getServerErrorMetrics(ctx).Inc()
			}
			getServerResCodeMetrics(ctx).Inc()
		}
	}
}
