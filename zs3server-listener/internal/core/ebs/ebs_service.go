package ebs

import (
	"os/exec"
)

type EBSService struct{}

func NewEBSService() *EBSService {
	return &EBSService{}
}

func (s *EBSService) IncreaseEBS() error {

	handle_ebs_increase("1", "2")

	// run the ebs_increase.sh script under /script
	cmd := exec.Command("./script/ebs_increase.sh", "1", "2") // example args: old size 1GB, new size 2GB
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}
