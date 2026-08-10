package mqttadmin

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// testClient connects an admin *Client to a real broker via
// MQTT_BROKER_URL/MQTT_ADMIN_USERNAME/MQTT_ADMIN_PASSWORD, matching the
// DATABASE_URL skip-cleanly convention used elsewhere in this repo's test
// suite — this package's tests need a live Mosquitto with the
// dynamic-security plugin loaded, not just a database, so they skip on a
// different (but analogous) set of env vars. Run locally via:
// `docker compose up -d mosquitto` against this repo's own
// docker-compose.yml, with the same MQTT_BROKER_URL/MQTT_ADMIN_USERNAME/
// MQTT_ADMIN_PASSWORD already in .env for the API to use.
func testClient(t *testing.T) *Client {
	t.Helper()
	brokerURL := os.Getenv("MQTT_BROKER_URL_LOCAL")
	if brokerURL == "" {
		// MQTT_BROKER_URL itself is normally the docker-network address
		// (tcp://mosquitto:1883), only reachable from inside compose.
		// Tests run on the host, so they need the host-mapped address —
		// MQTT_BROKER_URL_LOCAL lets that differ without duplicating the
		// admin credential env vars under a different name too.
		brokerURL = "tcp://localhost:1883"
	}
	username := os.Getenv("MQTT_ADMIN_USERNAME")
	password := os.Getenv("MQTT_ADMIN_PASSWORD")
	if username == "" || password == "" {
		t.Skip("MQTT_ADMIN_USERNAME/MQTT_ADMIN_PASSWORD not set — skipping integration test that needs a real broker with dynsec")
	}

	client, err := NewClient(context.Background(), brokerURL, username, password)
	if err != nil {
		t.Skipf("could not connect to a live dynsec broker at %s — skipping: %v", brokerURL, err)
	}
	t.Cleanup(func() { client.mqtt.Disconnect(250) })
	return client
}

func uniqueDeviceID(prefix string) string {
	return fmt.Sprintf("%s%d", prefix, time.Now().UnixNano())
}

// tryConnect attempts a real MQTT connection with the given credentials
// and reports whether it succeeded — the actual proof that a dynsec
// command had a real effect on the broker, not just "returned no error."
func tryConnect(t *testing.T, brokerURL, username, password string) (mqtt.Client, bool) {
	t.Helper()
	opts := mqtt.NewClientOptions().
		AddBroker(brokerURL).
		SetClientID(username + "-test-" + fmt.Sprint(time.Now().UnixNano())).
		SetUsername(username).
		SetPassword(password).
		SetConnectTimeout(5 * time.Second).
		SetAutoReconnect(false)
	c := mqtt.NewClient(opts)
	token := c.Connect()
	ok := token.WaitTimeout(5*time.Second) && token.Error() == nil
	if !ok {
		return nil, false
	}
	return c, true
}

func localBrokerURL() string {
	if u := os.Getenv("MQTT_BROKER_URL_LOCAL"); u != "" {
		return u
	}
	return "tcp://localhost:1883"
}

func TestNewClientFromEnv_ReturnsNilWithoutError(t *testing.T) {
	t.Setenv("MQTT_BROKER_URL", "")
	t.Setenv("MQTT_ADMIN_USERNAME", "")
	t.Setenv("MQTT_ADMIN_PASSWORD", "")

	client, err := NewClientFromEnv(context.Background())
	if err != nil {
		t.Fatalf("expected no error when dynsec admin env vars are unset, got %v", err)
	}
	if client != nil {
		t.Fatal("expected a nil client when dynsec admin env vars are unset — callers rely on nil meaning 'not configured', not an error")
	}
}

// TestCreateDevice_ProvisionsAWorkingBrokerCredential is the core
// regression for the whole point of this package: a device created via
// CreateDevice must actually be able to authenticate to the real broker
// afterward — not just that the admin API call itself returned no error.
func TestCreateDevice_ProvisionsAWorkingBrokerCredential(t *testing.T) {
	admin := testClient(t)
	ctx := context.Background()
	deviceID := uniqueDeviceID("TEST-MQTTADMIN-")
	secret := "test-secret-123"

	if err := admin.CreateDevice(ctx, deviceID, secret); err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}
	t.Cleanup(func() { _ = admin.DeleteDevice(context.Background(), deviceID) })

	conn, ok := tryConnect(t, localBrokerURL(), deviceID, secret)
	if !ok {
		t.Fatal("expected the newly-created device to authenticate successfully with its real secret, but the connection failed")
	}
	conn.Disconnect(250)
}

