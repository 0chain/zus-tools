package datalake

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"zs3server-listener/internal/transport"
)

const (
	DatalakeServerBaseURL     = "https://datalake.blimp.software"
	RegisterZS3ServerEndpoint = "/clusters/zs3server/register"
	metadataFile              = "/var/lib/zs3/metadata.env"
)

var errMissingRequiredField = errors.New("missing required metadata field")

type DataLakeService struct {
	Transport *transport.HTTPTransport
}

func NewDataLakeService(baseURL string, timeout int) *DataLakeService {
	return &DataLakeService{
		Transport: transport.NewHTTPTransport("DataLakeService", baseURL, timeout),
	}
}

// parseEnvBytes parses simple KEY=VALUE lines into a map.
// Supports lines with optional surrounding quotes for values, ignores comments and blank lines.
func parseEnvBytes(b []byte) map[string]string {
	result := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(string(b)))
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// split only on first '='
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		// strip surrounding quotes if present
		if len(val) >= 2 {
			if (strings.HasPrefix(val, `"`) && strings.HasSuffix(val, `"`)) ||
				(strings.HasPrefix(val, `'`) && strings.HasSuffix(val, `'`)) {
				val = val[1 : len(val)-1]
			}
		}
		// remove any trailing CR (for CRLF files)
		val = strings.TrimSuffix(val, "\r")
		result[key] = val
	}
	// ignore scanner.Err() because we read from bytes; optionally handle it
	return result
}

// cat /var/lib/zs3/metadata.env
// BLOBBER_ID_LIST=47a9e33f6f12f4024e21cd9be9c969fb4a7cc19cbd99a82de16dcf6b70a8f3b9,2a0558f0b5a289744cd5d69dbc6c73a81247f6bfc840f6351996dfa3d36a3da2,8c7e01cdf612517e4ae4258bcd5225bcad0d187fcb6492521b1a953f6b458ae2
// CLIENT_ID=b6de36bff29cd8b300e67279c6c607f8f3583ef913151fbcef1883e15aec3920
// ALLOCATION_ID=26998b89aadbd8f89427b11c06847abc57579cf29fd00bd31edb1bc9066c6c9f
// MIN_WRITE_PRICE=0.001
// BLOBBER_PRIVATE_IPS=10.0.1.167,10.0.1.92,10.0.1.239
// BLOBBER_PUBLIC_IPS=15.165.236.133,3.38.103.103,3.34.189.63
// BLOBBER_HOSTNAMES=naveen-test-38-1.zus.network,naveen-test-38-2.zus.network,naveen-test-38-3.zus.network
// ZS3SERVER_HOSTNAME=naveen-test-38-0.zus.network
// EBS_VOLUME_SIZE=50
// ZS3SERVER_CLIENT_ID=b6de36bff29cd8b300e67279c6c607f8f3583ef913151fbcef1883e15aec3920
// CLUSTER_ID=default

func (s *DataLakeService) RegisterZS3Server() error {

	metadataBytes, err := os.ReadFile(metadataFile)
	if err != nil {
		return fmt.Errorf("read metadata file %s: %w", metadataFile, err)
	}

	reqMap := parseEnvBytes(metadataBytes)

	// Validate required fields (adjust keys as needed)
	// "BLOBBER_ID_LIST=%s\n"+
	// 		"CLIENT_ID=%s\n"+
	// 		"ALLOCATION_ID=%s\n"+
	// 		"MIN_WRITE_PRICE=%s\n"+
	// 		"BLOBBER_PRIVATE_IPS=%s\n"+
	// 		"BLOBBER_PUBLIC_IPS=%s\n"+
	// 		"BLOBBER_HOSTNAMES=%s\n"+
	// 		"ZS3SERVER_HOSTNAME=%s\n"+
	// 		"EBS_VOLUME_SIZE=%s\n"+
	// 		"ZS3SERVER_CLIENT_ID=%s\n"+
	// 		"CLUSTER_ID=%s\n",
	clientID := reqMap["CLIENT_ID"]
	allocationID := reqMap["ALLOCATION_ID"]
	minWritePrice := reqMap["MIN_WRITE_PRICE"]
	blobberPrivateIPs := reqMap["BLOBBER_PRIVATE_IPS"]
	blobberPublicIPs := reqMap["BLOBBER_PUBLIC_IPS"]
	blobberHostNames := reqMap["BLOBBER_HOSTNAMES"]
	zs3ServerHostName := reqMap["ZS3SERVER_HOSTNAME"]
	ebsVolumeSize := reqMap["EBS_VOLUME_SIZE"]
	zs3ServerClientID := reqMap["ZS3SERVER_CLIENT_ID"]
	clusterID := reqMap["CLUSTER_ID"]
	blobberList := reqMap["BLOBBER_ID_LIST"]
	serverIP := "" // choose appropriate IP from private/public based on your logic

	if clientID == "" {
		return fmt.Errorf("%w: CLIENT_ID is required", errMissingRequiredField)
	}

	if zs3ServerClientID != "" {
		clientID = zs3ServerClientID
	}

	if serverIP == "" {
		return fmt.Errorf("%w: SERVER_IP is required", errMissingRequiredField)
	}

	if clusterID == "" {
		return fmt.Errorf("%w: CLUSTER_ID is required", errMissingRequiredField)
	}

	var blobberIDs []string
	if strings.TrimSpace(blobberList) != "" {
		// split and trim each id
		parts := strings.Split(blobberList, ",")
		for _, p := range parts {
			if t := strings.TrimSpace(p); t != "" {
				blobberIDs = append(blobberIDs, t)
			}
		}
	}

	reqPayload := RegisterZS3ServerRequest{
		ZS3ServerClientID: clientID,
		ZS3ServerIP:       serverIP,
		ClusterID:         clusterID,
		BlobberIDs:        blobberIDs,
		AllocationID:      allocationID,
		ZS3ServerHostName: zs3ServerHostName,
		BlobberHostNames:  strings.Split(blobberHostNames, ","),
		MinWritePrice:     minWritePrice,
		BlobberPrivateIPs: strings.Split(blobberPrivateIPs, ","),
		BlobberPublicIPs:  strings.Split(blobberPublicIPs, ","),
		EbsVolumeSize:     ebsVolumeSize,
	}

	fmt.Println("Registering ZS3Server with DataLake:", reqPayload)

	payloadBytes, err := json.Marshal(reqPayload)
	if err != nil {
		return fmt.Errorf("marshal request payload: %w", err)
	}

	_, err = s.Transport.POSTRequest(RegisterZS3ServerEndpoint, payloadBytes)
	if err != nil {
		return fmt.Errorf("POST request failed: %w", err)
	}

	return nil
}
