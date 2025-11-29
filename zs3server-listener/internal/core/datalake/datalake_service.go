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
	DatalakeServerBaseURL     = "http://localhost:9090/api/v1/datalake"
	RegisterZS3ServerEndpoint = "/datalake/register_zs3server"
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

func (s *DataLakeService) RegisterZS3Server() error {

	metadataBytes, err := os.ReadFile(metadataFile)
	if err != nil {
		return fmt.Errorf("read metadata file %s: %w", metadataFile, err)
	}

	reqMap := parseEnvBytes(metadataBytes)

	// Validate required fields (adjust keys as needed)
	clientID := reqMap["CLIENT_ID"]
	serverIP := reqMap["SERVER_IP"]
	clusterID := reqMap["CLUSTER_ID"]
	blobberList := reqMap["BLOBBER_ID_LIST"]
	allocationID := reqMap["ALLOCATION_ID"]

	if clientID == "" {
		return fmt.Errorf("%w: CLIENT_ID is required", errMissingRequiredField)
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
	}

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
