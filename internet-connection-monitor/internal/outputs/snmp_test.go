package outputs

import (
	"testing"
	"time"

	"github.com/gosnmp/gosnmp"

	"github.com/nickborgers/monorepo/internet-connection-monitor/internal/config"
	"github.com/nickborgers/monorepo/internet-connection-monitor/internal/models"
)

func TestSNMPAgentRespondsToGetAndWalk(t *testing.T) {
	cfg := &config.SNMPConfig{
		Enabled:       true,
		Port:          0,
		Community:     "public",
		ListenAddress: "127.0.0.1",
		EnterpriseOID: ".1.3.6.1.4.1.55555",
	}

	snmpOutput, err := NewSNMPOutput(cfg)
	if err != nil {
		t.Fatalf("failed to create SNMP output: %v", err)
	}
	defer snmpOutput.Close()
	t.Log("SNMP output started")

	now := time.Now()
	resultSuccess := &models.TestResult{
		Timestamp: now,
		Site: models.SiteInfo{
			Name: "example.com",
			URL:  "https://example.com",
		},
		Status:  models.StatusInfo{Success: true},
		Timings: models.TimingMetrics{TotalDurationMs: 150},
	}
	if err := snmpOutput.Write(resultSuccess); err != nil {
		t.Fatalf("failed to write success result: %v", err)
	}
	t.Log("wrote success result")

	resultFailure := &models.TestResult{
		Timestamp: now.Add(500 * time.Millisecond),
		Site: models.SiteInfo{
			Name: "example.com",
			URL:  "https://example.com",
		},
		Status:  models.StatusInfo{Success: false},
		Timings: models.TimingMetrics{TotalDurationMs: 320},
	}
	if err := snmpOutput.Write(resultFailure); err != nil {
		t.Fatalf("failed to write failure result: %v", err)
	}
	t.Log("wrote failure result")

	client := &gosnmp.GoSNMP{
		Target:    cfg.ListenAddress,
		Port:      uint16(snmpOutput.Port()),
		Community: cfg.Community,
		Version:   gosnmp.Version2c,
		Timeout:   time.Second,
		Retries:   1,
	}
	if err := client.Connect(); err != nil {
		t.Fatalf("failed to connect SNMP client: %v", err)
	}
	defer client.Conn.Close()
	t.Logf("connected SNMP client on port %d", snmpOutput.Port())

	baseOID := ".1.3.6.1.4.1.55555"

	packet, err := client.Get([]string{baseOID + ".1.0"})
	if err != nil {
		t.Fatalf("snmp get failed: %v", err)
	}
	if len(packet.Variables) != 1 {
		t.Fatalf("expected 1 variable, got %d", len(packet.Variables))
	}
	if got := pduValueAsUint32(t, packet.Variables[0]); got != 2 {
		t.Fatalf("expected cache size 2, got %d", got)
	}
	t.Log("verified cache size via SNMP get")

	// Ensure walking the tree returns the site name and metrics without error.
	walked := make([]gosnmp.SnmpPDU, 0)
	if err := client.Walk(baseOID, func(pdu gosnmp.SnmpPDU) error {
		walked = append(walked, pdu)
		return nil
	}); err != nil {
		t.Fatalf("snmp walk failed: %v", err)
	}
	t.Logf("walked %d OIDs", len(walked))
	if len(walked) == 0 {
		t.Fatalf("expected walk results")
	}

	foundSiteName := false
	for _, pdu := range walked {
		if pdu.Type == gosnmp.OctetString {
			switch v := pdu.Value.(type) {
			case []byte:
				if string(v) == "example.com" {
					foundSiteName = true
				}
			case string:
				if v == "example.com" {
					foundSiteName = true
				}
			}
		}
	}
	if !foundSiteName {
		t.Fatalf("expected to find site name in walk results")
	}

	// Unknown OIDs should return NoSuchObject
	packet, err = client.Get([]string{baseOID + ".99.0"})
	if err != nil {
		t.Fatalf("snmp get for missing OID failed: %v", err)
	}
	if len(packet.Variables) != 1 {
		t.Fatalf("expected 1 variable for missing OID, got %d", len(packet.Variables))
	}
	if packet.Variables[0].Type != gosnmp.NoSuchObject {
		t.Fatalf("expected NoSuchObject, got %v", packet.Variables[0].Type)
	}
	t.Log("verified missing OID response")

	// Walk should eventually end with EndOfMibView via GetNext past the last site metric.
	packet, err = client.GetNext([]string{baseOID + ".5.1.10"})
	if err != nil {
		t.Fatalf("snmp getnext failed: %v", err)
	}
	if len(packet.Variables) != 1 {
		t.Fatalf("expected 1 variable for getnext, got %d", len(packet.Variables))
	}
	if packet.Variables[0].Type != gosnmp.EndOfMibView {
		t.Fatalf("expected EndOfMibView, got %v", packet.Variables[0].Type)
	}
	t.Log("verified end of MIB view")
}

func pduValueAsUint32(t *testing.T, pdu gosnmp.SnmpPDU) uint32 {
	t.Helper()
	switch v := pdu.Value.(type) {
	case uint:
		return uint32(v)
	case uint32:
		return v
	case uint64:
		return uint32(v)
	case int:
		if v < 0 {
			t.Fatalf("negative value %d", v)
		}
		return uint32(v)
	case int64:
		if v < 0 {
			t.Fatalf("negative value %d", v)
		}
		return uint32(v)
	default:
		t.Fatalf("unexpected value type %T", v)
	}
	return 0
}
