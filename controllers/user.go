package controllers

import (
	"arttoy-hub/database"
	"arttoy-hub/models"
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"net/http"
	"strings"
    "log"
    "time"
)

// ฟังก์ชันอัปเดตโปรไฟล์
func UpdateProfile(c *gin.Context) {
	userID := c.GetString("user_id")
	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// รับข้อมูลจาก form-data
	username := c.PostForm("username")
	phonenumber := c.PostForm("phonenumber")
	gmail := c.PostForm("gmail") // เปลี่ยนจาก email เป็น gmail

	// ตรวจสอบความถูกต้องของ gmail
	if !strings.Contains(gmail, "@") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid gmail format"})
		return
	}

	var profileImageURL string
	file, header, err := c.Request.FormFile("profile_image")
	if err == nil {
		defer file.Close()

		// อ่าน content type
		contentType := header.Header.Get("Content-Type")
		if contentType == "" {
			contentType = "image/jpeg"
		}

		// อัปโหลดไป GCS
		profileImageURL, err = UploadImageToGCS(file, contentType, "profile")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to upload image: %v", err)})
			return
		}
	}

	update := bson.M{
		"$set": bson.M{
			"username":    username,
			"phonenumber": phonenumber,
			"gmail":       gmail,
		},
	}
	if profileImageURL != "" {
		update["$set"].(bson.M)["profile_image"] = profileImageURL
	}

	OpenCollection := db.OpenCollection("users")
	result, err := OpenCollection.UpdateOne(c.Request.Context(), bson.M{"_id": objID}, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to update profile: %v", err)})
		return
	}
	if result.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Update Successful"})
}

func GetProfile(c *gin.Context) {
	userID := c.GetString("user_id")
	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	OpenCollection := db.OpenCollection("users")
	var user models.User
	err = OpenCollection.FindOne(context.TODO(), bson.M{"_id": objID}).Decode(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user"})
		return
	}

	user.Password = "" // ไม่คืน password กลับ
	c.JSON(http.StatusOK, user)
}
func contains(slice []string, item string) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

func LikeProduct(c *gin.Context) {
    productId := c.Param("product_id")
    userID := c.GetString("user_id")

    // ตรวจสอบ userID
    if userID == "" {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "ไม่พบ ID ผู้ใช้ใน context"})
        return
    }

    // แปลงเป็น ObjectID
    objID, err := primitive.ObjectIDFromHex(userID)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "ID ผู้ใช้ไม่ถูกต้อง"})
        return
    }

    // ตรวจสอบ productId
    _, err = primitive.ObjectIDFromHex(productId)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "ID สินค้าไม่ถูกต้อง"})
        return
    }

    // ตั้งค่า context พร้อม timeout
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    // ตรวจสอบผู้ใช้
    var user models.User
    err = db.OpenCollection("users").FindOne(ctx, bson.M{"_id": objID}).Decode(&user)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "ไม่พบผู้ใช้"})
        return
    }

    // ตรวจสอบและตั้งค่า likedItems ถ้าเป็น null หรือไม่มี
    _, err = db.OpenCollection("users").UpdateOne(
        ctx,
        bson.M{
            "_id": objID,
            "$or": []bson.M{
                {"likedItems": nil},
                {"likedItems": bson.M{"$exists": false}},
            },
        },
        bson.M{"$set": bson.M{"likedItems": []string{}}},
    )
    if err != nil {
        log.Printf("ข้อผิดพลาดในการตั้งค่า likedItems: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "เกิดข้อผิดพลาดในการตั้งค่า likedItems"})
        return
    }

    // ตรวจสอบว่าไลก์แล้วหรือยัง
    alreadyLiked := contains(user.LikedItems, productId)

    if alreadyLiked {
        // ลบไลก์
        _, err := db.OpenCollection("users").UpdateOne(ctx, bson.M{"_id": objID},
            bson.M{"$pull": bson.M{"likedItems": productId}})
        if err != nil {
            log.Printf("ข้อผิดพลาดในการลบไลก์: %v", err)
            c.JSON(http.StatusInternalServerError, gin.H{"error": "เกิดข้อผิดพลาดในการลบไลก์"})
            return
        }
        c.JSON(http.StatusOK, gin.H{"message": "ลบไลก์สำเร็จ"})
    } else {
        // เพิ่มไลก์
        _, err := db.OpenCollection("users").UpdateOne(ctx, bson.M{"_id": objID},
            bson.M{"$addToSet": bson.M{"likedItems": productId}})
        if err != nil {
            log.Printf("ข้อผิดพลาดในการเพิ่มไลก์: %v", err)
            c.JSON(http.StatusInternalServerError, gin.H{"error": "เกิดข้อผิดพลาดในการเพิ่มไลก์"})
            return
        }
        c.JSON(http.StatusOK, gin.H{"message": "เพิ่มไลก์สำเร็จ"})
    }
}

