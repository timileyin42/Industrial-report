// Package mqttadmin automates syncing device credentials into
// Mosquitto's dynamic-security ("dynsec") plugin — the step that used
// to require an operator to manually run `mosquitto_passwd` and restart
// the broker every time a device was registered, rotated, or revoked.
//
// The wire protocol here (request/response topics, JSON shapes, ACL
// type names) is NOT taken from Mosquitto's own documentation — that
// page explicitly omits it, documenting only the `mosquitto_ctrl` CLI
// tool, which isn't packaged for Alpine (our runtime base image) at any
// available version. Every command and response shape in this file was
// instead verified live against a real eclipse-mosquitto:2 broker
// (create/setPassword/delete, role ACLs, %u substitution, and the
// ingestor's wildcard subscribe access all round-tripped and were
// checked for the correct allow/deny behavior, not just "no error").
package mqttadmin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const (
	controlTopic  = "$CONTROL/dynamic-security/v1"
	responseTopic = "$CONTROL/dynamic-security/v1/response"

	// DeviceRole is shared by every device client — one role, one ACL
	// using the %u (username) placeholder, rather than a bespoke role
	// per device. A device's username must equal its device_id for this
	// substitution to scope it to the right topic, same requirement the
	// old static acl.conf's %u pattern had.
	DeviceRole = "device"
	// IngestorRole is the one service account that needs to subscribe
	// across every device's topic — the mirror of the old acl.conf's
	// dedicated "ingestor-service" stanza.
	IngestorRole = "ingestor"
)

type Client struct {
	mqtt   mqtt.Client
	mu     sync.Mutex
	respCh chan []byte
}

// NewClientFromEnv returns nil (not an error) when MQTT_ADMIN_USERNAME/
// MQTT_ADMIN_PASSWORD aren't set — the same "optional, additive infra"
// pattern as internal/storage.NewFromEnv and internal/email.NewSenderFromEnv.
// Callers must treat a nil *Client as "dynsec sync isn't configured,"
// never as an error that blocks the action being synced.
func NewClientFromEnv(ctx context.Context) (*Client, error) {
	brokerURL := os.Getenv("MQTT_BROKER_URL")
	username := os.Getenv("MQTT_ADMIN_USERNAME")
	password := os.Getenv("MQTT_ADMIN_PASSWORD")
	if brokerURL == "" || username == "" || password == "" {
		return nil, nil
	}
	return NewClient(ctx, brokerURL, username, password)
}

func NewClient(ctx context.Context, brokerURL, username, password string) (*Client, error) {
	respCh := make(chan []byte, 1)

	opts := mqtt.NewClientOptions().
		AddBroker(brokerURL).
		SetClientID("zgnis-api-dynsec-admin").
		SetUsername(username).
		SetPassword(password).
		SetAutoReconnect(true).
		SetConnectTimeout(10 * time.Second)

	c := mqtt.NewClient(opts)
	if token := c.Connect(); token.WaitTimeout(10*time.Second) && token.Error() != nil {
		return nil, fmt.Errorf("dynsec admin connect: %w", token.Error())
	}

	handler := func(_ mqtt.Client, msg mqtt.Message) {
		select {
		case respCh <- msg.Payload():
		default:
			// A response arrived with nothing waiting for it (e.g. a
			// stale message from a prior connection) — drop it rather
			// than block the MQTT client's callback goroutine.
		}
	}
	if token := c.Subscribe(responseTopic, 1, handler); token.WaitTimeout(10*time.Second) && token.Error() != nil {
		c.Disconnect(250)
		return nil, fmt.Errorf("dynsec admin subscribe to response topic: %w", token.Error())
	}

	client := &Client{mqtt: c, respCh: respCh}
	if err := client.bootstrap(ctx); err != nil {
		return nil, err
	}
	return client, nil
}

type dynsecResponse struct {
	Responses []struct {
		Command string `json:"command"`
		Error   string `json:"error,omitempty"`
	} `json:"responses"`
}

