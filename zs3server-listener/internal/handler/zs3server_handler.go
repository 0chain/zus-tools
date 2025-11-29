package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"zs3server-listener/internal/core/ebs"
)

// ZS3ServerHandlerPort defines the methods your zs3server handler must implement.
type ZS3ServerHandlerPort interface {
	HealthCheck(w http.ResponseWriter, r *http.Request)
	IncreaseEBS(w http.ResponseWriter, r *http.Request)
}

type ZS3ServerHandler struct {
	EBSService ebs.EBSServicePort
}

func NewZS3ServerHandler(ebsService ebs.EBSServicePort) *ZS3ServerHandler {
	return &ZS3ServerHandler{
		EBSService: ebsService,
	}
}

func (h *ZS3ServerHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {

	log.Println("HealthCheck endpoint called....")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	resp := map[string]interface{}{
		"status":  "healthy",
		"service": "ZS3Server",
		"code":    http.StatusOK,
	}

	log.Println("HealthCheck response:", resp)

	json.NewEncoder(w).Encode(resp)
}

func (h *ZS3ServerHandler) IncreaseEBS(w http.ResponseWriter, r *http.Request) {
	// Call the EBS service to increase EBS
	err := h.EBSService.IncreaseEBS()
	if err != nil {
		http.Error(w, "Failed to increase EBS: "+err.Error(), http.StatusInternalServerError)
		return
	}
	// Respond with success
	w.Write([]byte("EBS increase endpoint"))
}