func GetUserFavorites(c *gin.Context) {
    userID := c.GetString("user_id")
	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

    // ค้นหาผู้ใช้ในฐานข้อมูล
    var user models.User
    ctx := context.TODO()
    err = db.OpenCollection("users").FindOne(ctx, bson.M{"_id": objID}).Decode(&user)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
        return
    }
    likedItems := []primitive.ObjectID{}
    for _, item := range user.LikedItems {
        objID, err := primitive.ObjectIDFromHex(item)
        if err == nil {
            likedItems = append(likedItems, objID)
        }
    }
    fmt.Println("Liked Items:", likedItems)

    // ตรวจสอบว่าผู้ใช้ไม่มีสินค้าที่ชื่นชอบ
    if len(user.LikedItems) == 0 {
        c.JSON(http.StatusOK, []models.Product{})  // ส่งกลับรายการว่าง
        return
    }

    // ค้นหาผลิตภัณฑ์ที่มี _id ตรงกับ likedItems
    var products []models.Product
    cursor, err := db.OpenCollection("products").Find(ctx, bson.M{"_id": bson.M{"$in": likedItems}})
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error retrieving favorite products"})
        return
    }
    defer cursor.Close(ctx)  // ปิด cursor เมื่อไม่ใช้งานแล้ว

    // ดึงข้อมูลทั้งหมดจาก cursor
    if err := cursor.All(ctx, &products); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decoding favorite products"})
        return
    }

    // ส่งข้อมูล favorite products กลับ
    c.JSON(http.StatusOK, products)
}

func DeleteUserFavorite(c *gin.Context) {
    userID := c.GetString("user_id")
    productID := c.Param("product_id")

    // แปลง userID และ productID ให้เป็น ObjectID
    userObjID, err := primitive.ObjectIDFromHex(userID)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
        return
    }

    productObjID, err := primitive.ObjectIDFromHex(productID)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
        return
    }

    // ค้นหาผู้ใช้ในฐานข้อมูล
    var user models.User
    ctx := context.TODO()
    err = db.OpenCollection("users").FindOne(ctx, bson.M{"_id": userObjID}).Decode(&user)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
        return
    }

    // ลบ product จาก likedItems ของผู้ใช้
    var updatedLikedItems []primitive.ObjectID
    for _, likedItem := range user.LikedItems {
        likedItemObjID, err := primitive.ObjectIDFromHex(likedItem)
        if err != nil {
            continue // ถ้าแปลงไม่ได้ให้ข้ามไป
        }
        // ถ้าไม่ใช่ productID ที่ต้องการลบ
        if likedItemObjID != productObjID {
            updatedLikedItems = append(updatedLikedItems, likedItemObjID)
        }
    }

    // อัพเดทข้อมูลของผู้ใช้
    _, err = db.OpenCollection("users").UpdateOne(ctx, bson.M{"_id": userObjID}, bson.M{"$set": bson.M{"likedItems": updatedLikedItems}})
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating user liked items"})
        return
    }

    // ส่งการตอบกลับว่าอัพเดตสำเร็จ
    c.JSON(http.StatusOK, gin.H{"message": "Product removed from favorites"})
}
func GetFavoriteStatus(c *gin.Context) {
    productId := c.Param("product_id")
    userID := c.GetString("user_id")
 
    // ตรวจสอบ userID
    if userID == "" {
       c.JSON(http.StatusUnauthorized, gin.H{"error": "ไม่พบ ID ผู้ใช้ใน context"})
       return
    }
 
    objID, err := primitive.ObjectIDFromHex(userID)
    if err != nil {
       c.JSON(http.StatusBadRequest, gin.H{"error": "ID ผู้ใช้ไม่ถูกต้อง"})
       return
    }
 
    var user models.User
    err = db.OpenCollection("users").FindOne(context.Background(), bson.M{"_id": objID}).Decode(&user)
    if err != nil {
       c.JSON(http.StatusNotFound, gin.H{"error": "ไม่พบผู้ใช้"})
       return
    }
 
    // ตรวจสอบว่าไลก์แล้วหรือยัง
    alreadyLiked := contains(user.LikedItems, productId)
    c.JSON(http.StatusOK, gin.H{"liked": alreadyLiked})
 }
 