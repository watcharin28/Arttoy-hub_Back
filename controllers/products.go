package controllers

import (
	"arttoy-hub/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
	"net/http"
	// "log"
	"fmt"
	// "io"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"strconv"
)

func AddProduct(c *gin.Context) {
	var product models.Product
	// รับข้อมูลทั่วไปจาก form-data
	product.Name = c.DefaultPostForm("name", "")
    product.Description = c.DefaultPostForm("description", "")
    price := c.DefaultPostForm("price", "")
    stock := c.DefaultPostForm("stock", "")
    categoryID := c.DefaultPostForm("category_id", "")
    rating := c.DefaultPostForm("rating", "")
	if err := c.ShouldBind(&product); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid form data"})
		return
	}
	priceValue, err := strconv.ParseFloat(price, 64)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid price"})
        return
    }

    stockValue, err := strconv.Atoi(stock)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid stock"})
        return
    }

    ratingValue, err := strconv.ParseFloat(rating, 64)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid rating"})
        return
    }

    // แปลง category_id ให้เป็น ObjectID
    objID, err := primitive.ObjectIDFromHex(categoryID)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category ID"})
        return
    }

    // ตั้งค่าในโครงสร้างสินค้า
    product.Price = priceValue
    product.Stock = stockValue
    product.CategoryID = objID
    product.Rating = ratingValue


	// รับไฟล์รูปภาพจาก request
	file, _, err := c.Request.FormFile("product_image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Product image not found"})
		return
	}
	fmt.Println("Received product:", product)
	// อัพโหลดรูปภาพไปยัง GCS
	imageURL, err := UploadImageToGCS(file, "image/jpeg", "product_images")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload image to GCS"})
		return
	}

	// เก็บ URL ของรูปภาพในข้อมูลสินค้า
	product.ImageURL = imageURL

	// เพิ่มสินค้าลงในฐานข้อมูล
	newProduct, err := models.AddProduct(product)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// ตอบกลับเป็นสินค้าใหม่ที่ถูกเพิ่มเข้าไป
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
	product, err := models.GetProductByID(id)
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

func UpdateProduct(c *gin.Context) {
	id := c.Param("id")
	var updatedProduct models.Product
	if err := c.ShouldBindJSON(&updatedProduct); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	product, err := models.UpdateProduct(id, updatedProduct)
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

