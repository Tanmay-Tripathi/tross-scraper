// Package awssqs is a thin wrapper over the AWS SQS and SNS SDKs: topic/queue
// provisioning, publishing, and a worker-pool receiver.
package awssqs

import "context"

// MessageHandler processes a message that arrived on a queue via an SNS topic.
type MessageHandler func(ctx context.Context, message SQSMessage) error

// RawMessageHandler processes a message body written directly to a queue,
// with no SNS envelope around it.
type RawMessageHandler func(ctx context.Context, body string, messageID *string) error

// Queue identifies a provisioned SQS queue.
type Queue struct {
	Name string
	URL  string
	ARN  string
}

// QueueType selects standard or FIFO delivery semantics.
type QueueType string

const (
	QueueTypeStandard QueueType = "standard"
	QueueTypeFIFO     QueueType = "fifo"
)

// MessageAttribute is one SNS message attribute as it appears in the envelope.
type MessageAttribute struct {
	Type  string `json:"Type"`
	Value string `json:"Value"`
}

// SQSMessage is the SNS notification envelope delivered to a subscribed queue.
type SQSMessage struct {
	Type              string                      `json:"Type"`
	MessageID         string                      `json:"MessageId"`
	TopicArn          string                      `json:"TopicArn"`
	Message           string                      `json:"Message"`
	Timestamp         string                      `json:"Timestamp"`
	MessageAttributes map[string]MessageAttribute `json:"MessageAttributes"`
}

// QueueErrorCode classifies where in the receive loop a failure happened.
type QueueErrorCode string

const (
	ReceiveError   QueueErrorCode = "RECEIVE_ERROR"
	UnmarshalError QueueErrorCode = "UNMARSHAL_ERROR"
	HandlerError   QueueErrorCode = "HANDLER_ERROR"
	DeleteError    QueueErrorCode = "DELETE_ERROR"
)

// QueueError is emitted on the receiver's error channel.
type QueueError struct {
	Err       error
	ErrCode   QueueErrorCode
	Message   string
	MessageID *string
	Payload   *SQSMessage
}

func (e *QueueError) Error() string {
	if e == nil || e.Err == nil {
		return string(e.ErrCode)
	}
	return string(e.ErrCode) + ": " + e.Err.Error()
}

func (e *QueueError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}
