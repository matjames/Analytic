package handlers

import (
	"net/http"
	"strconv"

	"go-backend/configs"
	"go-backend/models"

	"github.com/gin-gonic/gin"
)

func CreatePost(c *gin.Context) {
	var post models.Post

	if err := c.BindJSON(&post); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return
	}

	post.UserID = c.GetInt64("user_id")

	_, err := configs.DB.Exec(
		"INSERT INTO posts(user_id,title,content) VALUES($1,$2,$3)",
		post.UserID, post.Title, post.Content,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create post"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "post created"})
}

func GetPost(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var post models.Post

	err := configs.DB.QueryRow(
		"SELECT id,user_id,title,content FROM posts WHERE id=$1",
		id,
	).Scan(&post.ID, &post.UserID, &post.Title, &post.Content)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "post not found"})
		return
	}

	c.JSON(http.StatusOK, post)
}

func ListPosts(c *gin.Context) {
	// 1️⃣ Read query params
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	search := c.Query("search")

	offset := (page - 1) * limit

	// 2️⃣ Total count query
	var total int
	err := configs.DB.QueryRow(`
		SELECT COUNT(*)
		FROM posts
		WHERE ($1 = '' OR title ILIKE '%' || $1 || '%')
	`, search).Scan(&total)

	if err != nil {
		c.JSON(500, gin.H{"error": "count failed"})
		return
	}

	// 3️⃣ Data query
	rows, err := configs.DB.Query(`
		SELECT id,user_id,title,content
		FROM posts
		WHERE ($1 = '' OR title ILIKE '%' || $1 || '%')
		ORDER BY id DESC
		LIMIT $2 OFFSET $3
	`, search, limit, offset)

	if err != nil {
		c.JSON(500, gin.H{"error": "fetch failed"})
		return
	}

	var posts []models.Post
	for rows.Next() {
		var p models.Post
		rows.Scan(&p.ID, &p.UserID, &p.Title, &p.Content)
		posts = append(posts, p)
	}

	// 4️⃣ Response
	c.JSON(200, gin.H{
		"page":  page,
		"limit": limit,
		"total": total,
		"data":  posts,
	})
}

func UpdatePost(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	userID := c.GetInt64("user_id")

	var post models.Post
	c.BindJSON(&post)

	res, err := configs.DB.Exec(
		"UPDATE posts SET title=$1,content=$2 WHERE id=$3 AND user_id=$4",
		post.Title, post.Content, id, userID,
	)

	affected, _ := res.RowsAffected()
	if err != nil || affected == 0 {
		c.JSON(403, gin.H{"error": "not allowed"})
		return
	}

	c.JSON(200, gin.H{"message": "updated"})
}

func DeletePost(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	userID := c.GetInt64("user_id")

	res, err := configs.DB.Exec(
		"DELETE FROM posts WHERE id=$1 AND user_id=$2",
		id, userID,
	)

	affected, _ := res.RowsAffected()
	if err != nil || affected == 0 {
		c.JSON(403, gin.H{"error": "not allowed"})
		return
	}

	c.JSON(200, gin.H{"message": "deleted"})
}
