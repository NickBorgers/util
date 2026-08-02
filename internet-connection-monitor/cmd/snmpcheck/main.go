package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gosnmp/gosnmp"
)

func main() {
	log.SetFlags(0)

	target := flag.String("target", "127.0.0.1", "SNMP agent host")
	port := flag.Int("port", 161, "SNMP agent UDP port")
	community := flag.String("community", "public", "SNMP community string")
	baseOID := flag.String("base", ".1.3.6.1.4.1.99999", "Base OID to query")
	retries := flag.Int("retries", 3, "Number of SNMP retries")
	timeout := flag.Duration("timeout", 3*time.Second, "Timeout for SNMP requests")
	flag.Parse()

	normalizedBase := normalizeOID(*baseOID)
	cacheOID := normalizedBase + ".1.0"

	client := &gosnmp.GoSNMP{
		Target:    *target,
		Port:      uint16(*port),
		Community: *community,
		Version:   gosnmp.Version2c,
		Retries:   *retries,
		Timeout:   *timeout,
		MaxOids:   gosnmp.MaxOids,
		Transport: "udp",
	}

	if err := client.Connect(); err != nil {
		log.Fatalf("failed to connect to SNMP agent %s:%d: %v", *target, *port, err)
	}
	defer func() {
		_ = client.Conn.Close()
	}()

	response, err := client.Get([]string{cacheOID})
	if err != nil {
		log.Fatalf("failed to fetch cache OID %s: %v", cacheOID, err)
	}
	if len(response.Variables) == 0 {
		log.Fatalf("no variables returned for cache OID %s", cacheOID)
	}

	cacheSize, err := numericValue(response.Variables[0])
	if err != nil {
		log.Fatalf("unable to parse cache size from %s: %v", cacheOID, err)
	}

	var totalVars int
	var siteEntries int
	sitePrefix := normalizedBase + ".5."

	err = client.Walk(normalizedBase, func(pdu gosnmp.SnmpPDU) error {
		totalVars++
		if strings.HasPrefix(pdu.Name, sitePrefix) && strings.HasSuffix(pdu.Name, ".1") {
			siteEntries++
		}
		return nil
	})
	if err != nil {
		log.Fatalf("failed to walk SNMP tree at %s: %v", normalizedBase, err)
	}

	if totalVars == 0 {
		log.Fatalf("SNMP walk for %s returned no results", normalizedBase)
	}
	if siteEntries == 0 {
		log.Fatalf("SNMP walk did not include any site entries under %s. Received %d variables", sitePrefix, totalVars)
	}

	fmt.Printf("SNMP agent healthy: cache_size=%d, variables=%d, site_entries=%d\n", cacheSize, totalVars, siteEntries)
}

func normalizeOID(oid string) string {
	trimmed := strings.TrimSpace(oid)
	if trimmed == "" {
		return ".1.3.6.1.4.1.99999"
	}
	if !strings.HasPrefix(trimmed, ".") {
		trimmed = "." + trimmed
	}
	for strings.HasSuffix(trimmed, ".") && len(trimmed) > 1 {
		trimmed = trimmed[:len(trimmed)-1]
	}
	return trimmed
}

func numericValue(pdu gosnmp.SnmpPDU) (uint64, error) {
	bigInt := gosnmp.ToBigInt(pdu.Value)
	if bigInt == nil {
		return 0, fmt.Errorf("non-numeric SNMP value type %T", pdu.Value)
	}
	if bigInt.Sign() < 0 {
		return 0, errors.New("expected non-negative value")
	}
	return bigInt.Uint64(), nil
}
