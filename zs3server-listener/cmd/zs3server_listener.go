package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"zs3server-listener/internal/core/datalake"
	"zs3server-listener/internal/core/ebs"
	"zs3server-listener/internal/handler"
	"zs3server-listener/internal/router"
)

func InitLogging() {
	f, err := os.OpenFile("server.log",
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}

	// Log to both console and file
	mw := io.MultiWriter(os.Stdout, f)

	// Standard log
	log.SetOutput(mw)

	// Capture everything printed
	os.Stdout = f
	os.Stderr = f
}

func main() {

	InitLogging()

	log.Println("Starting ZS3Server Listener...")

	// datalake service
	datalakeService := datalake.NewDataLakeService(datalake.DatalakeServerBaseURL, 30)
	err := datalakeService.RegisterZS3Server()
	if err != nil {
		log.Fatalf("Failed to register ZS3Server with DataLake: %v", err)
	}
	log.Println("Successfully registered ZS3Server with DataLake.")

	ebsService := ebs.NewEBSService()

	zs3serverHandler := handler.NewZS3ServerHandler(ebsService)

	router := router.NewRouter("8081", zs3serverHandler)
	router.InitRouter()
	if err := router.Run(); err != nil {
		fmt.Println("Error starting server:", err)
	}
	defer router.Shutdown(context.Background())
}
