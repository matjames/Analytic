package handlers

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"go-backend/configs"
	"go-backend/models"
	"go-backend/utils"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// Default password used for CSV-uploaded users (e.g. district). If a district user logs in with this, they must change password.
const defaultPassword = "biostat@2026"

// toSafeUser removes password from user object
func toSafeUser(u models.User) models.User {
	u.Password = ""
	u.PasswordHash = ""
	return u
}

// RegisterUser - POST /users/register - Register a new user
func RegisterUser(c *gin.Context) {
	var req struct {
		Role         *string `json:"role"`
		FirstName    *string `json:"first_name"`
		LastName     *string `json:"last_name"`
		Email        string  `json:"email" binding:"required"`
		Username     string  `json:"username" binding:"required"`
		Password     string  `json:"password" binding:"required"`
		Organisation *string `json:"organisation"`
		Phoneno      *string `json:"phoneno"`
		DistrictID   *string `json:"district_id"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email, username and password are required"})
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), 10)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	var user models.User
	err = configs.DB.QueryRow(
		`INSERT INTO users (role, first_name, last_name, email, username, password, organisation, phoneno, district_id, "createdAt", "updatedAt")
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
		 RETURNING id, first_name, last_name, username, email, role, organisation, phoneno, district_id, "createdAt", "updatedAt"`,
		req.Role, req.FirstName, req.LastName, req.Email, req.Username, string(hashed), req.Organisation, req.Phoneno, req.DistrictID,
	).Scan(&user.ID, &user.FirstName, &user.LastName, &user.Username, &user.Email, &user.Role, &user.Organisation, &user.Phoneno, &user.DistrictID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		// Check for duplicate key error (PostgreSQL error code 23505)
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "23505") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "email or username already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	user = toSafeUser(user)
	token, err := utils.SignUserToken(user.ID, user.Role, user.DistrictID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"user": user, "token": token})
}

// LoginUser - POST /users/login - Login with email or username
func LoginUser(c *gin.Context) {
	var req struct {
		EmailOrUsername string `json:"emailOrUsername" binding:"required"`
		Password        string `json:"password" binding:"required"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "emailOrUsername and password are required"})
		return
	}

	var user models.User
	err := configs.DB.QueryRow(
		`SELECT id, first_name, last_name, username, email, role, password, organisation, phoneno, district_id, "createdAt", "updatedAt"
		 FROM users WHERE email = $1 OR username = $1 LIMIT 1`,
		req.EmailOrUsername,
	).Scan(&user.ID, &user.FirstName, &user.LastName, &user.Username, &user.Email, &user.Role, &user.PasswordHash, &user.Organisation, &user.Phoneno, &user.DistrictID, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// District users logging in with the default password must change password (works for existing users and when DB flag is missing)
	roleStr := ""
	if user.Role != nil {
		roleStr = strings.TrimSpace(strings.ToLower(*user.Role))
	}
	if roleStr == "district" && req.Password == defaultPassword {
		user.MustChangePassword = true
		// Persist so /users/me and future logins reflect it (ignore error if column missing)
		_, _ = configs.DB.Exec("UPDATE users SET must_change_password = true WHERE id = $1", user.ID)
	}

	user = toSafeUser(user)
	token, err := utils.SignUserToken(user.ID, user.Role, user.DistrictID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user, "token": token})
}

