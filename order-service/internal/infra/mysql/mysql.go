package mysql

import (
	"fmt"
	"order-service/internal/domain"
	"os"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

func NewMySQLFromEnv() (*gorm.DB, error) {
		user := os.Getenv("MYSQL_USER")       
	pass := os.Getenv("MYSQL_PASSWORD")  
	host := os.Getenv("MYSQL_HOST")       
	port := os.Getenv("MYSQL_PORT")       
	dbname := os.Getenv("MYSQL_DATABASE") 

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", user, pass, host, port, dbname)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: false,
		},
	})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&domain.Order{}); err != nil {
		return nil, err
	}

	return db, nil
}
