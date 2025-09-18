package meters

import (
	"net/http"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/server/auth/authn"
)

// WithMeter is a middleware function that wraps an http.Handler to provide metering functionality.
// It checks the rate limits for a given meter and records events if the request is successful.
// It uses the provided MeterProvider to create a Meterer for the current user context.
// The meterName is the name of the meter, meterValue is the value to be recorded,
// and dims are the dimensions associated with the meter event.
// If the rate limit is exceeded, it responds with a 429 Too Many Requests status code.
// If the request is successful (status code < 400), it meters the event using the Meterer.
func WithMeter(apiMeter MeterProvider, meterName string, meterValue float64, dims Dimensions) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Wrap ResponseWriter so we can check status code
			wr := &responseWriterWrapper{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			// Make ctxMeter available in context
			ctx := r.Context()
			ctxUser := authn.ForContext(ctx)
			meterLog := log.With().
				Str("user", ctxUser.ID()).
				Str("meter", meterName).
				Float64("meter_value", meterValue).
				Logger()

			ctxMeter := apiMeter.NewMeter(ctxUser)
			ctx = InjectContext(ctx, ctxMeter)
			r = r.WithContext(ctx)
			// Check if we are within available rate limits
			meterCheck, meterErr := ctxMeter.Check(ctx, meterName, meterValue, dims)
			if meterErr != nil {
				meterLog.Error().Err(meterErr).Msg("meter check error")
			}
			if !meterCheck {
				meterLog.Debug().Msg("not metering event due to rate limit 429")
				http.Error(w, "429", http.StatusTooManyRequests)
				return
			}
			// Call next handler
			next.ServeHTTP(wr, r)

			// Create a new MeterEvent with the current time in UTC
			event := NewMeterEvent(meterName, meterValue, dims)
			event.RequestID = log.GetReqID(r.Context())
			event.StatusCode = wr.statusCode
			event.Success = wr.statusCode < 400
			// Fetch meterer again from context
			if err := ForContext(ctx).Meter(ctx, event); err != nil {
				meterLog.Error().Err(err).Msg("failed to meter event")
			}
		})
	}
}

type responseWriterWrapper struct {
	statusCode int
	http.ResponseWriter
}

func (w *responseWriterWrapper) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}
