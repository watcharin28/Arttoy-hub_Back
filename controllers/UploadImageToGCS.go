package controllers

import (
    "context"
    "fmt"
    "io"
    "log"
    "strings"
    "time"
    "arttoy-hub/gcs"
	// "cloud.google.com/go/storage"
	
)

func UploadImageToGCS(reader io.Reader, contentType, folder string) (string, error) {
    ctx := context.Background()
    bucket := "arttoy-profile-images"

    // กำหนดนามสกุลไฟล์ตาม contentType
    extension := "jpg" // ค่าเริ่มต้น
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

    // สร้างชื่อไฟล์โดยใช้ timestamp
    objectName := fmt.Sprintf("%s/%d_image.%s", folder, time.Now().Unix(), extension)
    log.Printf("Uploading file to bucket: %s, object: %s", bucket, objectName)

    // สร้าง writer สำหรับ GCS
    writer := gcs.Client.Bucket(bucket).Object(objectName).NewWriter(ctx)
    if contentType == "" {
        contentType = "image/jpeg" // ค่าเริ่มต้นสำหรับรูปภาพ
        log.Printf("Content-Type not specified, defaulting to: %s", contentType)
    }
    writer.ContentType = contentType

    // คัดลอกข้อมูลจาก reader ไปยัง GCS
    if _, err := io.Copy(writer, reader); err != nil {
        log.Printf("Failed to copy file to GCS: %v", err)
        return "", fmt.Errorf("ไม่สามารถคัดลอกไฟล์ไป GCS: %v", err)
    }

    // ปิด writer
    if err := writer.Close(); err != nil {
        log.Printf("Failed to close writer: %v", err)
        return "", fmt.Errorf("ไม่สามารถปิด writer: %v", err)
    }

    // // ตั้งค่า object เป็น public
    // if err := gcs.Client.Bucket(bucket).Object(objectName).ACL().Set(ctx, storage.AllUsers, storage.RoleReader); err != nil {
    //     log.Printf("Failed to set object as public: %v", err)
    //     return "", fmt.Errorf("ไม่สามารถตั้งค่า object เป็น public: %v", err)
    // }

    // สร้าง URL สาธารณะ
    publicURL := fmt.Sprintf("https://storage.googleapis.com/%s/%s", bucket, objectName)
    log.Printf("File uploaded successfully: %s", publicURL)
    return publicURL, nil
}
// ฟังก์ชันนี้จะใช้สำหรับอัพโหลดรูปสินค้าขึ้น GCS
func UploadProductImageToGCS(reader io.Reader, contentType, folder string) (string, error) {
    ctx := context.Background()
    bucket := "arttoy-profile-images"  // กำหนด bucket สำหรับเก็บรูปสินค้า

    // กำหนดนามสกุลไฟล์ตาม contentType
    extension := "jpg" // ค่าเริ่มต้น
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

    // สร้างชื่อไฟล์โดยใช้ timestamp
    objectName := fmt.Sprintf("%s/%d_product_image.%s", folder, time.Now().Unix(), extension)
    log.Printf("Uploading file to bucket: %s, object: %s", bucket, objectName)

    // สร้าง writer สำหรับ GCS
    writer := gcs.Client.Bucket(bucket).Object(objectName).NewWriter(ctx)
    if contentType == "" {
        contentType = "image/jpeg" // ค่าเริ่มต้นสำหรับรูปภาพ
        log.Printf("Content-Type not specified, defaulting to: %s", contentType)
    }
    writer.ContentType = contentType

    // คัดลอกข้อมูลจาก reader ไปยัง GCS
    if _, err := io.Copy(writer, reader); err != nil {
        log.Printf("Failed to copy file to GCS: %v", err)
        return "", fmt.Errorf("ไม่สามารถคัดลอกไฟล์ไป GCS: %v", err)
    }

    // ปิด writer
    if err := writer.Close(); err != nil {
        log.Printf("Failed to close writer: %v", err)
        return "", fmt.Errorf("ไม่สามารถปิด writer: %v", err)
    }

    // สร้าง URL สาธารณะ
    publicURL := fmt.Sprintf("https://storage.googleapis.com/%s/%s", bucket, objectName)
    log.Printf("File uploaded successfully: %s", publicURL)
    return publicURL, nil
}