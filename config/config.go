package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var Session *gorm.DB

func GetSession() *gorm.DB {
	return Session
}

func ConnectToDB() {
	environment := os.Getenv("ENVIRONMENT")
	var db *gorm.DB
	var err error
	if environment == "development" {
		//local development DB config:::
		host := os.Getenv("HOST")
		username := os.Getenv("USER")
		password := os.Getenv("PASSWORD")
		port := os.Getenv("DBPORT")
		dbName := os.Getenv("DBNAME")

		dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable", host, username, password, dbName, port)

		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	} else {
		//production DB config:::
		connectionString := os.Getenv("CONNECTION_STRING")
		db, err = gorm.Open(postgres.Open(connectionString), &gorm.Config{})
	}

	if err != nil {
		log.Fatal(err)
	}

	Session = db.Session(&gorm.Session{SkipDefaultTransaction: true, PrepareStmt: false})
	if Session != nil {
		log.Print("success: created db session")
	}
}

func LoadEnv() {
	err := godotenv.Load()

	if err != nil {
		log.Print(err)
		log.Print("failed to load envirionment variables")
		// log.Fatal("failed to load environment variables")
	}
}
