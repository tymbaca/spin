package kafka

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	kafkago "github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/plain"
)

const (
	defaultUsername = "admin"
	defaultPassword = "admin"
	defaultProtocol = "SASL_PLAINTEXT"
	defaultMechanism = "PLAIN"
	pollInterval    = 500 * time.Millisecond
)

type AuthConfig struct {
	Port      int
	Protocol  string
	Mechanism string
	User      string
	Password  string
}

func (cfg AuthConfig) withDefaults() AuthConfig {
	if cfg.Protocol == "" {
		cfg.Protocol = defaultProtocol
	}
	if cfg.Mechanism == "" {
		cfg.Mechanism = defaultMechanism
	}
	if cfg.User == "" {
		cfg.User = defaultUsername
	}
	if cfg.Password == "" {
		cfg.Password = defaultPassword
	}
	return cfg
}

func WaitForKafka(ctx context.Context, cfg AuthConfig) error {
	cfg = cfg.withDefaults()
	address := Address(cfg.Port)
	for {
		conn, err := dial(ctx, address, cfg)
		if err == nil {
			_, readErr := conn.ReadPartitions()
			_ = conn.Close()
			if readErr == nil {
				return nil
			}
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("wait for kafka: %w", ctx.Err())
		case <-time.After(pollInterval):
		}
	}
}

func CreateTopicsFromFile(ctx context.Context, cfg AuthConfig, path string) error {
	topics, err := readTopicList(path)
	if err != nil {
		return err
	}
	return createTopics(ctx, cfg, topics, path)
}

func CreateTopics(ctx context.Context, cfg AuthConfig, topics []string) error {
	return createTopics(ctx, cfg, topics, "command line")
}

func ParseTopics(s string) []string {
	seen := make(map[string]struct{})
	var topics []string
	for _, part := range strings.Split(s, ",") {
		topic := strings.TrimSpace(part)
		if topic == "" {
			continue
		}
		if _, ok := seen[topic]; ok {
			continue
		}
		seen[topic] = struct{}{}
		topics = append(topics, topic)
	}
	return topics
}

func createTopics(ctx context.Context, cfg AuthConfig, topics []string, source string) error {
	cfg = cfg.withDefaults()
	if len(topics) == 0 {
		fmt.Printf("no topics found in %s\n", source)
		return nil
	}

	if err := WaitForKafka(ctx, cfg); err != nil {
		return err
	}

	address := Address(cfg.Port)
	conn, err := dial(ctx, address, cfg)
	if err != nil {
		return fmt.Errorf("connect to kafka: %w", err)
	}
	defer conn.Close()

	existing, err := existingTopics(conn)
	if err != nil {
		return err
	}

	configs := make([]kafkago.TopicConfig, 0, len(topics))
	for _, topic := range topics {
		if _, ok := existing[topic]; ok {
			continue
		}
		configs = append(configs, kafkago.TopicConfig{
			Topic:             topic,
			NumPartitions:     1,
			ReplicationFactor: 1,
		})
	}

	if len(configs) == 0 {
		fmt.Printf("topics already exist from %s\n", source)
		return nil
	}

	if err := conn.CreateTopics(configs...); err != nil {
		return fmt.Errorf("create topics: %w", err)
	}

	fmt.Printf("created %d topic(s) from %s\n", len(configs), source)
	return nil
}

func Address(port int) string {
	return fmt.Sprintf("127.0.0.1:%d", port)
}

func Credentials(port int) string {
	return CredentialsWithAuth(AuthConfig{Port: port})
}

func CredentialsWithAuth(cfg AuthConfig) string {
	cfg = cfg.withDefaults()
	return fmt.Sprintf(
		"%s %s %s username=%s password=%s",
		Address(cfg.Port), cfg.Protocol, cfg.Mechanism, cfg.User, cfg.Password,
	)
}

func dial(ctx context.Context, address string, cfg AuthConfig) (*kafkago.Conn, error) {
	cfg = cfg.withDefaults()
	dialer := &kafkago.Dialer{Timeout: 5 * time.Second}
	if cfg.Protocol != "PLAINTEXT" && cfg.Protocol != "SSL" && cfg.Mechanism == "PLAIN" {
		dialer.SASLMechanism = plain.Mechanism{Username: cfg.User, Password: cfg.Password}
	}
	return dialer.DialContext(ctx, "tcp", address)
}

func existingTopics(conn *kafkago.Conn) (map[string]struct{}, error) {
	partitions, err := conn.ReadPartitions()
	if err != nil {
		return nil, fmt.Errorf("read topics: %w", err)
	}

	topics := make(map[string]struct{})
	for _, partition := range partitions {
		topics[partition.Topic] = struct{}{}
	}
	return topics, nil
}

func readTopicList(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open topic list: %w", err)
	}
	defer file.Close()

	seen := make(map[string]struct{})
	var topics []string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if _, ok := seen[line]; ok {
			continue
		}
		seen[line] = struct{}{}
		topics = append(topics, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read topic list: %w", err)
	}

	return topics, nil
}
