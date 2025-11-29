package ebs

// EBSServicePort defines the methods your EBS service must implement.
type EBSServicePort interface {
	IncreaseEBS() error
}
