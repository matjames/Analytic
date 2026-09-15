package events

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func TestDurablePublishFallsBackToInProcessTransport(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	producer, err := NewEventBus(Config{Source: "durable-producer"})
	if err != nil {
		t.Fatalf("producer bus: %v", err)
	}
	consumer, err := NewEventBus(Config{Source: "durable-consumer"})
	if err != nil {
		t.Fatalf("consumer bus: %v", err)
	}
	defer producer.Close()
	defer consumer.Close()

	received := make(chan EnterpriseEvent, 1)
	if err := consumer.SubscribeDurable(ctx, "durable-test-group", "durable-test-consumer", func(_ context.Context, event EnterpriseEvent) error {
		received <- event
		return nil
	}); err != nil {
		t.Fatalf("subscribe durable: %v", err)
	}
	time.Sleep(25 * time.Millisecond)

	if err := producer.PublishDurable(ctx, EnterpriseEvent{
		EventType:  EventSubmissionReceived,
		Source:     "durable-producer",
		ObjectType: "submission",
		ObjectID:   "submission-1",
		TenantID:   "tenant-1",
	}); err != nil {
		t.Fatalf("publish durable: %v", err)
	}

	select {
	case event := <-received:
		if event.EventID == "" || event.Version != "1.0" || event.TenantID != "tenant-1" {
			t.Fatalf("unexpected durable fallback event: %+v", event)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for durable fallback event")
	}
}

func TestDurablePendingMessageIsReclaimedAfterConsumerLoss(t *testing.T) {
	addr := os.Getenv("STATGATE_TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("set STATGATE_TEST_REDIS_ADDR to run Redis reclaim certification")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	stream := fmt.Sprintf("statgate:test:reclaim:%s", uuid.NewString())
	group := "reclaim-test-group"
	client := redis.NewClient(&redis.Options{Addr: addr})
	defer client.Close()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis unavailable: %v", err)
	}
	defer client.Del(context.Background(), stream)

	if err := client.XGroupCreateMkStream(ctx, stream, group, "$").Err(); err != nil {
		t.Fatalf("create group: %v", err)
	}
	producer, err := NewEventBus(Config{RedisAddr: addr, Source: "reclaim-producer", Stream: stream, Group: group})
	if err != nil {
		t.Fatalf("producer bus: %v", err)
	}
	defer producer.Close()

	if err := producer.PublishDurable(ctx, EnterpriseEvent{EventType: EventSubmissionReceived, ObjectType: "submission", ObjectID: "reclaim-1", TenantID: "tenant-1"}); err != nil {
		t.Fatalf("publish durable: %v", err)
	}
	claimed, err := client.XReadGroup(ctx, &redis.XReadGroupArgs{Group: group, Consumer: "crashed-consumer", Streams: []string{stream, ">"}, Count: 1, Block: time.Second}).Result()
	if err != nil || len(claimed) != 1 || len(claimed[0].Messages) != 1 {
		t.Fatalf("simulate crashed consumer claim: batches=%d err=%v", len(claimed), err)
	}

	consumer, err := NewEventBus(Config{RedisAddr: addr, Source: "reclaim-consumer", Stream: stream, Group: group, RetryDelay: 25 * time.Millisecond})
	if err != nil {
		t.Fatalf("consumer bus: %v", err)
	}
	defer consumer.Close()
	received := make(chan EnterpriseEvent, 1)
	if err := consumer.SubscribeDurable(ctx, group, "reclaimer", func(_ context.Context, event EnterpriseEvent) error {
		received <- event
		return nil
	}); err != nil {
		t.Fatalf("subscribe durable: %v", err)
	}

	select {
	case event := <-received:
		if event.ObjectID != "reclaim-1" {
			t.Fatalf("unexpected reclaimed event: %+v", event)
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for pending message reclaim")
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		pending, err := client.XPending(ctx, stream, group).Result()
		if err != nil {
			t.Fatalf("read pending state: %v", err)
		}
		if pending.Count == 0 {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatal("expected reclaimed message to be acknowledged before timeout")
}
