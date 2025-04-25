package gcs
import (
    "context"
    "log"
    "cloud.google.com/go/storage"
)

var Client *storage.Client

func InitGCS() {
    ctx := context.Background()
    var err error
    Client, err = storage.NewClient(ctx)
    if err != nil {
        log.Fatalf("ไม่สามารถเชื่อมต่อ Google Cloud Storage: %v", err)
    }
    log.Println("เชื่อมต่อ Google Cloud Storage สำเร็จ")

    // ตัวอย่าง: ตรวจสอบ bucket
    bucketName := "arttoy-profile-images"
    _, err = Client.Bucket(bucketName).Attrs(ctx)
    if err != nil {
        log.Fatalf("ไม่สามารถเข้าถึง bucket %s: %v", bucketName, err)
    }
    log.Printf("Bucket %s พร้อมใช้งาน", bucketName)
}

func Close() {
    if Client != nil {
        Client.Close()
    }
}