// GetCurrentUser - GET /users/me - Get current user
func GetCurrentUser(c *gin.Context) {
	userID := c.GetInt64("user_id")

	var user models.User
	err := configs.DB.QueryRow(
		`SELECT id, first_name, last_name, username, email, role, organisation, phoneno, district_id, COALESCE(must_change_password, false), "createdAt", "updatedAt"
		 FROM users WHERE id = $1 LIMIT 1`,
		userID,
	).Scan(&user.ID, &user.FirstName, &user.LastName, &user.Username, &user.Email, &user.Role, &user.Organisation, &user.Phoneno, &user.DistrictID, &user.MustChangePassword, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	user = toSafeUser(user)
	c.JSON(http.StatusOK, gin.H{"user": user})
}

// ListUsers - GET /users - List all users
func ListUsers(c *gin.Context) {
	rows, err := configs.DB.Query(
		`SELECT id, first_name, last_name, username, email, role, organisation, phoneno, district_id, "createdAt", "updatedAt"
		 FROM users ORDER BY id ASC`,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(&user.ID, &user.FirstName, &user.LastName, &user.Username, &user.Email, &user.Role, &user.Organisation, &user.Phoneno, &user.DistrictID, &user.CreatedAt, &user.UpdatedAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		users = append(users, toSafeUser(user))
	}

	if err = rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, users)
}

// GetUser - GET /users/:id - Get user by id
func GetUser(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var user models.User
	err = configs.DB.QueryRow(
		`SELECT id, first_name, last_name, username, email, role, organisation, phoneno, district_id, "createdAt", "updatedAt"
		 FROM users WHERE id = $1`,
		id,
	).Scan(&user.ID, &user.FirstName, &user.LastName, &user.Username, &user.Email, &user.Role, &user.Organisation, &user.Phoneno, &user.DistrictID, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	user = toSafeUser(user)
	c.JSON(http.StatusOK, user)
}

// UpdateUser - PUT /users/:id - Update user
func UpdateUser(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		Role         *string `json:"role"`
		FirstName    *string `json:"first_name"`
		LastName     *string `json:"last_name"`
		Email        *string `json:"email"`
		Username     *string `json:"username"`
		Password     *string `json:"password"`
		Organisation *string `json:"organisation"`
		Phoneno      *string `json:"phoneno"`
		DistrictID   *string `json:"district_id"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Build dynamic update query
	var fields []string
	var values []interface{}
	idx := 1

	if req.Role != nil {
		fields = append(fields, "role = $"+strconv.Itoa(idx))
		if *req.Role == "" {
			values = append(values, nil)
		} else {
			values = append(values, *req.Role)
		}
		idx++
	}

	if req.FirstName != nil {
		fields = append(fields, "first_name = $"+strconv.Itoa(idx))
		if *req.FirstName == "" {
			values = append(values, nil)
		} else {
			values = append(values, *req.FirstName)
		}
		idx++
	}

	if req.LastName != nil {
		fields = append(fields, "last_name = $"+strconv.Itoa(idx))
		if *req.LastName == "" {
			values = append(values, nil)
		} else {
			values = append(values, *req.LastName)
		}
		idx++
	}

	if req.Email != nil {
		fields = append(fields, "email = $"+strconv.Itoa(idx))
		if *req.Email == "" {
			values = append(values, nil)
		} else {
			values = append(values, *req.Email)
		}
		idx++
	}

	if req.Username != nil {
		fields = append(fields, "username = $"+strconv.Itoa(idx))
		if *req.Username == "" {
			values = append(values, nil)
		} else {
			values = append(values, *req.Username)
		}
		idx++
	}

	if req.Password != nil && *req.Password != "" {
		hashed, err := bcrypt.GenerateFromPassword([]byte(*req.Password), 10)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
			return
		}
		fields = append(fields, "password = $"+strconv.Itoa(idx))
		values = append(values, string(hashed))
		idx++
	}

	if req.Organisation != nil {
		fields = append(fields, "organisation = $"+strconv.Itoa(idx))
		if *req.Organisation == "" {
			values = append(values, nil)
		} else {
			values = append(values, *req.Organisation)
		}
		idx++
	}

	if req.Phoneno != nil {
		fields = append(fields, "phoneno = $"+strconv.Itoa(idx))
		if *req.Phoneno == "" {
			values = append(values, nil)
		} else {
			values = append(values, *req.Phoneno)
		}
		idx++
	}

	if req.DistrictID != nil {
		fields = append(fields, "district_id = $"+strconv.Itoa(idx))
		values = append(values, *req.DistrictID)
		idx++
	}

	if len(fields) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	fields = append(fields, `"updatedAt" = NOW()`)
	values = append(values, id)

	query := `UPDATE users SET ` + strings.Join(fields, ", ") + ` WHERE id = $` + strconv.Itoa(idx) + ` RETURNING id, first_name, last_name, username, email, role, organisation, phoneno, district_id, "createdAt", "updatedAt"`

	var user models.User
	err = configs.DB.QueryRow(query, values...).Scan(&user.ID, &user.FirstName, &user.LastName, &user.Username, &user.Email, &user.Role, &user.Organisation, &user.Phoneno, &user.DistrictID, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "23505") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "email or username already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	user = toSafeUser(user)
	c.JSON(http.StatusOK, user)
}

// ChangePassword - POST /users/change-password - Change user password (requires old password verification)
func ChangePassword(c *gin.Context) {
	var req struct {
		EmailOrUsername string `json:"emailOrUsername" binding:"required"`
		OldPassword     string `json:"oldPassword" binding:"required"`
		NewPassword     string `json:"newPassword" binding:"required"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "emailOrUsername, oldPassword and newPassword are required"})
		return
	}

	if len(req.NewPassword) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "New password must be at least 6 characters long"})
		return
	}

	// Get user by email or username
	var user models.User
	err := configs.DB.QueryRow(
		`SELECT id, first_name, last_name, username, email, role, password, organisation, phoneno, district_id, COALESCE(must_change_password, false), "createdAt", "updatedAt"
		 FROM users WHERE email = $1 OR username = $1 LIMIT 1`,
		req.EmailOrUsername,
	).Scan(&user.ID, &user.FirstName, &user.LastName, &user.Username, &user.Email, &user.Role, &user.PasswordHash, &user.Organisation, &user.Phoneno, &user.DistrictID, &user.MustChangePassword, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or username"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Verify old password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Old password is incorrect"})
		return
	}

	// Hash new password
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), 10)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Update password and clear must_change_password
	err = configs.DB.QueryRow(
		`UPDATE users SET password = $1, must_change_password = false, "updatedAt" = NOW() WHERE id = $2
		 RETURNING id, first_name, last_name, username, email, role, organisation, phoneno, district_id, must_change_password, "createdAt", "updatedAt"`,
		string(hashed), user.ID,
	).Scan(&user.ID, &user.FirstName, &user.LastName, &user.Username, &user.Email, &user.Role, &user.Organisation, &user.Phoneno, &user.DistrictID, &user.MustChangePassword, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	user = toSafeUser(user)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Password changed successfully", "user": user})
}