// TestCreateDevice_RejectsWrongPassword confirms CreateDevice doesn't
// accidentally provision a broker account that accepts any password —
// the control case for the test above.
func TestCreateDevice_RejectsWrongPassword(t *testing.T) {
	admin := testClient(t)
	ctx := context.Background()
	deviceID := uniqueDeviceID("TEST-MQTTADMIN-")

	if err := admin.CreateDevice(ctx, deviceID, "correct-secret"); err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}
	t.Cleanup(func() { _ = admin.DeleteDevice(context.Background(), deviceID) })

	if _, ok := tryConnect(t, localBrokerURL(), deviceID, "wrong-secret"); ok {
		t.Fatal("expected authentication to fail with an incorrect password, but it succeeded")
	}
}

// TestDeleteDevice_RevokesBrokerAccess is the broker-side half of device
// revocation (internal/registry.Devices.Revoke's other half, alongside
// the app-level revoked_at check) — CLAUDE.md requires both paths to
// actually stop data flow, not just one.
func TestDeleteDevice_RevokesBrokerAccess(t *testing.T) {
	admin := testClient(t)
	ctx := context.Background()
	deviceID := uniqueDeviceID("TEST-MQTTADMIN-")
	secret := "test-secret-123"

	if err := admin.CreateDevice(ctx, deviceID, secret); err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}
	if conn, ok := tryConnect(t, localBrokerURL(), deviceID, secret); ok {
		conn.Disconnect(250)
	} else {
		t.Fatal("sanity check failed: device could not connect even before revocation")
	}

	if err := admin.DeleteDevice(ctx, deviceID); err != nil {
		t.Fatalf("DeleteDevice: %v", err)
	}

	if _, ok := tryConnect(t, localBrokerURL(), deviceID, secret); ok {
		t.Fatal("expected the deleted device's credential to be rejected, but it authenticated successfully")
	}
}

// TestSetDevicePassword_RotatesCredential confirms the old secret stops
// working the instant a new one is issued (not just that a new one also
// happens to work) — Devices.RotateSecret's whole premise.
func TestSetDevicePassword_RotatesCredential(t *testing.T) {
	admin := testClient(t)
	ctx := context.Background()
	deviceID := uniqueDeviceID("TEST-MQTTADMIN-")
	oldSecret := "old-secret-123"
	newSecret := "new-secret-456"

	if err := admin.CreateDevice(ctx, deviceID, oldSecret); err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}
	t.Cleanup(func() { _ = admin.DeleteDevice(context.Background(), deviceID) })

	if err := admin.SetDevicePassword(ctx, deviceID, newSecret); err != nil {
		t.Fatalf("SetDevicePassword: %v", err)
	}

	if _, ok := tryConnect(t, localBrokerURL(), deviceID, oldSecret); ok {
		t.Fatal("expected the old secret to be rejected after rotation, but it still authenticated")
	}
	conn, ok := tryConnect(t, localBrokerURL(), deviceID, newSecret)
	if !ok {
		t.Fatal("expected the new secret to authenticate successfully after rotation, but it failed")
	}
	conn.Disconnect(250)
}

