package main

import (
	"fmt"
	"os"
	"os/signal"

	fakes3 "github.com/bradenrayhorn/pickle/internal/fake_s3"
)

func main() {
	port := os.Getenv("FAKES3_HTTP_PORT")
	if len(port) == 0 {
		port = "7840"
	}

	host := os.Getenv("FAKES3_HTTP_HOST")

	bucket := os.Getenv("FAKES3_BUCKET")
	if bucket == "" {
		bucket = "my-bucket"
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	server := fakes3.NewFakeS3(bucket)
	server.StartServerWithHostPort(host, port)

	fmt.Printf("started fakeS3 at %s\n", server.GetEndpoint())

	<-c
	server.StopServer()
}
