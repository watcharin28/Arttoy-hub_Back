package gcs

import (
	"context"
	"fmt"
	"log"
	"os"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

var Client *storage.Client

func  InitGCS(credentialJSON string) error {
	ctx := context.Background()

	// แปลง JSON เป็น []byte
	credBytes := []byte(credentialJSON)

	// ใช้ JSON จาก environment ในการสร้าง Client
	c, err := storage.NewClient(ctx, option.WithCredentialsJSON(credBytes))
	if err != nil {
		return fmt.Errorf("ไม่สามารถเชื่อมต่อ Google Cloud Storage: %v", err)
	}
	Client = c

	// ทดสอบ bucket
	bucketName := os.Getenv("GCS_BUCKET_NAME")
	if bucketName == "" {
		return fmt.Errorf("GCS_BUCKET_NAME ไม่ได้ตั้งค่าใน environment")
	}
	_, err = Client.Bucket(bucketName).Attrs(ctx)
	if err != nil {
		return fmt.Errorf("ไม่สามารถเข้าถึง bucket %s: %v", bucketName, err)
	}

	log.Printf("เชื่อมต่อ GCS bucket %s สำเร็จ", bucketName)
	return nil
}

func Close() {
	if Client != nil {
		Client.Close()
	}
}