// TestDeviceACL_CanPublishOwnTopicButNotAnothers is the ACL enforcement
// CLAUDE.md calls out explicitly: "a leaked credential for one device
// can't publish data for another site." Proven by actually subscribing
// as the ingestor role and observing which publishes arrive — a denied
// publish is silently dropped by the broker (no error on the publisher
// side), so arrival at a real subscriber is the only reliable signal.
func TestDeviceACL_CanPublishOwnTopicButNotAnothers(t *testing.T) {
	admin := testClient(t)
	ctx := context.Background()

	deviceA := uniqueDeviceID("TEST-MQTTADMIN-A-")
	deviceB := uniqueDeviceID("TEST-MQTTADMIN-B-")
	secretA := "secret-a-123"
	if err := admin.CreateDevice(ctx, deviceA, secretA); err != nil {
		t.Fatalf("CreateDevice A: %v", err)
	}
	t.Cleanup(func() { _ = admin.DeleteDevice(context.Background(), deviceA) })
	if err := admin.CreateDevice(ctx, deviceB, "secret-b-456"); err != nil {
		t.Fatalf("CreateDevice B: %v", err)
	}
	t.Cleanup(func() { _ = admin.DeleteDevice(context.Background(), deviceB) })

	// A throwaway ingestor-role subscriber, same role/ACL bootstrap()
	// already grants the real ingestor service account.
	subUser := uniqueDeviceID("TEST-MQTTADMIN-SUB-")
	subPass := "sub-secret-123"
	if err := admin.send(ctx, []map[string]any{
		{"command": "createClient", "username": subUser, "password": subPass},
		{"command": "addClientRole", "username": subUser, "rolename": IngestorRole, "priority": 0},
	}); err != nil {
		t.Fatalf("provision test subscriber: %v", err)
	}
	t.Cleanup(func() {
		_ = admin.send(context.Background(), []map[string]any{{"command": "deleteClient", "username": subUser}})
	})

	sub := mqtt.NewClient(mqtt.NewClientOptions().
		AddBroker(localBrokerURL()).
		SetClientID(subUser).
		SetUsername(subUser).
		SetPassword(subPass).
		SetConnectTimeout(5 * time.Second))
	if token := sub.Connect(); token.WaitTimeout(5*time.Second) && token.Error() != nil {
		t.Fatalf("subscriber connect: %v", token.Error())
	}
	t.Cleanup(func() { sub.Disconnect(250) })

	received := make(chan string, 10)
	if token := sub.Subscribe("devices/+/telemetry", 1, func(_ mqtt.Client, msg mqtt.Message) {
		received <- msg.Topic()
	}); token.WaitTimeout(5*time.Second) && token.Error() != nil {
		t.Fatalf("subscribe: %v", token.Error())
	}
	// Give the subscription a moment to actually register with the
	// broker before publishing — otherwise an early publish can race the
	// SUBSCRIBE ack.
	time.Sleep(300 * time.Millisecond)

	pubA := mqtt.NewClient(mqtt.NewClientOptions().
		AddBroker(localBrokerURL()).
		SetClientID(deviceA).
		SetUsername(deviceA).
		SetPassword(secretA).
		SetConnectTimeout(5 * time.Second))
	if token := pubA.Connect(); token.WaitTimeout(5*time.Second) && token.Error() != nil {
		t.Fatalf("device A connect: %v", token.Error())
	}
	t.Cleanup(func() { pubA.Disconnect(250) })

	// Allowed: device A publishing to its own topic.
	ownTopic := "devices/" + deviceA + "/telemetry"
	if token := pubA.Publish(ownTopic, 1, false, `{"device_id":"`+deviceA+`"}`); token.WaitTimeout(5*time.Second) && token.Error() != nil {
		t.Fatalf("publish to own topic: %v", token.Error())
	}
	select {
	case topic := <-received:
		if topic != ownTopic {
			t.Fatalf("expected to receive device A's own topic %q, got %q", ownTopic, topic)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("expected device A's publish to its own topic to be delivered to the ingestor-role subscriber, but nothing arrived")
	}

	// Denied: device A publishing to device B's topic.
	otherTopic := "devices/" + deviceB + "/telemetry"
	if token := pubA.Publish(otherTopic, 1, false, `{"device_id":"`+deviceB+`","spoofed":true}`); token.WaitTimeout(5*time.Second) && token.Error() != nil {
		t.Fatalf("publish to another device's topic (expected to be silently denied, not a client-side error): %v", token.Error())
	}
	select {
	case topic := <-received:
		t.Fatalf("expected device A's publish to device B's topic to be denied by the ACL, but it was delivered on topic %q", topic)
	case <-time.After(2 * time.Second):
		// Correct: nothing arrived within the window.
	}
}
