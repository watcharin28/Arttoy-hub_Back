package controllers

import (
	"arttoy-hub/database"
	"arttoy-hub/models"
	"context"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net/http"
	"strconv"
	"time"
	// "fmt"
)

type ReviewResponse struct {
	ID           primitive.ObjectID `json:"id"`
	UserName     string             `json:"userName"`
	Rating       int                `json:"rating"`
	Comment      string             `json:"comment"`
	Date         time.Time          `json:"date"`
	ProfileImage string             `json:"profileImage"`
}

type ProductDetailResponse struct {
	models.Product `json:",inline"`
	Images         []string `json:"images"`
	Seller         struct {
		ID   primitive.ObjectID `json:"id"`
		Name string `json:"name"`
	} `json:"seller"`
	Reviews []ReviewResponse `json:"reviews"`
}

func AddProduct(c *gin.Context) {
	var product models.Product

	product.Name = c.PostForm("name")
	product.Description = c.PostForm("description")
	product.Category = c.PostForm("category")
	product.Model = c.PostForm("model")
	product.Color = c.PostForm("color")
	product.Size = c.PostForm("size")
	price := c.PostForm("price")

	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	sellerObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	priceValue, err := strconv.ParseFloat(price, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid price"})
		return
	}

	product.Rating = 0.0

	// ตรวจสอบผู้ขาย
	var user models.User
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.OpenCollection("users").FindOne(ctx, bson.M{"_id": sellerObjID}).Decode(&user)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Seller not found"})
		return
	}

	if !user.IsSeller || user.SellerInfo == nil || !user.SellerInfo.IsVerified {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only verified sellers can post products"})
		return
	}

	product.Price = priceValue
	product.SellerID = sellerObjID
	product.IsSold = false

	// รูปภาพ
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid multipart form"})
		return
	}

	files := form.File["product_image"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No image files uploaded"})
		return
	}

	for _, file := range files {
		f, err := file.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open image"})
			return
		}
		defer f.Close()

		imageURL, err := UploadImageToGCS(f, file.Header.Get("Content-Type"), "product_images")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload image to GCS"})
			return
		}

		product.ImageURLs = append(product.ImageURLs, imageURL)
	}

	product.CreatedAt = time.Now()

	newProduct, err := models.AddProduct(product)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, newProduct)
}

func GetAllProducts(c *gin.Context) {
	products, err := models.GetAllProducts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, products)
}

func GetProductByID(c *gin.Context) {
	id := c.Param("id")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	productObjID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	var product models.Product
	err = db.ProductCollection.FindOne(ctx, bson.M{"_id": productObjID}).Decode(&product)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	var user models.User
	err = db.UserCollection.FindOne(ctx, bson.M{"_id": product.SellerID}).Decode(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Seller not found"})
		return
	}

	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	reviewCursor, err := db.ReviewCollection.Find(ctx, bson.M{"seller_id": product.SellerID}, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch reviews"})
		return
	}
	defer reviewCursor.Close(ctx)

	var reviews []ReviewResponse
	totalRating := 0

	for reviewCursor.Next(ctx) {
		var r models.Review
		if err := reviewCursor.Decode(&r); err != nil {
			continue
		}

		var reviewer models.User
		_ = db.UserCollection.FindOne(ctx, bson.M{"_id": r.UserID}).Decode(&reviewer)

		reviews = append(reviews, ReviewResponse{
			ID:           r.ID,
			UserName:     reviewer.Username,
			Rating:       r.Rating,
			Comment:      r.Comment,
			Date:         r.CreatedAt,
			ProfileImage: reviewer.ProfileImage,
		})

		totalRating += r.Rating
	}

	// ✅ คำนวณค่าเฉลี่ย Rating จากรีวิว
	averageRating := 0.0
	if len(reviews) > 0 {
		averageRating = float64(totalRating) / float64(len(reviews))
	}
	product.Rating = averageRating

	res := ProductDetailResponse{
		Product: product,
		Images:  product.ImageURLs,
		Seller: struct {
			ID   primitive.ObjectID `json:"id"`
			Name string             `json:"name"`
		}{
			ID:   user.ID,
			Name: user.Username,
		},
		Reviews: reviews,
	}

	c.JSON(http.StatusOK, res)
}


func UpdateProduct(c *gin.Context) {
	id := c.Param("id")

	// Parse multipart form
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse multipart form"})
		return
	}

	// ดึงค่าจาก form
	name := c.PostForm("name")
	description := c.PostForm("description")
	priceStr := c.PostForm("price")
	category := c.PostForm("category")
	model := c.PostForm("model")
	color := c.PostForm("color")
	size := c.PostForm("size")
	existingImagesJSON := c.PostForm("existing_images")

	// แปลง price เป็น float
	price, err := strconv.ParseFloat(priceStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid price"})
		return
	}

	// แปลง existing_images จาก JSON เป็น []string
	var existingImages []string
	if err := json.Unmarshal([]byte(existingImagesJSON), &existingImages); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid existing_images"})
		return
	}

	// รับไฟล์ใหม่ (product_image)
	form, _ := c.MultipartForm()
	files := form.File["product_image"]

	var newImageURLs []string
	for _, file := range files {
		f, err := file.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open image"})
			return
		}
		defer f.Close()

		// อัปโหลดจริง
		imageURL, err := UploadImageToGCS(f, file.Header.Get("Content-Type"), "product_images")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload image"})
			return
		}
		newImageURLs = append(newImageURLs, imageURL)
	}

	// รวมรูปทั้งหมด
	allImages := append(existingImages, newImageURLs...)

	// 🔥 ดึงสินค้าเดิมจาก DB เพื่อเอา seller_id เดิมกลับมา
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	var oldProduct models.Product
	err = db.OpenCollection("products").FindOne(ctx, bson.M{"_id": objID}).Decode(&oldProduct)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	// สร้าง struct ใหม่
	updated := models.Product{
		Name:        name,
		Description: description,
		Price:       price,
		Category:    category,
		Model:       model,
		Color:       color,
		Size:        size,
		ImageURLs:   allImages,
		SellerID:    oldProduct.SellerID, // ✅ ใส่ seller_id เดิมกลับเข้าไป
	}

	// แก้ไขใน DB
	product, err := models.UpdateProduct(id, updated)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, product)
}

func DeleteProduct(c *gin.Context) {
	id := c.Param("id")
	err := models.DeleteProduct(id)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Product deleted"})
}

func GetMyProducts(c *gin.Context) {
	userID := c.GetString("user_id") // ได้จาก JWT middleware
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// แปลง userID เป็น ObjectID
	sellerID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := db.OpenCollection("products")
	cursor, err := collection.Find(ctx, bson.M{"seller_id": sellerID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get products"})
		return
	}
	defer cursor.Close(ctx)

	var products []models.Product
	if err := cursor.All(ctx, &products); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error parsing product list"})
		return
	}

	c.JSON(http.StatusOK, products)
}
