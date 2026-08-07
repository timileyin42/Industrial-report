// Command loadtest simulates N virtual devices publishing telemetry over
// MQTT, to find where this stack's throughput first degrades before a
// real fleet rollout (concept note §13: "load-test with simulated devices
// before large roll-outs"). This is diagnostic, not a pass/fail gate —
// see docs/load-test-results.md for recorded runs and findings.
//
// Devices must be pre-provisioned first: scripts/loadtest-provision.sh
// registers them through the real API (exercising the actual registry,
// not bypassing it) and writes their credentials to loadtest-devices.csv.
//
// Optionally also hammers the HTTP API concurrently (-api-token,
// -api-rps) to see whether read load compounds with write load, instead
// of building a second separate tool for that.
package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type deviceCred struct {
	deviceID string
	secret   string
}

func main() {
	devicesFile := flag.String("devices-file", "loadtest-devices.csv", "CSV of device_id,secret from scripts/loadtest-provision.sh")
	broker := flag.String("broker", "tcp://localhost:1883", "MQTT broker URL")
	duration := flag.Duration("duration", 30*time.Second, "how long to run the test")
	interval := flag.Duration("interval", 5*time.Second, "publish interval per device")
	connectStagger := flag.Duration("connect-stagger", 5*time.Millisecond, "delay between successive device connections, to avoid a connection storm")
	apiBase := flag.String("api-base", "http://localhost:8080/v1", "API base URL for the optional concurrent read load")
	apiToken := flag.String("api-token", "", "operator JWT — if set, also hammers GET /fleet/summary concurrently")
	apiRPS := flag.Int("api-rps", 0, "concurrent API requests/sec during the test (0 = disabled)")
	flag.Parse()

	creds, err := loadDeviceCreds(*devicesFile)
	if err != nil {
		log.Fatalf("load devices file: %v", err)
	}
	log.Printf("loaded %d device credentials from %s", len(creds), *devicesFile)

	var connected, connectFailed, published, publishFailed int64

	ctx, cancel := context.WithTimeout(context.Background(), *duration+10*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	clients := make([]mqtt.Client, 0, len(creds))
	var clientsMu sync.Mutex

	log.Printf("connecting %d devices (staggered %s apart)...", len(creds), *connectStagger)
	for _, cred := range creds {
		opts := mqtt.NewClientOptions().
			AddBroker(*broker).
			SetClientID("loadtest-" + cred.deviceID).
			SetUsername(cred.deviceID).
			SetPassword(cred.secret).
			SetConnectTimeout(10 * time.Second)
		client := mqtt.NewClient(opts)
		if token := client.Connect(); token.WaitTimeout(10*time.Second) && token.Error() != nil {
			atomic.AddInt64(&connectFailed, 1)
			log.Printf("connect failed for %s: %v", cred.deviceID, token.Error())
			continue
		}
		atomic.AddInt64(&connected, 1)
		clientsMu.Lock()
		clients = append(clients, client)
		clientsMu.Unlock()
		time.Sleep(*connectStagger)
	}
	log.Printf("connected: %d, failed: %d", connected, connectFailed)

	if *apiRPS > 0 && *apiToken != "" {
		wg.Add(1)
		go hammerAPI(ctx, &wg, *apiBase, *apiToken, *apiRPS)
	}

	log.Printf("publishing every %s for %s...", *interval, *duration)
	stopPublishing := time.After(*duration)
	publishCtx, cancelPublish := context.WithCancel(ctx)

	for i, cred := range creds {
		if i >= len(clients) {
			break
		}
		wg.Add(1)
		go simulateDevice(publishCtx, &wg, clients[i], cred.deviceID, *interval, &published, &publishFailed)
	}

	<-stopPublishing
	cancelPublish()
	wg.Wait()

	for _, c := range clients {
		c.Disconnect(250)
	}

	elapsed := *duration
	log.Println("--- results ---")
	log.Printf("devices connected:   %d / %d", connected, len(creds))
	log.Printf("connect failures:    %d", connectFailed)
	log.Printf("publishes succeeded: %d", published)
	log.Printf("publishes failed:    %d", publishFailed)
	log.Printf("effective rate:      %.1f publishes/sec", float64(published)/elapsed.Seconds())
	log.Println("Record this run in docs/load-test-results.md.")
}

func loadDeviceCreds(path string) ([]deviceCred, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	rows, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(rows) < 2 {
		return nil, fmt.Errorf("no device rows found in %s", path)
	}

	creds := make([]deviceCred, 0, len(rows)-1)
	for _, row := range rows[1:] { // skip header
		if len(row) < 2 {
			continue
		}
		creds = append(creds, deviceCred{deviceID: row[0], secret: row[1]})
	}
	return creds, nil
}

// simulateDevice publishes a plausible day/night power curve with a
// monotonically increasing energy counter and occasional timestamp jitter
// — same shape of data the ingestor's validation logic expects to see
// from a real device (internal/domain).
func simulateDevice(ctx context.Context, wg *sync.WaitGroup, client mqtt.Client, deviceID string, interval time.Duration, published, publishFailed *int64) {
	defer wg.Done()

	energy := rand.Float64() * 1000
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			hour := time.Now().UTC().Hour()
			power := 0.0
			if hour >= 6 && hour <= 18 {
				power = 1 + rand.Float64()*3
			}
			energy += power * interval.Hours()

			ts := time.Now().UTC()
			if rand.Float64() < 0.05 {
				ts = ts.Add(-time.Duration(rand.Intn(20)) * time.Minute) // occasional jitter/backfill
			}

			payload := fmt.Sprintf(
				`{"device_id":"%s","ts":"%s","power_kw":%.2f,"energy_kwh_total":%.2f,"status":"ok"}`,
				deviceID, ts.Format(time.RFC3339), power, energy,
			)
			token := client.Publish("devices/"+deviceID+"/telemetry", 1, false, payload)
			if token.WaitTimeout(5*time.Second) && token.Error() != nil {
				atomic.AddInt64(publishFailed, 1)
				continue
			}
			atomic.AddInt64(published, 1)
		}
	}
}

// hammerAPI fires concurrent reads against the fleet summary endpoint
// during the run, to see whether read load compounds with write load —
// folded into this tool rather than building a second one.
func hammerAPI(ctx context.Context, wg *sync.WaitGroup, apiBase, token string, rps int) {
	defer wg.Done()
	ticker := time.NewTicker(time.Second / time.Duration(rps))
	defer ticker.Stop()

	client := &http.Client{Timeout: 5 * time.Second}
	var ok, failed int64

	for {
		select {
		case <-ctx.Done():
			log.Printf("api reads: ok=%d failed=%d", ok, failed)
			return
		case <-ticker.C:
			req, _ := http.NewRequestWithContext(ctx, http.MethodGet, apiBase+"/fleet/summary", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			resp, err := client.Do(req)
			if err != nil || resp.StatusCode != http.StatusOK {
				failed++
				if resp != nil {
					resp.Body.Close()
				}
				continue
			}
			resp.Body.Close()
			ok++
		}
	}
}
