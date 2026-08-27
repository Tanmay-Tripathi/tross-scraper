package awssqs

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"

	"github.com/Tanmay-Tripathi/tross-scraper/pkg/global"
)

const (
	defaultWorkers     = 5
	defaultWaitTime    = 20 // seconds of long polling
	defaultMaxMessages = 10
	receiveBackoff     = time.Second
)

// Receiver runs a bounded pool of workers that long-poll one queue and hand
// each message to a handler. Messages are deleted only after the handler
// succeeds, so a failure leaves the message for redelivery (and eventually the
// dead-letter queue, when one is configured).
type Receiver struct {
	client      *Client
	queueURL    string
	handler     MessageHandler
	rawHandler  RawMessageHandler
	maxWorkers  int
	waitTime    int32
	maxMessages int32

	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	errorChan chan *QueueError
	stopOnce  sync.Once
}

// NewReceiver builds a receiver for SNS-enveloped messages on queueURL. The
// receiver stops when ctx is cancelled or Stop is called.
func (c *Client) NewReceiver(ctx context.Context, queueURL string, handler MessageHandler) *Receiver {
	receiverCtx, cancel := context.WithCancel(ctx)
	return &Receiver{
		client:      c,
		queueURL:    queueURL,
		handler:     handler,
		maxWorkers:  defaultWorkers,
		waitTime:    defaultWaitTime,
		maxMessages: defaultMaxMessages,
		ctx:         receiverCtx,
		cancel:      cancel,
		errorChan:   make(chan *QueueError, defaultMaxMessages),
	}
}

// WithWorkers sets the number of concurrent polling workers.
func (r *Receiver) WithWorkers(count int) *Receiver {
	if count > 0 {
		r.maxWorkers = count
	}
	return r
}

// WithWaitTime sets the long-polling wait, in seconds.
func (r *Receiver) WithWaitTime(seconds int32) *Receiver {
	if seconds > 0 {
		r.waitTime = seconds
	}
	return r
}

// WithMaxMessages sets how many messages a single poll may return.
func (r *Receiver) WithMaxMessages(count int32) *Receiver {
	if count > 0 {
		r.maxMessages = count
	}
	return r
}

// WithRawHandler switches the receiver to raw SQS bodies, skipping SNS
// envelope parsing. It replaces any handler passed to NewReceiver.
func (r *Receiver) WithRawHandler(handler RawMessageHandler) *Receiver {
	r.rawHandler = handler
	r.handler = nil
	return r
}

// Start launches the worker pool and returns the channel on which processing
// errors are reported. The channel is closed once Stop has drained the pool.
func (r *Receiver) Start() <-chan *QueueError {
	r.wg.Add(r.maxWorkers)
	for range r.maxWorkers {
		go r.worker()
	}
	return r.errorChan
}

// Stop cancels the workers, waits for them to finish and closes the error
// channel. It is safe to call more than once.
func (r *Receiver) Stop() {
	r.stopOnce.Do(func() {
		r.cancel()
		r.wg.Wait()
		close(r.errorChan)
	})
}

func (r *Receiver) worker() {
	defer r.wg.Done()

	for {
		if r.ctx.Err() != nil {
			return
		}

		messages, err := r.client.ReceiveMessage(r.ctx, r.queueURL, r.maxMessages, r.waitTime)
		if err != nil {
			if errors.Is(err, context.Canceled) || r.ctx.Err() != nil {
				return
			}
			r.report(&QueueError{Err: err, ErrCode: ReceiveError, Message: "receive error"})

			select {
			case <-r.ctx.Done():
				return
			case <-time.After(receiveBackoff):
			}
			continue
		}

		for _, message := range messages {
			r.process(message)
		}
	}
}

func (r *Receiver) process(message sqstypes.Message) {
	body := aws.ToString(message.Body)

	if r.rawHandler != nil {
		if err := r.rawHandler(r.ctx, body, message.MessageId); err != nil {
			r.report(&QueueError{Err: err, ErrCode: HandlerError, Message: "handler error", MessageID: message.MessageId})
			return
		}
		r.acknowledge(r.ctx, message, nil)
		return
	}

	var parsed SQSMessage
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		r.report(&QueueError{Err: err, ErrCode: UnmarshalError, Message: "unmarshal error", MessageID: message.MessageId})
		return
	}

	msgCtx := traceContext(r.ctx, parsed)
	if err := r.handler(msgCtx, parsed); err != nil {
		r.report(&QueueError{Err: err, ErrCode: HandlerError, Message: "handler error", MessageID: message.MessageId, Payload: &parsed})
		return
	}

	r.acknowledge(msgCtx, message, &parsed)
}

func (r *Receiver) acknowledge(ctx context.Context, message sqstypes.Message, payload *SQSMessage) {
	if err := r.client.DeleteMessage(ctx, r.queueURL, aws.ToString(message.ReceiptHandle)); err != nil {
		r.report(&QueueError{Err: err, ErrCode: DeleteError, Message: "delete error", MessageID: message.MessageId, Payload: payload})
	}
}

// traceContext lifts the request and correlation IDs the publisher attached
// back onto the context so queue work stays traceable end to end.
func traceContext(ctx context.Context, message SQSMessage) context.Context {
	if attr, ok := message.MessageAttributes[global.CorrelationID.String()]; ok {
		ctx = global.WithCorrelationID(ctx, attr.Value)
	}
	if attr, ok := message.MessageAttributes[global.RequestID.String()]; ok {
		ctx = global.WithRequestID(ctx, attr.Value)
	}
	return ctx
}

// report emits err without blocking a worker when nobody is draining the
// channel; the receive loop must keep running even if the consumer stalls.
func (r *Receiver) report(err *QueueError) {
	select {
	case r.errorChan <- err:
	default:
	}
}
