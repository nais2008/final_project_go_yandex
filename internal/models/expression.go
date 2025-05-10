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
	StorageID   uint     `gorm:"not null"`
	Storage     Storage  `gorm:"foreignKey:StorageID"`
}

// Task ...
type Task struct {
	ID            uint       `gorm:"primaryKey"`
	Arg1          float64    `gorm:"not null"`
	Arg2          *float64   `gorm:"default:null"`
	Operation     string     `gorm:"not null"`
	Status        string     `gorm:"not null"`
	Result        *float64   `gorm:"default:null"`
	OperationTime int        `gorm:"not null"`
	ExpressionID  uint     	 `gorm:"not null"`
	Expression    Expression `gorm:"foreignKey:ExpressionID"`
}

// Storage ...
type Storage struct {
	ID            uint         `gorm:"primaryKey"`
	Expressions   []Expression `gorm:"foreignKey:StorageID;constraint:OnDelete:CASCADE"`
	NextExprID    int          `gorm:"not null"`
	NextTaskID    int          `gorm:"not null"`
}
