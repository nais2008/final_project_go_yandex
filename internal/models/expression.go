package models

// Expression ...
type Expression struct {
	ID          uint     `gorm:"primaryKey"`
	Expr        string   `gorm:"not null"`
	Status      string   `gorm:"not null"`
	Result      *float64 `gorm:"default:null"`
	UserID      uint     `gorm:"not null"`
	User        User     `gorm:"foreignKey:UserID"`
	Tasks       []Task   `gorm:"foreignKey:ExpressionID;constraint:OnDelete:CASCADE"`
}

// Task ...
type Task struct {
	ID            uint       `gorm:"primaryKey"`
	Arg1          float64    `gorm:"not null"`
	Arg2          *float64   `gorm:"default:null"`
	Operation     string     `gorm:"not null"`
	Status        string     `gorm:"not null;default:'pending'"`
	Result        *float64   `gorm:"default:null"`
	OperationTime int        `gorm:"not null"`
	Order         int        `gorm:"not null;default:0"`
	ExpressionID  uint     	 `gorm:"not null"`
	Expression    Expression `gorm:"foreignKey:ExpressionID"`
}
