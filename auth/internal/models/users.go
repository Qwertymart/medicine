package models

type RegisterRequest struct {
	Name     string `json:"name" binding:"required,min=2,max=100"`
	LastName string `json:"last_name" binding:"required,min=2,max=100"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=100"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type User struct {
	ID            string `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	CountPictures uint   `gorm:"not null;default:0" json:"count_pictures"`
	Name          string `gorm:"not null;type:varchar(100)" json:"name"`
	LastName      string `gorm:"not null;type:varchar(100)" json:"last_name"`
	Email         string `gorm:"type:varchar(255);unique;not_null" json:"email"`
	PasswordHash  string `gorm:"column:password_hash;type:varchar(255);not null" json:"-"` // json:"-" !
}