// ResetPassword - POST /users/:id/reset-password - Reset user password
func ResetPassword(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		NewPassword string `json:"newPassword" binding:"required"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "New password is required"})
		return
	}

	if len(req.NewPassword) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Password must be at least 6 characters long"})
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), 10)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	var user models.User
	err = configs.DB.QueryRow(
		`UPDATE users SET password = $1, "updatedAt" = NOW() WHERE id = $2
		 RETURNING id, username, email, role, organisation, phoneno, district_id, "createdAt", "updatedAt"`,
		string(hashed), id,
	).Scan(&user.ID, &user.Username, &user.Email, &user.Role, &user.Organisation, &user.Phoneno, &user.DistrictID, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	user = toSafeUser(user)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Password reset successfully", "user": user})
}

// DeleteUser - DELETE /users/:id - Delete user
func DeleteUser(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	result, err := configs.DB.Exec("DELETE FROM users WHERE id = $1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// CreateUser - POST /users - Create a new user (authenticated users)
func CreateUser(c *gin.Context) {
	var req struct {
		Role         *string `json:"role"`
		FirstName    *string `json:"first_name"`
		LastName     *string `json:"last_name"`
		Email        string  `json:"email" binding:"required"`
		Username     string  `json:"username" binding:"required"`
		Password     string  `json:"password" binding:"required"`
		Organisation *string `json:"organisation"`
		Phoneno      *string `json:"phoneno"`
		DistrictID   *string `json:"district_id"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email, username and password are required"})
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), 10)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	var user models.User
	err = configs.DB.QueryRow(
		`INSERT INTO users (role, first_name, last_name, email, username, password, organisation, phoneno, district_id, "createdAt", "updatedAt")
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
		 RETURNING id, first_name, last_name, username, email, role, organisation, phoneno, district_id, "createdAt", "updatedAt"`,
		req.Role, req.FirstName, req.LastName, req.Email, req.Username, string(hashed), req.Organisation, req.Phoneno, req.DistrictID,
	).Scan(&user.ID, &user.FirstName, &user.LastName, &user.Username, &user.Email, &user.Role, &user.Organisation, &user.Phoneno, &user.DistrictID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		// Check for duplicate key error (PostgreSQL error code 23505)
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "23505") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "email or username already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	user = toSafeUser(user)
	// Return user without token (admin creates user, doesn't log them in)
	c.JSON(http.StatusCreated, gin.H{"user": user, "token": nil})
}

