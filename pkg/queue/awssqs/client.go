package awssqs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	snstypes "github.com/aws/aws-sdk-go-v2/service/sns/types"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"

	"github.com/Tanmay-Tripathi/tross-scraper/pkg/global"
	"github.com/Tanmay-Tripathi/tross-scraper/pkg/log"
)

const dlqSuffix = "-dlq"

// Config describes how to reach SQS and SNS.
type Config struct {
	Region string
	// Endpoint overrides the AWS endpoint. Point it at a LocalStack instance
	// (http://localhost:4566) for local development; leave it empty to use the
	// real AWS endpoints resolved from the ambient credentials.
	Endpoint string
}

// Client wraps the SQS and SNS SDK clients.
type Client struct {
	sqs    *sqs.Client
	sns    *sns.Client
	region string
	logger log.Logger
}

// NewClient builds the SQS/SNS client pair from the ambient AWS configuration.
func NewClient(ctx context.Context, cfg Config, logger log.Logger) (*Client, error) {
	if strings.TrimSpace(cfg.Region) == "" {
		return nil, errors.New("aws region is empty")
	}

	loadOptions := []func(*awsconfig.LoadOptions) error{awsconfig.WithRegion(cfg.Region)}
	if cfg.Endpoint != "" {
		// A custom endpoint means LocalStack, which accepts any credentials.
		loadOptions = append(loadOptions, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider("test", "test", "test"),
		))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, loadOptions...)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := &Client{region: cfg.Region, logger: logger}
	if cfg.Endpoint != "" {
		logger.Infof("using custom aws endpoint %s", cfg.Endpoint)
		client.sqs = sqs.NewFromConfig(awsCfg, func(o *sqs.Options) { o.BaseEndpoint = aws.String(cfg.Endpoint) })
		client.sns = sns.NewFromConfig(awsCfg, func(o *sns.Options) { o.BaseEndpoint = aws.String(cfg.Endpoint) })
	} else {
		client.sqs = sqs.NewFromConfig(awsCfg)
		client.sns = sns.NewFromConfig(awsCfg)
	}

	return client, nil
}

// CreateTopic returns the ARN of topicName, creating the topic if needed.
// SNS CreateTopic is idempotent, so an existing topic is simply returned.
func (c *Client) CreateTopic(ctx context.Context, topicName string) (string, error) {
	result, err := c.sns.CreateTopic(ctx, &sns.CreateTopicInput{Name: aws.String(topicName)})
	if err != nil {
		return "", fmt.Errorf("create topic %q: %w", topicName, err)
	}
	return aws.ToString(result.TopicArn), nil
}

// CreateQueue returns queueName, creating it when absent. When maxRetryCount is
// positive and dlq is set, a redrive policy sends repeatedly failing messages
// to the dead-letter queue.
func (c *Client) CreateQueue(ctx context.Context, queueName string, queueType QueueType, maxRetryCount int32, dlq *Queue) (*Queue, error) {
	finalName := normalizeQueueName(queueName, queueType)

	attributes := map[string]string{
		string(sqstypes.QueueAttributeNameVisibilityTimeout):      "30",
		string(sqstypes.QueueAttributeNameMessageRetentionPeriod): "1209600", // 14 days
	}

	if queueType == QueueTypeFIFO {
		attributes[string(sqstypes.QueueAttributeNameFifoQueue)] = "true"
		attributes[string(sqstypes.QueueAttributeNameContentBasedDeduplication)] = "true"
	}

	if maxRetryCount > 0 && dlq != nil {
		policy, err := json.Marshal(map[string]string{
			"deadLetterTargetArn": dlq.ARN,
			"maxReceiveCount":     fmt.Sprintf("%d", maxRetryCount),
		})
		if err != nil {
			return nil, fmt.Errorf("marshal redrive policy for %q: %w", finalName, err)
		}
		attributes[string(sqstypes.QueueAttributeNameRedrivePolicy)] = string(policy)
	}

	return c.ensureQueue(ctx, finalName, attributes)
}

// CreateDlQueue provisions the dead-letter queue paired with queueName.
func (c *Client) CreateDlQueue(ctx context.Context, queueName string, queueType QueueType) (*Queue, error) {
	name := normalizeQueueName(strings.TrimSuffix(queueName, dlqSuffix)+dlqSuffix, queueType)
	return c.ensureQueue(ctx, name, nil)
}

// ensureQueue creates the queue when it does not exist and resolves its ARN.
// SQS CreateQueue is idempotent for identical attributes, so this is safe to
// call on every boot.
func (c *Client) ensureQueue(ctx context.Context, name string, attributes map[string]string) (*Queue, error) {
	existing, err := c.sqs.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{QueueName: aws.String(name)})
	if err == nil {
		return c.describeQueue(ctx, name, aws.ToString(existing.QueueUrl))
	}

	var notFound *sqstypes.QueueDoesNotExist
	if !errors.As(err, &notFound) {
		return nil, fmt.Errorf("look up queue %q: %w", name, err)
	}

	c.logger.Infof("creating sqs queue %s", name)
	created, err := c.sqs.CreateQueue(ctx, &sqs.CreateQueueInput{
		QueueName:  aws.String(name),
		Attributes: attributes,
	})
	if err != nil {
		return nil, fmt.Errorf("create queue %q: %w", name, err)
	}

	return c.describeQueue(ctx, name, aws.ToString(created.QueueUrl))
}

