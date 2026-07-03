package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

const defaultOUIDBPath = "/usr/local/share/bindnet/oui.csv"

var (
	localOUIOnce    sync.Once
	localOUIVendors map[string]string
	localOUIErr     error
)

func lookupMACVendor(ctx context.Context, mac string) (string, error) {
	vendor, err := lookupMACVendorRemote(ctx, mac)
	if vendor != "" {
		return vendor, nil
	}
	if localVendor := lookupLocalOUIVendor(mac); localVendor != "" {
		if err != nil {
			log.Printf("[backend] MAC Vendors indisponivel para %s (%v); usando OUI local", mac, err)
		}
		return localVendor, nil
	}
	if err != nil {
		return "", err
	}
	return "", nil
}

func lookupMACVendorRemote(ctx context.Context, mac string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.macvendors.com/"+url.PathEscape(mac), nil)
	if err != nil {
		return "", err
	}
	resp, err := (&http.Client{Timeout: 8 * time.Second}).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	if resp.StatusCode == http.StatusNotFound {
		return "", nil
	}
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("MAC Vendors retornou %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return strings.TrimSpace(string(body)), nil
}

func lookupLocalOUIVendor(mac string) string {
	prefix := normalizedMACHex(mac)
	if len(prefix) < 6 {
		return ""
	}
	localOUIOnce.Do(func() {
		localOUIVendors, localOUIErr = loadLocalOUIVendors()
		if localOUIErr != nil {
			log.Printf("[backend] base OUI local indisponivel: %v", localOUIErr)
		}
	})
	for _, size := range []int{9, 7, 6} {
		if len(prefix) >= size {
			if vendor := localOUIVendors[prefix[:size]]; vendor != "" {
				return vendor
			}
		}
	}
	for _, size := range []int{9, 7, 6} {
		if len(prefix) >= size {
			if vendor := embeddedOUIFallback[prefix[:size]]; vendor != "" {
				return vendor
			}
		}
	}
	return ""
}

func loadLocalOUIVendors() (map[string]string, error) {
	path := strings.TrimSpace(os.Getenv("BINDNET_OUI_DB_PATH"))
	if path == "" {
		path = defaultOUIDBPath
	}
	file, err := os.Open(path)
	if err != nil {
		return map[string]string{}, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil {
		return map[string]string{}, err
	}

	vendors := map[string]string{}
	for index, record := range records {
		if index == 0 || len(record) < 3 {
			continue
		}
		prefix := strings.ToUpper(strings.TrimSpace(record[1]))
		vendor := strings.TrimSpace(record[2])
		if prefix != "" && vendor != "" && !strings.EqualFold(vendor, "PRIVATE") {
			vendors[prefix] = vendor
		}
	}
	return vendors, nil
}

func normalizedMACHex(mac string) string {
	var builder strings.Builder
	for _, r := range mac {
		switch {
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		case r >= 'a' && r <= 'f':
			builder.WriteRune(r - 'a' + 'A')
		case r >= 'A' && r <= 'F':
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

var embeddedOUIFallback = map[string]string{
	"00163E": "Xensource",
	"001C42": "Parallels",
	"005056": "VMware",
	"080027": "PCS Systemtechnik GmbH",
	"0A0027": "VirtualBox",
	"525400": "QEMU/KVM",
	"001A11": "Google",
	"F4F5D8": "Google",
	"3C5A37": "Samsung Electronics",
	"5C0A5B": "Samsung Electronics",
	"001788": "Philips Lighting",
	"B827EB": "Raspberry Pi Foundation",
	"DCA632": "Raspberry Pi Trading",
	"E45F01": "Raspberry Pi Trading",
	"18FE34": "Espressif",
	"246F28": "Espressif",
	"ACD074": "Espressif",
	"FCF5C4": "Espressif",
	"F4CF38": "TP-Link",
	"50C7BF": "TP-Link",
	"7483C2": "Ubiquiti Networks",
	"24A43C": "Ubiquiti Networks",
	"DC9FDB": "Ubiquiti Networks",
}

type inferredHotspotProfile struct {
	DeviceName string
	OSName     string
	Confidence int
}

func inferHotspotDeviceProfile(vendor, hostname string, fingerprint hotspotFingerprintResponse) inferredHotspotProfile {
	lowerVendor := strings.ToLower(vendor)
	lowerHost := strings.ToLower(hostname)
	lowerDHCPVendor := strings.ToLower(fingerprint.DHCPVendor)
	options := "," + fingerprint.DHCPFingerprint + ","

	switch {
	case strings.Contains(lowerDHCPVendor, "android") || strings.Contains(lowerHost, "android"):
		return profile("Dispositivo Android", "Android", 76)
	case strings.Contains(lowerDHCPVendor, "msft") || strings.Contains(options, ",44,") && strings.Contains(options, ",46,"):
		return profile("Computador Windows", "Windows", 72)
	case strings.Contains(lowerHost, "iphone") || strings.Contains(lowerHost, "ipad"):
		return profile("Dispositivo iOS", "iOS", 78)
	case strings.Contains(lowerHost, "macbook") || strings.Contains(lowerHost, "imac") || strings.Contains(lowerHost, "mac-"):
		return profile("Computador macOS", "macOS", 74)
	case strings.Contains(lowerVendor, "apple"):
		return profile("Dispositivo Apple", "iOS/macOS", 56)
	case strings.Contains(lowerDHCPVendor, "udhcp") || strings.Contains(lowerDHCPVendor, "busybox"):
		return profile("Dispositivo embarcado", "Linux/embarcado", 62)
	case strings.Contains(lowerDHCPVendor, "dhcpcd"):
		return profile("Dispositivo Unix-like", "Linux/Unix", 52)
	case isLikelyIoTVendor(lowerVendor):
		return profile("Dispositivo IoT/rede", "Linux/embarcado", 48)
	case vendor != "":
		return profile("Dispositivo de "+vendor, "", 36)
	default:
		return inferredHotspotProfile{}
	}
}

func profile(deviceName, osName string, confidence int) inferredHotspotProfile {
	return inferredHotspotProfile{DeviceName: deviceName, OSName: osName, Confidence: confidence}
}

func isLikelyIoTVendor(vendor string) bool {
	for _, token := range []string{
		"espressif", "tuya", "tp-link", "tplink", "zte", "huawei", "ubiquiti",
		"amazon", "google", "roku", "ring", "hikvision", "dahua", "sonoff",
		"xiaomi", "philips lighting", "shelly",
	} {
		if strings.Contains(vendor, token) {
			return true
		}
	}
	return false
}
