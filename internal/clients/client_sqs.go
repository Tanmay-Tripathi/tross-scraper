package clients

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Tanmay-Tripathi/tross-scraper/pkg/log"
	"github.com/Tanmay-Tripathi/tross-scraper/pkg/queue/awssqs"
)

// SubscriptionConfig describes a queue that should consume a topic.
type SubscriptionConfig struct {
	TopicName string
	QueueName string
	QueueType awssqs.QueueType

	// Workers, WaitTime and MaxMessages tune the receiver; zero values fall
	// back to the receiver's defaults.
	Workers     int
	WaitTime    int32
	MaxMessages int32

	// EnableAutoDlq provisions a dead-letter queue and attaches a redrive
	// policy once MaxRetryCount deliveries have failed.
	EnableAutoDlq bool
	MaxRetryCount int32

	// Handler receives SNS-enveloped messages. Exactly one of Handler and
	// RawHandler must be set.
	Handler awssqs.MessageHandler
	// RawHandler receives message bodies written straight to the queue.
	RawHandler awssqs.RawMessageHandler
	// OnError, when set, is called for every failure the receiver reports.
	OnError func(ctx context.Context, queueErr *awssqs.QueueError)
}

// ClientSqsMethods is the messaging surface services depend on. Queue and topic
// names are namespaced by environment, so a stage consumer can never read
// production traffic. Clients.Sqs is nil when messaging is disabled, so callers
// must check it before use.
type ClientSqsMethods interface {
	// Subscribe provisions the topic and queue, links them, and starts
	// consuming. It blocks only for the provisioning calls.
	Subscribe(ctx context.Context, cfg *SubscriptionConfig) error
	// Consume provisions a standalone queue and starts consuming it, without
	// attaching it to any topic.
	Consume(ctx context.Context, cfg *SubscriptionConfig) error
	// PublishToTopic fans a message out to every queue subscribed to a topic.
	PublishToTopic(ctx context.Context, topicName, message string) error
	// SendToQueue writes a message straight to one queue, optionally delayed.
	SendToQueue(ctx context.Context, queueName, message string, delay time.Duration) error
	// Close stops every receiver this client started.
	Close()
}

// ClientSqs owns the AWS client plus the receivers it has started.
type ClientSqs struct {
	client  *awssqs.Client
	logger  log.Logger
	envName string

	mu        sync.Mutex
	topicArns map[string]string
	receivers []*awssqs.Receiver
}

// NewClientSqs builds the messaging client. It returns (nil, nil) when
// messaging is disabled, which callers treat as "not configured" rather than a
// failure.
func NewClientSqs(ctx context.Context, access *clientAccess) (ClientSqsMethods, error) {
	cfg := access.cfg.SQS
	if !cfg.Enabled {
		access.logger.Infof("messaging is disabled, skipping sqs client setup")
		return nil, nil
	}

	client, err := awssqs.NewClient(ctx, awssqs.Config{
		Region:   cfg.Region,
		Endpoint: cfg.Endpoint,
	}, access.logger)
	if err != nil {
		return nil, err
	}

	return &ClientSqs{
		client:    client,
		logger:    access.logger,
		envName:   access.cfg.Environment,
		topicArns: make(map[string]string),
	}, nil
}

func (c *ClientSqs) Subscribe(ctx context.Context, cfg *SubscriptionConfig) error {
	logger := c.logger.With(ctx)
	queueName := c.scoped(cfg.QueueName)

	var dlq *awssqs.Queue
	if cfg.EnableAutoDlq {
		created, err := c.client.CreateDlQueue(ctx, queueName, cfg.QueueType)
		if err != nil {
			return fmt.Errorf("provision dead-letter queue for %q: %w", queueName, err)
		}
		dlq = created
	}

	topicArn, err := c.topicArn(ctx, c.scoped(cfg.TopicName))
	if err != nil {
		return err
	}

	queue, err := c.client.CreateQueue(ctx, queueName, cfg.QueueType, cfg.MaxRetryCount, dlq)
	if err != nil {
		return fmt.Errorf("provision queue %q: %w", queueName, err)
	}

	if _, err := c.client.SubscribeQueue(ctx, topicArn, queue); err != nil {
		return err
	}

	logger.Infof("subscribed queue %s to topic %s", queue.Name, cfg.TopicName)
	return c.startReceiver(ctx, queue, cfg)
}

