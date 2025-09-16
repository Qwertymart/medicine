package models

type User struct {
	ID            string `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	CountPictures uint   `gorm:"not null;default:0" json:"count_pictures"`
	Name          string `gorm:"not null;type:varchar(100)" json:"name"`
	LastName      string `gorm:"not null;type:varchar(100)" json:"last_name"`
	Email         string `json:"email" gorm:"type:varchar(255);unique;not null"`
	PasswordHash  string `json:"-" gorm:"column:password_hash;type:varchar(255);not null"`
}
