package test

import (
	"context"

	"firebase.google.com/go/v4/messaging"
)

// FirebaseMock mocks the firebase service
type FirebaseMock struct{}

// Send simulates sending a notification to a third party service.
func (f *FirebaseMock) Send(ctx context.Context, message *messaging.Message) (string, error) {
	return "", nil
}

// SendMulticast simulates sending multiple notifications in a single request to a third party service.
func (f *FirebaseMock) SendMulticast(ctx context.Context, messages []*messaging.MulticastMessage) (*messaging.BatchResponse, error) {
	return nil, nil
}

// SendAll simulates sending many notifications in a single request to a third party service.
func (f *FirebaseMock) SendAll(ctx context.Context, messages []*messaging.Message) (*messaging.BatchResponse, error) {
	return nil, nil
}

// SubscribeToTopic simulates subscribing to a firebase topic.
func (f *FirebaseMock) SubscribeToTopic(ctx context.Context, tokens []string, topic string) (*messaging.TopicManagementResponse, error) {
	return nil, nil
}

// UnsubscribeFromTopic simulates unsubscribing from a firebase topic.
func (f *FirebaseMock) UnsubscribeFromTopic(ctx context.Context, tokens []string, topic string) (*messaging.TopicManagementResponse, error) {
	return nil, nil
}
