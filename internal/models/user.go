package models

// User ...
type User struct {
	ID         uint					`gorm:"primeryKey"`
	Username   string				`gorm:"unique;not null"`
	Email      string				`gorm:"unique;not null"`
	Password   []byte				`gorm:"not null"`
}