func (c *Client) describeQueue(ctx context.Context, name, queueURL string) (*Queue, error) {
	attrs, err := c.sqs.GetQueueAttributes(ctx, &sqs.GetQueueAttributesInput{
		QueueUrl:       aws.String(queueURL),
		AttributeNames: []sqstypes.QueueAttributeName{sqstypes.QueueAttributeNameQueueArn},
	})
	if err != nil {
		return nil, fmt.Errorf("get attributes for queue %q: %w", name, err)
	}

	return &Queue{
		Name: name,
		URL:  queueURL,
		ARN:  attrs.Attributes[string(sqstypes.QueueAttributeNameQueueArn)],
	}, nil
}

// SubscribeQueue allows topicARN to publish into queue and subscribes it.
func (c *Client) SubscribeQueue(ctx context.Context, topicARN string, queue *Queue) (string, error) {
	policy := fmt.Sprintf(`{
		"Version": "2012-10-17",
		"Statement": [{
			"Effect": "Allow",
			"Principal": {"AWS": "*"},
			"Action": "sqs:SendMessage",
			"Resource": %q,
			"Condition": {"ArnEquals": {"aws:SourceArn": %q}}
		}]
	}`, queue.ARN, topicARN)

	_, err := c.sqs.SetQueueAttributes(ctx, &sqs.SetQueueAttributesInput{
		QueueUrl:   aws.String(queue.URL),
		Attributes: map[string]string{string(sqstypes.QueueAttributeNamePolicy): policy},
	})
	if err != nil {
		return "", fmt.Errorf("set policy on queue %q: %w", queue.Name, err)
	}

	result, err := c.sns.Subscribe(ctx, &sns.SubscribeInput{
		TopicArn: aws.String(topicARN),
		Protocol: aws.String("sqs"),
		Endpoint: aws.String(queue.ARN),
	})
	if err != nil {
		return "", fmt.Errorf("subscribe queue %q to topic: %w", queue.Name, err)
	}

	return aws.ToString(result.SubscriptionArn), nil
}

// PublishToTopic fans a message out to every queue subscribed to topicARN,
// carrying the current trace identifiers as message attributes.
func (c *Client) PublishToTopic(ctx context.Context, topicARN, payload string) error {
	_, err := c.sns.Publish(ctx, &sns.PublishInput{
		TopicArn: aws.String(topicARN),
		Message:  aws.String(payload),
		MessageAttributes: map[string]snstypes.MessageAttributeValue{
			global.CorrelationID.String(): stringAttribute(global.CorrelationIDFromContext(ctx)),
			global.RequestID.String():     stringAttribute(global.RequestIDFromContext(ctx)),
		},
	})
	if err != nil {
		return fmt.Errorf("publish to topic: %w", err)
	}
	return nil
}

// SendToQueue writes a message straight to a queue, optionally delayed.
func (c *Client) SendToQueue(ctx context.Context, queueURL, message string, delay time.Duration) error {
	_, err := c.sqs.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:     aws.String(queueURL),
		MessageBody:  aws.String(message),
		DelaySeconds: int32(delay.Seconds()),
	})
	if err != nil {
		return fmt.Errorf("send message to queue: %w", err)
	}
	return nil
}

// ReceiveMessage long-polls a queue for up to maxMessages messages.
func (c *Client) ReceiveMessage(ctx context.Context, queueURL string, maxMessages, waitTimeSeconds int32) ([]sqstypes.Message, error) {
	result, err := c.sqs.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:              aws.String(queueURL),
		MaxNumberOfMessages:   maxMessages,
		WaitTimeSeconds:       waitTimeSeconds,
		MessageAttributeNames: []string{"All"},
	})
	if err != nil {
		return nil, fmt.Errorf("receive messages: %w", err)
	}
	return result.Messages, nil
}

// DeleteMessage acknowledges a message so it is not redelivered.
func (c *Client) DeleteMessage(ctx context.Context, queueURL, receiptHandle string) error {
	_, err := c.sqs.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(queueURL),
		ReceiptHandle: aws.String(receiptHandle),
	})
	if err != nil {
		return fmt.Errorf("delete message: %w", err)
	}
	return nil
}

func normalizeQueueName(name string, queueType QueueType) string {
	if queueType == QueueTypeFIFO && !strings.HasSuffix(name, ".fifo") {
		return name + ".fifo"
	}
	return name
}

func stringAttribute(value string) snstypes.MessageAttributeValue {
	if value == "" {
		value = "unknown"
	}
	return snstypes.MessageAttributeValue{
		DataType:    aws.String("String"),
		StringValue: aws.String(value),
	}
}
