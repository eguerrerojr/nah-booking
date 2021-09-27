package main

import (
	"encoding/gob"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/eguerrerojr/nah-booking/internal/driver"
	"github.com/eguerrerojr/nah-booking/internal/helpers"

	"github.com/eguerrerojr/nah-booking/internal/config"
	"github.com/eguerrerojr/nah-booking/internal/handlers"
	"github.com/eguerrerojr/nah-booking/internal/models"
	"github.com/eguerrerojr/nah-booking/internal/render"
	"github.com/alexedwards/scs/v2"
)

const port = ":8080"

var app config.AppConfig
var session *scs.SessionManager
var infoLog *log.Logger
var errorLog *log.Logger

func main() {
	db, err := run()
	if err != nil {
		log.Fatal(err)
	}
	defer db.SQL.Close()

	defer close(app.MailChan)

	srv := &http.Server{
		Addr:    port,
		Handler: routes(&app),
	}

	log.Printf("Server is listening on port %s\n", port)

	err = srv.ListenAndServe()
	log.Fatal(err)
}

func run() (*driver.DB, error) {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(dir)

	// what am I going to put in the session
	gob.Register(models.Reservation{})
	gob.Register(models.User{})
	gob.Register(models.Room{})
	gob.Register(models.Restriction{})

	mailChan := make(chan models.MailData)
	app.MailChan = mailChan

	app.InProd = false

	infoLog = log.New(os.Stdout, "ERROR\t", log.Ldate|log.Ltime)
	app.InfoLog = infoLog

	errorLog = log.New(os.Stdout, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)
	app.ErrorLog = errorLog

	session = scs.New()
	session.Lifetime = 24 * time.Hour
	session.Cookie.Persist = true
	session.Cookie.SameSite = http.SameSiteLaxMode
	session.Cookie.Secure = app.InProd

	app.Session = session

	// connect to database
	log.Println("Connecting to database")
	db, err := driver.ConnectSQL("host=localhost port=5432 dbname=bookings user=postgres password=212121")
	if err != nil {
		log.Fatal("Cannot connect to DB!!")
	}
	log.Println("Connected to DB")

	tc, err := render.CreateTemplateCache()
	if err != nil {
		log.Fatal("Cannot creat template cache")
		return nil, err
	}

	app.TemplateCache = tc
	app.UseCache = app.InProd

	repo := handlers.NewRepo(&app, db)
	handlers.NewHandler(repo)
	render.NewRenderer(&app)
	helpers.NewHelpers(&app)

	return db, nil
}