// send publishes one batch of commands and waits for the broker's
// response, matching purely on request ordering: every dynsec call
// through this Client is serialized by mu, so the very next message on
// the response topic after a publish is guaranteed to be that publish's
// response, with no need for the correlation-data mechanism the plugin's
// own docs don't describe.
func (c *Client) send(ctx context.Context, commands []map[string]any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	select {
	case <-c.respCh:
	default:
	}

	payload, err := json.Marshal(map[string]any{"commands": commands})
	if err != nil {
		return err
	}

	token := c.mqtt.Publish(controlTopic, 1, false, payload)
	if token.WaitTimeout(10*time.Second) && token.Error() != nil {
		return fmt.Errorf("dynsec publish: %w", token.Error())
	}

	select {
	case raw := <-c.respCh:
		var resp dynsecResponse
		if err := json.Unmarshal(raw, &resp); err != nil {
			return fmt.Errorf("dynsec: unparseable response: %w", err)
		}
		for _, r := range resp.Responses {
			if r.Error != "" {
				return &CommandError{Command: r.Command, Message: r.Error}
			}
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(10 * time.Second):
		return errors.New("dynsec: timed out waiting for broker response")
	}
}

// CommandError carries the broker's own error string (e.g. "Client
// already exists") so callers like bootstrap() can pattern-match on
// expected idempotency errors without string-matching a wrapped message.
type CommandError struct {
	Command string
	Message string
}

func (e *CommandError) Error() string {
	return fmt.Sprintf("dynsec %s: %s", e.Command, e.Message)
}

func alreadyExists(err error) bool {
	var ce *CommandError
	return errors.As(err, &ce) && strings.Contains(strings.ToLower(ce.Message), "already exists")
}

// bootstrap ensures the two roles this platform needs exist, and that
// the ingestor's own service account (MQTT_USERNAME/MQTT_PASSWORD —
// the same credential cmd/ingestor connects with) is provisioned and
// assigned the ingestor role. Runs on every API startup; every step is
// idempotent (an "already exists" response is treated as success, not
// an error), so this is safe to repeat on every restart rather than
// needing a one-time manual setup script.
func (c *Client) bootstrap(ctx context.Context) error {
	if err := c.send(ctx, []map[string]any{{
		"command":  "createRole",
		"rolename": DeviceRole,
		"acls": []map[string]any{
			{"acltype": "publishClientSend", "topic": "devices/%u/telemetry", "allow": true, "priority": 0},
		},
	}}); err != nil && !alreadyExists(err) {
		return fmt.Errorf("bootstrap device role: %w", err)
	}

	if err := c.send(ctx, []map[string]any{{
		"command":  "createRole",
		"rolename": IngestorRole,
		"acls": []map[string]any{
			{"acltype": "subscribePattern", "topic": "devices/+/telemetry", "allow": true, "priority": 0},
			{"acltype": "publishClientReceive", "topic": "devices/+/telemetry", "allow": true, "priority": 0},
		},
	}}); err != nil && !alreadyExists(err) {
		return fmt.Errorf("bootstrap ingestor role: %w", err)
	}

	ingestorUser := os.Getenv("MQTT_USERNAME")
	ingestorPass := os.Getenv("MQTT_PASSWORD")
	if ingestorUser != "" && ingestorPass != "" {
		err := c.send(ctx, []map[string]any{
			{"command": "createClient", "username": ingestorUser, "password": ingestorPass, "textname": "Telemetry ingestor service"},
			{"command": "addClientRole", "username": ingestorUser, "rolename": IngestorRole, "priority": 0},
		})
		if err != nil && !alreadyExists(err) {
			return fmt.Errorf("bootstrap ingestor client: %w", err)
		}
	} else {
		log.Println("mqttadmin: MQTT_USERNAME/MQTT_PASSWORD not set — skipping ingestor service account bootstrap")
	}

	return nil
}

// CreateDevice provisions a device's broker credential and assigns it
// the shared device role — the automated replacement for the old
// "run mosquitto_passwd by hand, then restart the broker" step. Called
// right after Devices.Register generates a device's plaintext secret.
func (c *Client) CreateDevice(ctx context.Context, deviceID, secret string) error {
	return c.send(ctx, []map[string]any{
		{"command": "createClient", "username": deviceID, "password": secret, "textname": "Zgnis device " + deviceID},
		{"command": "addClientRole", "username": deviceID, "rolename": DeviceRole, "priority": 0},
	})
}

// SetDevicePassword rotates a device's broker credential — called
// alongside Devices.RotateSecret so the old secret stops authenticating
// to the broker the instant a new one is issued, not just in our own
// database.
func (c *Client) SetDevicePassword(ctx context.Context, deviceID, newSecret string) error {
	return c.send(ctx, []map[string]any{
		{"command": "setClientPassword", "username": deviceID, "password": newSecret},
	})
}

// DeleteDevice removes a device's broker credential entirely — called
// alongside Devices.Revoke. Deletion (not disablement) matches this
// platform's revoke semantics: there is no "unrevoke" path today, so a
// revoked device's broker identity is gone for good, the same as its
// app-level revoked_at is never cleared.
func (c *Client) DeleteDevice(ctx context.Context, deviceID string) error {
	return c.send(ctx, []map[string]any{{"command": "deleteClient", "username": deviceID}})
}
