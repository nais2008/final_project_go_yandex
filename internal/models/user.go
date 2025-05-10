package models

// User ...
type User struct {
	ID         uint					`gorm:"primeryKey"`
	Username   string				`gorm:"unique;not null"`
	Email      string				`gorm:"unique;not null"`
	Password   string				`gorm:"not null"`
	Expression []Expression `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}
