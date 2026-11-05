package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/hibiken/asynq"
	"github.com/joho/godotenv"

	"api-students/app/model"
	"api-students/config"
	"api-students/database"
)

func main() {
	outboxID := flag.Int64("outbox", 0, "ID outbox yang akan di-enqueue ulang")
	copies := flag.Int("copies", 5, "Jumlah salinan task yang dikirim ke Redis")
	flag.Parse()

	if *outboxID <= 0 {
		fmt.Println("Gunakan: go run ./tools/redeliver -outbox <id> -copies <n>")
		os.Exit(1)
	}

	_ = godotenv.Load(filepath.Join(".", ".env"))
	config.LoadEnv()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := database.NewPool(ctx)
	if err != nil {
		log.Fatalf("Gagal terhubung database: %v", err)
	}
	defer pool.Close()

	var eventType string
	var payload []byte
	err = pool.QueryRow(ctx, "SELECT event_type, payload FROM outbox WHERE id = $1", *outboxID).
		Scan(&eventType, &payload)
	if err != nil {
		log.Fatalf("Gagal membaca outbox %d: %v", *outboxID, err)
	}

	job := model.NotificationJob{
		OutboxID:  *outboxID,
		EventType: eventType,
		Payload:   payload,
	}
	jobBytes, err := json.Marshal(job)
	if err != nil {
		log.Fatalf("Gagal menyusun payload job: %v", err)
	}

	client := asynq.NewClient(config.RedisOpt())
	defer client.Close()

	fmt.Printf("Mengirim %d salinan task outbox %d ke queue %s...\n", *copies, *outboxID, model.QueueNotifications)
	for i := 1; i <= *copies; i++ {
		// TaskID dibuat unik per salinan agar diterima oleh Redis tanpa ErrTaskIDConflict
		// namun membawa OutboxID yang sama agar menguji idempotency pada level worker
		task := asynq.NewTask(
			model.TaskSendNotification,
			jobBytes,
			asynq.Queue(model.QueueNotifications),
			asynq.TaskID(fmt.Sprintf("redeliver-%d-copy-%d-%d", *outboxID, i, time.Now().UnixNano())),
			asynq.MaxRetry(5),
			asynq.Timeout(30*time.Second),
			asynq.Retention(24*time.Hour),
		)

		info, err := client.EnqueueContext(ctx, task)
		if err != nil {
			log.Fatalf("Salinan ke-%d gagal di-enqueue: %v", i, err)
		}
		fmt.Printf("  [%d/%d] Enqueued task ID: %s\n", i, *copies, info.ID)
	}

	fmt.Println("Selesai! Seluruh salinan berhasil di-enqueue ke Redis.")
}
