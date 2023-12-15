package consumer

import (
	"time"

	"github.com/cenkalti/backoff/v4"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"
)

func (r *rabbitmqConsumer) retryBackoff(msg amqp.Delivery, logger *zerolog.Logger) {
	r.backoffWg.Add(1)
	defer r.backoffWg.Done()

	// Get the current retry count
	attempts, ok := msg.Headers["x-delivery-count"]
	if !ok {
		attempts = 0
	}

	if attempts.(int) >= r.config.MaxRetries {
		// We should stop processing the message
		err := msg.Nack(false, false)
		if err != nil {
			logger.Error().Err(err).Msg("failed to put message in dead letter queue")
		}
		return
	}

	backOff := backoff.NewExponentialBackOff()
	backOff.InitialInterval = r.config.InitialInterval
	backOff.RandomizationFactor = r.config.RandomizationFactor
	backOff.Multiplier = r.config.Multiplier
	backOff.MaxInterval = r.config.MaxInterval

	interval := backOff.NextBackOff()
	for i := 0; i < attempts.(int); i++ {
		backOff.NextBackOff()
	}

	if interval == backoff.Stop {
		// We should stop processing the message
		err := msg.Nack(false, false)
		if err != nil {
			logger.Error().Err(err).Msg("failed to put message in dead letter queue")
		}
		return
	}

	time.Sleep(interval)

	err := msg.Nack(false, true)
	if err != nil {
		logger.Error().Err(err).Msg("failed to requeue message")
	}
}