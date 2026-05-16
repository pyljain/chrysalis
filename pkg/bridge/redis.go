package bridge

import (
	"chrysalis/pkg/models"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	redisWorkQueue                      = "work"
	redisSessionNewMessageQueue         = "message"
	redisLogLineQueue                   = "logs-queue"
	redisHistoryUpdateNotificationTopic = "history-updates"
	redisStatusUpdateQueue              = "status-updates"
)

type redisBridge struct {
	conn *redis.Client
}

var _ Bridge = (*redisBridge)(nil)

func NewRedis(ctx context.Context, connectionString string) (*redisBridge, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     connectionString,
		Password: "",
		DB:       0,
	})

	err := rdb.Ping(ctx).Err()
	if err != nil {
		return nil, err
	}

	return &redisBridge{
		conn: rdb,
	}, nil
}

func (r *redisBridge) PublishWork(ctx context.Context, session *models.Session) error {
	return r.publishSession(ctx, redisWorkQueue, session)
}

func (r *redisBridge) publishSession(ctx context.Context, queue string, session *models.Session) error {
	sessionBytes, err := json.Marshal(&session)
	if err != nil {
		return err
	}
	res := r.conn.LPush(ctx, queue, string(sessionBytes))
	if res.Err() != nil {
		return res.Err()
	}

	return nil
}

func (r *redisBridge) SubscribeForWork(ctx context.Context) chan models.Session {
	ch := make(chan models.Session)
	go func() {
		for {
			res := r.conn.BRPop(ctx, 0, redisWorkQueue)
			if res.Err() != nil {
				log.Printf("Error received while reading from status queue %s", res.Err())
				continue
			}

			responses, err := res.Result()
			if err != nil {
				log.Printf("Unable to get a status update from the queue %s", err)
				continue
			}

			result := models.Session{}
			err = json.Unmarshal([]byte(responses[1]), &result)
			if err != nil {
				log.Printf("Unable to get a status update from the queue %s", err)
				break
			}
			ch <- result
		}
	}()

	return ch
}

func (r *redisBridge) WatchQueue(ctx context.Context, sessionId string) string {
	queueName := fmt.Sprintf("%s:%s", redisSessionNewMessageQueue, sessionId)
	messages := r.conn.BRPop(ctx, 1*time.Minute, queueName)
	if messages.Err() != nil {
		log.Printf("Could not watch queue %s", messages.Err())
		return ""
	}

	return messages.Val()[1]
}

func (r *redisBridge) DeleteQueue(ctx context.Context, sessionId string) error {
	queueName := fmt.Sprintf("%s:%s", redisSessionNewMessageQueue, sessionId)
	return r.conn.Del(ctx, queueName).Err()
}

func (r *redisBridge) SendMessage(ctx context.Context, session *models.Session) error {
	// If session is NOT inactive, then new messages should be added to the session specific queue
	if session.Status == models.SessionStatusInactive || session.Status == models.SessionStatusPending {
		err := r.PublishWork(ctx, session)
		if err != nil {
			return err
		}
	} else {
		queueName := fmt.Sprintf("%s:%s", redisSessionNewMessageQueue, session.Id)
		err := r.publishSession(ctx, queueName, session)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *redisBridge) RecordLogLine(ctx context.Context, sessionId string, logLine map[string]string) error {
	logLine["sessionId"] = sessionId

	dataBytes, err := json.Marshal(logLine)
	if err != nil {
		return err
	}

	err = r.conn.LPush(ctx, redisLogLineQueue, string(dataBytes)).Err()
	if err != nil {
		return err
	}

	return nil
}

func (r *redisBridge) TrackLogs(ctx context.Context) chan map[string]string {
	ch := make(chan map[string]string)
	go func() {
		for {
			res := r.conn.BRPop(ctx, 0, redisLogLineQueue)
			if res.Err() != nil {
				log.Printf("Error received while reading from log line queue %s", res.Err())
				continue
			}

			responses, err := res.Result()
			if err != nil {
				log.Printf("Unable to get an update from the log line queue %s", err)
				continue
			}

			result := map[string]string{}
			err = json.Unmarshal([]byte(responses[1]), &result)
			if err != nil {
				log.Printf("Unable to get a log line from the queue %s", err)
				break
			}
			ch <- result
		}
	}()

	return ch
}

func (r *redisBridge) PublishHistoryUpdateNotification(ctx context.Context, sessionId string) error {
	err := r.conn.Publish(ctx, redisHistoryUpdateNotificationTopic, sessionId).Err()
	if err != nil {
		return err
	}

	return nil
}

func (r *redisBridge) SubscribeToHistoryUpdateNotification(ctx context.Context) chan string {
	ch := make(chan string)

	go func() {
		sub := r.conn.Subscribe(ctx, redisHistoryUpdateNotificationTopic)

		for msg := range sub.Channel() {
			ch <- msg.Payload
		}
	}()

	return ch
}

func (r *redisBridge) RecordSessionStatusUpdate(ctx context.Context, sessionId string, status models.SessionStatus) error {
	dataBytes, err := json.Marshal(SessionStatusUpdate{
		SessionID: sessionId,
		Status:    status,
	})
	if err != nil {
		return err
	}

	err = r.conn.LPush(ctx, redisStatusUpdateQueue, string(dataBytes)).Err()
	if err != nil {
		return err
	}

	return nil
}

func (r *redisBridge) WatchForSessionStatusUpdates(ctx context.Context) chan SessionStatusUpdate {
	ch := make(chan SessionStatusUpdate)
	go func() {
		for {
			res := r.conn.BRPop(ctx, 0, redisStatusUpdateQueue)
			if res.Err() != nil {
				log.Printf("Error received while reading from log line queue %s", res.Err())
				continue
			}

			responses, err := res.Result()
			if err != nil {
				log.Printf("Unable to get an update from the log line queue %s", err)
				continue
			}

			result := SessionStatusUpdate{}
			err = json.Unmarshal([]byte(responses[1]), &result)
			if err != nil {
				log.Printf("Unable to get a log line from the queue %s", err)
				break
			}
			ch <- result
		}
	}()

	return ch
}
