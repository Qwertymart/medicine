package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	_ "sync"
	"syscall"
	"time"

	"google.golang.org/grpc"

	pb "batch_grps/proto"
)

func main() {
	addr := flag.String("addr", "localhost:50051", "gRPC server address")
	devices := flag.String("devices", "", "comma-separated device IDs filter")
	types := flag.String("types", "", "comma-separated data types filter")
	flag.Parse()

	// 1. Подключаемся к gRPC
	conn, err := grpc.Dial(*addr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to dial: %v", err)
	}
	defer conn.Close()
	client := pb.NewCTGStreamServiceClient(conn)

	// 2. Собираем request
	req := &pb.StreamRequest{
		DeviceIds: splitAndTrim(*devices),
		DataTypes: splitAndTrim(*types),
	}

	// 3. Запрашиваем батчевый поток
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stream, err := client.StreamBatchCTGData(ctx, req)
	if err != nil {
		log.Fatalf("StreamBatchCTGData error: %v", err)
	}

	// 4. Читаем из потока и сразу печатаем каждый батч
	for {
		batchResp, err := stream.Recv()
		if err != nil {
			log.Printf("stream.Recv finished: %v", err)
			break
		}

		// Преобразуем в JSON для наглядности
		output := struct {
			Timestamp int64                 `json:"timestamp"`
			Count     int32                 `json:"count"`
			Data      []*pb.CTGDataResponse `json:"data"`
		}{
			Timestamp: batchResp.Timestamp,
			Count:     batchResp.Count,
			Data:      batchResp.Data,
		}
		b, _ := json.MarshalIndent(output, "", "  ")
		fmt.Printf("\n=== Received batch at %s ===\n%s\n",
			time.Unix(0, batchResp.Timestamp).Format(time.RFC3339), string(b))
	}

	// 5. Ожидаем сигнала CTRL+C
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
	log.Println("Batch subscriber exiting")
}

func splitAndTrim(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}
