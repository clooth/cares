package main

import (
	"bufio"
	"database/sql"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

func Prompt(prompt string) (ret string) {
	input := bufio.NewReader(os.Stdin)
	os.Stdout.Write([]byte(prompt))
	for {
		line, isPrefix, err := input.ReadLine()
		if err != nil {
			return
		}
		ret += string(line)
		if !isPrefix {
			break
		}
	}
	ret = strings.TrimSpace(ret)
	return
}

func InitializeDatabase() {
	schemaBytes, err := ioutil.ReadFile("cares.sql")
	if err != nil {
		log.Println("Error reading database schema:", err.Error())
		return
	}

	statements := strings.Split(string(schemaBytes), ";\n")
	for _, statement := range statements {
		statement = strings.TrimSpace(statement)
		if statement == "" {
			continue
		}

		_, err = db.Exec(statement)
		if err != nil {
			log.Println("Error initializing database:", err.Error())
			return
		}
	}

	// Then make the owner record too.
	MakeAccount()
}

func MakeAccount() {
	name := Prompt("Name: ")
	pass := Prompt("Password: ")
	displayName := Prompt("Display name: ")

	account := NewAccount()
	account.Name = name
	account.DisplayName = displayName
	account.SetPassword(pass)
	err := account.Save()
	if err != nil {
		log.Println("Error saving new account:", err.Error())
	}
}

func ServeWeb() {
	err := LoadAccountForOwner()
	if err != nil {
		log.Println("Error loading site owner:", err.Error())
		return
	}

	http.HandleFunc("/static/", static)
	http.HandleFunc("/rss", rss)
	http.HandleFunc("/rssCloud", rssCloud)
	http.HandleFunc("/post", post)
	//http.HandleFunc("/archive/", archive)
	http.HandleFunc("/post/", permalink)
	http.HandleFunc("/", index)

	log.Println("Ohai web servin'")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func main() {
	var dsn string
	var makeaccount bool
	var initdb bool
	flag.StringVar(&dsn, "database", "dbname=cares sslmode=disable", "database connection info")
	flag.BoolVar(&initdb, "init-db", false, "initialize the database")
	flag.BoolVar(&makeaccount, "make-account", false, "create a new account interactively")
	flag.Parse()

	var err error
	db, err = sql.Open("postgres", dsn)
	if err == nil {
		// Try a query to make sure it worked.
		_, err = db.Query("SELECT 1")
	}
	if err != nil {
		log.Println("Error connecting to db:", err.Error())
		return
	}

	if initdb {
		InitializeDatabase()
	} else if makeaccount {
		MakeAccount()
	} else {
		ServeWeb()
	}
}