func (c *ClientSqs) Consume(ctx context.Context, cfg *SubscriptionConfig) error {
	queueName := c.scoped(cfg.QueueName)
	queue, err := c.client.CreateQueue(ctx, queueName, cfg.QueueType, 0, nil)
	if err != nil {
		return fmt.Errorf("provision queue %q: %w", queueName, err)
	}

	return c.startReceiver(ctx, queue, cfg)
}

func (c *ClientSqs) startReceiver(ctx context.Context, queue *awssqs.Queue, cfg *SubscriptionConfig) error {
	if cfg.Handler == nil && cfg.RawHandler == nil {
		return fmt.Errorf("no handler configured for queue %q", queue.Name)
	}

	receiver := c.client.NewReceiver(ctx, queue.URL, cfg.Handler).
		WithWorkers(cfg.Workers).
		WithWaitTime(cfg.WaitTime).
		WithMaxMessages(cfg.MaxMessages)

	if cfg.RawHandler != nil {
		receiver = receiver.WithRawHandler(cfg.RawHandler)
	}

	errChan := receiver.Start()

	c.mu.Lock()
	c.receivers = append(c.receivers, receiver)
	c.mu.Unlock()

	go c.drainErrors(ctx, queue.Name, errChan, cfg.OnError)

	c.logger.Infof("listening on queue %s", queue.Name)
	return nil
}

// drainErrors keeps the receiver's error channel moving; a stalled consumer
// would otherwise cause the receiver to drop error reports on the floor.
func (c *ClientSqs) drainErrors(ctx context.Context, queueName string, errChan <-chan *awssqs.QueueError, onError func(context.Context, *awssqs.QueueError)) {
	for queueErr := range errChan {
		c.logger.Errorf("queue %s: %s: %v (message_id=%s)", queueName, queueErr.ErrCode, queueErr.Err, messageID(queueErr))
		if onError != nil {
			onError(ctx, queueErr)
		}
	}
}

func (c *ClientSqs) PublishToTopic(ctx context.Context, topicName, message string) error {
	topicArn, err := c.topicArn(ctx, c.scoped(topicName))
	if err != nil {
		return err
	}
	return c.client.PublishToTopic(ctx, topicArn, message)
}

func (c *ClientSqs) SendToQueue(ctx context.Context, queueName, message string, delay time.Duration) error {
	queue, err := c.client.CreateQueue(ctx, c.scoped(queueName), awssqs.QueueTypeStandard, 0, nil)
	if err != nil {
		return fmt.Errorf("resolve queue %q: %w", queueName, err)
	}
	return c.client.SendToQueue(ctx, queue.URL, message, delay)
}

func (c *ClientSqs) Close() {
	c.mu.Lock()
	receivers := c.receivers
	c.receivers = nil
	c.mu.Unlock()

	for _, receiver := range receivers {
		receiver.Stop()
	}
}

// topicArn resolves and memoises a topic's ARN so repeated publishes to the
// same topic do not hit SNS on every call.
func (c *ClientSqs) topicArn(ctx context.Context, topicName string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if arn, ok := c.topicArns[topicName]; ok {
		return arn, nil
	}

	arn, err := c.client.CreateTopic(ctx, topicName)
	if err != nil {
		return "", err
	}

	c.topicArns[topicName] = arn
	return arn, nil
}

// scoped namespaces a topic or queue name with the current environment.
func (c *ClientSqs) scoped(name string) string {
	return fmt.Sprintf("%s_%s", c.envName, name)
}

func messageID(queueErr *awssqs.QueueError) string {
	if queueErr.MessageID == nil {
		return "unknown"
	}
	return *queueErr.MessageID
}