// UploadUsersCSV - POST /users/upload - Bulk upload users from CSV with default password
func UploadUsersCSV(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "CSV file is required"})
		return
	}

	f, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to open uploaded file"})
		return
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.TrimLeadingSpace = true

	// Read header
	header, err := reader.Read()
	if err != nil {
		if err == io.EOF {
			c.JSON(http.StatusBadRequest, gin.H{"error": "CSV file is empty"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read CSV header"})
		return
	}

	// Build column index map
	colIndex := map[string]int{}
	for i, col := range header {
		name := strings.ToLower(strings.TrimSpace(col))
		colIndex[name] = i
	}

	requiredCols := []string{"email", "username"}
	for _, col := range requiredCols {
		if _, ok := colIndex[col]; !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Missing required column '%s' in CSV header", col)})
			return
		}
	}

	// Helper to look up a column index by one of several possible header names
	lookupCol := func(names ...string) (int, bool) {
		for _, n := range names {
			if idx, ok := colIndex[strings.ToLower(n)]; ok {
				return idx, true
			}
		}
		return 0, false
	}

	// Optional columns (supporting multiple common header variants)
	firstNameIdx, hasFirstName := lookupCol("first_name", "firstname", "first name")
	lastNameIdx, hasLastName := lookupCol("last_name", "lastname", "last name")
	roleIdx, hasRole := lookupCol("role")
	orgIdx, hasOrg := lookupCol("organisation", "organization", "org", "organization name")
	phoneIdx, hasPhone := lookupCol("phoneno", "phone", "phone_number", "phone number")
	districtIdx, hasDistrict := lookupCol("district_id", "district", "districtid")

	// Pre-hash the default password once
	hashed, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), 10)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash default password"})
		return
	}

	type failure struct {
		Row      int    `json:"row"`
		Email    string `json:"email,omitempty"`
		Username string `json:"username,omitempty"`
		Error    string `json:"error"`
	}

	var (
		totalRows int
		created   int
		failed    []failure
	)

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		totalRows++

		if err != nil {
			failed = append(failed, failure{
				Row:   totalRows + 1, // +1 to account for header row
				Error: "Failed to read row: " + err.Error(),
			})
			continue
		}

		// basic bounds check
		if len(record) < len(header) {
			failed = append(failed, failure{
				Row:   totalRows + 1,
				Error: "Row has fewer columns than header",
			})
			continue
		}

		email := strings.TrimSpace(record[colIndex["email"]])
		username := strings.TrimSpace(record[colIndex["username"]])

		if email == "" || username == "" {
			failed = append(failed, failure{
				Row:      totalRows + 1,
				Email:    email,
				Username: username,
				Error:    "email and username are required",
			})
			continue
		}

		var (
			role       *string
			firstName  *string
			lastName   *string
			org        *string
			phone      *string
			districtID *string
		)

		if hasRole {
			val := strings.TrimSpace(record[roleIdx])
			if val != "" {
				role = &val
			}
		}
		if hasFirstName {
			val := strings.TrimSpace(record[firstNameIdx])
			if val != "" {
				firstName = &val
			}
		}
		if hasLastName {
			val := strings.TrimSpace(record[lastNameIdx])
			if val != "" {
				lastName = &val
			}
		}
		if hasOrg {
			val := strings.TrimSpace(record[orgIdx])
			if val != "" {
				org = &val
			}
		}
		if hasPhone {
			val := strings.TrimSpace(record[phoneIdx])
			if val != "" {
				phone = &val
			}
		}
		if hasDistrict {
			val := strings.TrimSpace(record[districtIdx])
			if val != "" {
				districtID = &val
			}
		}

		var user models.User
		// District users created with default password must change password on first login
		mustChange := role != nil && strings.EqualFold(*role, "district")
		var mustChangeVal interface{}
		if mustChange {
			mustChangeVal = true
		} else {
			mustChangeVal = false
		}

		err = configs.DB.QueryRow(
			`INSERT INTO users (role, first_name, last_name, email, username, password, organisation, phoneno, district_id, must_change_password, "createdAt", "updatedAt")
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())
			 RETURNING id, first_name, last_name, username, email, role, organisation, phoneno, district_id, must_change_password, "createdAt", "updatedAt"`,
			role, firstName, lastName, email, username, string(hashed), org, phone, districtID, mustChangeVal,
		).Scan(&user.ID, &user.FirstName, &user.LastName, &user.Username, &user.Email, &user.Role, &user.Organisation, &user.Phoneno, &user.DistrictID, &user.MustChangePassword, &user.CreatedAt, &user.UpdatedAt)

		if err != nil {
			msg := err.Error()
			if strings.Contains(msg, "duplicate key") || strings.Contains(msg, "23505") {
				msg = "email or username already exists"
			}
			failed = append(failed, failure{
				Row:      totalRows + 1,
				Email:    email,
				Username: username,
				Error:    msg,
			})
			continue
		}

		created++
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"total":   totalRows,
		"created": created,
		"failed":  len(failed),
		"errors":  failed,
	})
}
