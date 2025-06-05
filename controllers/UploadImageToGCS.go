package controllers

import (
    "context"
    "fmt"
    "io"
    "log"
    "strings"
    "time"

    "github.com/google/uuid"
    "arttoy-hub/gcs"
)

// ใช้อัปโหลดภาพทั่วไป (เช่นโปรไฟล์ ฯลฯ)
func UploadImageToGCS(reader io.Reader, contentType, folder string) (string, error) {
    ctx := context.Background()
    bucket := "arttoy-profile-images"

    extension := "jpg"
    switch strings.ToLower(contentType) {
    case "image/png":
        extension = "png"
    case "image/jpeg", "image/jpg":
        extension = "jpeg"
    case "image/gif":
        extension = "gif"
    default:
        log.Printf("Unsupported content type: %s, defaulting to .jpg", contentType)
    }

    // ✅ ใช้ UUID + Nano timestamp เพื่อให้ชื่อไฟล์ไม่ซ้ำ
    objectName := fmt.Sprintf("%s/%s_%d.%s", folder, uuid.NewString(), time.Now().UnixNano(), extension)
    log.Printf("Uploading file to bucket: %s, object: %s", bucket, objectName)

    writer := gcs.Client.Bucket(bucket).Object(objectName).NewWriter(ctx)
    if contentType == "" {
        contentType = "image/jpeg"
        log.Printf("Content-Type not specified, defaulting to: %s", contentType)
    }
    writer.ContentType = contentType

    if _, err := io.Copy(writer, reader); err != nil {
        log.Printf("Failed to copy file to GCS: %v", err)
        return "", fmt.Errorf("ไม่สามารถคัดลอกไฟล์ไป GCS: %v", err)
    }

    if err := writer.Close(); err != nil {
        log.Printf("Failed to close writer: %v", err)
        return "", fmt.Errorf("ไม่สามารถปิด writer: %v", err)
    }

    publicURL := fmt.Sprintf("https://storage.googleapis.com/%s/%s", bucket, objectName)
    log.Printf("File uploaded successfully: %s", publicURL)
    return publicURL, nil
}

// ใช้อัปโหลดรูปสินค้า
func UploadProductImageToGCS(reader io.Reader, contentType, folder string) (string, error) {
    ctx := context.Background()
    bucket := "arttoy-profile-images"

    extension := "jpg"
    switch strings.ToLower(contentType) {
    case "image/png":
        extension = "png"
    case "image/jpeg", "image/jpg":
        extension = "jpeg"
    case "image/gif":
        extension = "gif"
    default:
        log.Printf("Unsupported content type: %s, defaulting to .jpg", contentType)
    }

    // ✅ ใช้ UUID + Nano timestamp
    objectName := fmt.Sprintf("%s/%s_%d.%s", folder, uuid.NewString(), time.Now().UnixNano(), extension)
    log.Printf("Uploading file to bucket: %s, object: %s", bucket, objectName)

    writer := gcs.Client.Bucket(bucket).Object(objectName).NewWriter(ctx)
    if contentType == "" {
        contentType = "image/jpeg"
        log.Printf("Content-Type not specified, defaulting to: %s", contentType)
    }
    writer.ContentType = contentType

    if _, err := io.Copy(writer, reader); err != nil {
        log.Printf("Failed to copy file to GCS: %v", err)
        return "", fmt.Errorf("ไม่สามารถคัดลอกไฟล์ไป GCS: %v", err)
    }

    if err := writer.Close(); err != nil {
        log.Printf("Failed to close writer: %v", err)
        return "", fmt.Errorf("ไม่สามารถปิด writer: %v", err)
    }

    publicURL := fmt.Sprintf("https://storage.googleapis.com/%s/%s", bucket, objectName)
    log.Printf("File uploaded successfully: %s", publicURL)
    return publicURL, nil
}
