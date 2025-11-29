package datalake

type RegisterZS3ServerRequest struct {
	ZS3ServerClientID string   `json:"zs3_server_id"`
	ZS3ServerIP       string   `json:"zs3_server_ip"`
	ClusterID         string   `json:"cluster_id"`
	BlobberIDs        []string `json:"blobber_ids"`
	AllocationID      string   `json:"allocation_id"`
}

// DataLakeServicePort defines the methods your DataLake service must implement.
type DataLakeServicePort interface {
	// Define methods related to DataLake service here
	RegisterZS3Server(req RegisterZS3ServerRequest) error
}
