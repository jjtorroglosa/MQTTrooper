package internal

import (
	"context"
	"fmt"
	"hash/fnv"
	"log"
	"os"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/stretchr/testify/assert"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type testLogConsumer struct{}

func (g *testLogConsumer) Accept(l testcontainers.Log) {
	log.Println("[🐳] " + string(l.Content))
}

type mqttTest struct {
	*testing.T
	ctx           context.Context
	any           *gofakeit.Faker
	testContainer testcontainers.Container
	teardown      func()
	cfg           Config
}

func hash(s string) uint64 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return uint64(h.Sum32())
}

func setupMqttTest(t *testing.T, fixedHostPort *string) *mqttTest {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	any := gofakeit.New(hash(t.Name()))
	req := testcontainers.ContainerRequest{
		Image:        "eclipse-mosquitto:2.0",
		ExposedPorts: []string{"1883/tcp"},
		Cmd:          []string{"mosquitto", "-c", "/mosquitto-no-auth.conf", "-v"},
		WaitingFor:   wait.ForListeningPort("1883/tcp"),
		LogConsumerCfg: &testcontainers.LogConsumerConfig{
			Opts: []testcontainers.LogProductionOption{
				testcontainers.WithLogProductionTimeout(10 * time.Second),
			},
			Consumers: []testcontainers.LogConsumer{&testLogConsumer{}},
		},
	}
	if fixedHostPort != nil {
		// Use fixed port on host instead of random (needed when testing reconnection
		// so that it assigns the same port after restarting the container)
		req.HostConfigModifier = func(hostConfig *container.HostConfig) {
			hostConfig.PortBindings = nat.PortMap{
				"1883/tcp": []nat.PortBinding{
					{
						HostIP:   "127.0.0.1",
						HostPort: *fixedHostPort,
					},
				},
			}
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	mqttC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	assert.NoError(t, err)

	return &mqttTest{
		T:             t,
		ctx:           ctx,
		any:           any,
		testContainer: mqttC,
		teardown: func() {
			cancel()
			testcontainers.CleanupContainer(t, mqttC)
			log.Println("Teardown finished")
		},
		cfg: Config{
			Mqtt: MqttConfig{
				Enabled: true,
				Topic:   "/any/topic",
			},
			Executor: ExecutorConfig{
				Shell:  "/bin/bash",
				DryRun: false,
			},
		},
	}
}

func TestMqttSubscriptionsReceiveCommands(test *testing.T) {
	t := setupMqttTest(test, nil)
	defer t.teardown()
	filename := fmt.Sprintf("/tmp/mqttrooper-test-%s", t.any.LetterN(8))
	defer func() {
		_ = os.Remove(filename)
	}()
	serviceName := t.any.LetterN(8)
	expectedString := t.any.LetterN(8)
	cfg := t.cfg
	cfg.Services = map[string]string{
		serviceName: fmt.Sprintf("echo -n %s > %s", expectedString, filename),
	}
	cfg.Executor.DryRun = false
	actualExecutor := CreateExecutor(cfg.Executor.DryRun, cfg.Executor.Shell, cfg.Services)
	host, _ := t.testContainer.Host(t.ctx)
	port, _ := t.testContainer.MappedPort(t.ctx, "1883/tcp")
	received := make(chan string, 2)
	executor := Executor(func(service string) error {
		err := actualExecutor(service)
		received <- string(service)
		return err
	})
	subscriber := Connect(fmt.Sprintf("%s:%s", host, port), "subs", "subs", "subs", cfg.Mqtt.Topic, executor)
	assert.True(t, subscriber.IsConnected())
	defer subscriber.Disconnect(250)

	publisher := Connect(
		fmt.Sprintf("%s:%s", host, port),
		"pub",
		"pub",
		"pub",
		cfg.Mqtt.Topic,
		nil,
	)
	defer publisher.Disconnect(250)
	token := publisher.Publish(cfg.Mqtt.Topic, qos, false, serviceName)
	if token.Error() != nil {
		log.Printf("Error publishing payload %s on topic %s. \nError: %s\n", serviceName, cfg.Mqtt.Topic, token.Error())
	}
	token.Wait()
	// Wait for message to be received with timeout
	select {
	case msg := <-received:
		assert.Equal(t, serviceName, msg)
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for message")
	}

	bytes, err := os.ReadFile(filename)
	assert.NoError(t, err)
	assert.Equal(t, expectedString, string(bytes))
}

func TestWhenBrokerIsRestartedClientReconnects(test *testing.T) {
	port := "63033"
	t := setupMqttTest(test, &port)
	defer t.teardown()

	filename := fmt.Sprintf("/tmp/mqttrooper-test-%s", t.any.LetterN(8))
	serviceName := t.any.LetterN(8)
	expectedString := t.any.LetterN(8)

	cfg := t.cfg
	cfg.Services = map[string]string{
		serviceName: fmt.Sprintf("echo -n %s > %s", expectedString, filename),
	}
	cfg.Executor.DryRun = false

	actualExecutor := CreateExecutor(cfg.Executor.DryRun, cfg.Executor.Shell, cfg.Services)
	host, _ := t.testContainer.Host(t.ctx)
	received := make(chan string, 2)
	executor := Executor(func(service string) error {
		err := actualExecutor(service)
		received <- string(service)
		return err
	})
	subscriber := Connect(fmt.Sprintf("%s:%s", host, port), "subs", "subs", "subs", cfg.Mqtt.Topic, executor)
	assert.True(t, subscriber.IsConnectionOpen())
	defer subscriber.Disconnect(250)

	restartContainer(t, subscriber)
	assert.True(t, subscriber.IsConnectionOpen())

	publisher := Connect(
		fmt.Sprintf("%s:%s", host, port),
		"pub",
		"pub",
		"pub",
		cfg.Mqtt.Topic,
		nil,
	)
	defer publisher.Disconnect(250)
	token := publisher.Publish(cfg.Mqtt.Topic, qos, false, serviceName)
	assert.NoError(t, token.Error())
	token.WaitTimeout(2 * time.Second)
	//Wait for message to be received with timeout
	select {
	case msg := <-received:
		assert.Equal(t, serviceName, msg)
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for message")
	}

	bytes, err := os.ReadFile(filename)
	assert.NoError(t, err)
	assert.Equal(t, expectedString, string(bytes))
}

func restartContainer(t *mqttTest, subscriber mqtt.Client) {
	timeout := 5 * time.Second
	err := t.testContainer.Stop(t.ctx, &timeout)
	assert.False(t, subscriber.IsConnectionOpen())
	assert.NoError(t, err)
	err = t.testContainer.Start(t.ctx)
	assert.NoError(t, err)
	waitUntilConnected(subscriber)
}

func waitUntilConnected(subscriber mqtt.Client) {
	for i := range 100 {
		log.Println("Checking if client connected: ", i)
		if subscriber.IsConnectionOpen() {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
}
