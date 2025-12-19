package datalake

type RegisterZS3ServerRequest struct {
	ClusterID         string   `json:"cluster_id"`
	ZS3ServerClientID string   `json:"zs3_server_id"`
	ZS3ServerIP       string   `json:"zs3_server_ip"`
	ZS3ServerHostName string   `json:"zs3_server_hostname"`
	BlobberIDs        []string `json:"blobber_ids"`
	BlobberHostNames  []string `json:"blobber_hostnames"`
	AllocationID      string   `json:"allocation_id"`
	MinWritePrice     string   `json:"min_write_price"`
	BlobberPrivateIPs []string `json:"blobber_private_ips"`
	BlobberPublicIPs  []string `json:"blobber_public_ips"`
	EbsVolumeSize     string   `json:"ebs_volume_size"`
}

// DataLakeServicePort defines the methods your DataLake service must implement.
type DataLakeServicePort interface {
	// Define methods related to DataLake service here
	RegisterZS3Server(req RegisterZS3ServerRequest) error
}
