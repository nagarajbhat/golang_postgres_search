package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"code.launchyard.com/root/aircto-backend/logx"

	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
)

//change the db configration here
var (
	db          *gorm.DB
	DATA_SOURCE = "user=nagaraj dbname=blogps password=launchyard sslmode=disable"
)

//create all the models

type Applicant struct {
	Model
	CandidateId uint   `json:"candidate_id"`
	Candidate   *User  `json:"candidate,omitempty"`
	Status      string `json:"status" sql:"not null"`
}

type Model struct {
	Id        uint       `json:"id" sql:"primary_key"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at"`
}

type User struct {
	Model
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Password  string `json:"password,omitempty"`
}
type AppError struct {
	Status  int    `json:"status"`
	Message string `json:"error_message,omitempty"`
}

func NewAppError(status int, message string) *AppError {
	return &AppError{status, message}
}

func init() {
	var err error
	db, err = gorm.Open("postgres", DATA_SOURCE)
	if err != nil {
		log.Fatal(err)
	}
	db.AutoMigrate(&User{}, &Applicant{})

}

func main() {
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/search/", Search)
	router.HandleFunc("/setup/", Populate)
	router.HandleFunc("/teardown/", Teardown)
	log.Fatal(http.ListenAndServe(":8080", router))
}

//populate the db
func Populate(w http.ResponseWriter, r *http.Request) {
	applicants := []Applicant{
		{Candidate: &User{Email: "jedi@aircto.com", Password: "secret.", FirstName: "Jedi", LastName: "Yoda"}, Status: "Processed"},
		{Candidate: &User{Email: "chandler@launchyard.com", Password: "secret.", FirstName: "Chandler", LastName: "Bing"}, Status: "Processing"},
		{Candidate: &User{Email: "leonard@launchyard.com", Password: "secret.", FirstName: "Leonard", LastName: "hoff"}, Status: "Processed"},
		{Candidate: &User{Email: "ross@launchyard.com", Password: "secret.", FirstName: "Ross", LastName: "Geller"}, Status: "Processing"},
		{Candidate: &User{Email: "superman@marvel.com", Password: "secret.", FirstName: "Clark", LastName: "Kent"}, Status: "Processed"},
		{Candidate: &User{Email: "batman@dc.com", Password: "secret.", FirstName: "Bruce", LastName: "Wayne"}, Status: "Processed"},
		{Candidate: &User{Email: "monica@friends.com", Password: "secret.", FirstName: "Monica", LastName: "Geller"}, Status: "Processing"},
		{Candidate: &User{Email: "rachael@friends.com", Password: "secret.", FirstName: "Rachael", LastName: "Green"}, Status: "Processed"},
		{Candidate: &User{Email: "joey@friends.com", Password: "secret.", FirstName: "Joey", LastName: "Tribbiani"}, Status: "Processed"},
	}
	for i := range applicants {
		if err := db.Save(&applicants[i]).Error; err != nil {
			log.Fatal(err)
		}
	}
}

//teardown the db tables
func Teardown(w http.ResponseWriter, r *http.Request) {
	db.Delete(User{})
	db.Delete(Applicant{})
}

func Search(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	q := params.Get("q")
	var qlist []string
	qlist = strings.Fields(q)
	for i := range qlist {
		qlist[i] = qlist[i] + ":*"
		q = strings.Join(qlist[:], "|")
	}
	var applicants []Applicant

	if err := db.Joins("JOIN users ON users.id = applicants.candidate_id").Where("(to_tsvector(replace(users.email,'@','')) || to_tsvector(users.first_name) || to_tsvector(users.last_name) @@ to_tsquery(replace(?,'@','')))", q).
		Group("applicants.id").
		Preload("Candidate").
		Find(&applicants).Error; err != nil {
		log.Fatal("db error")

	}
	respond(w, r, http.StatusOK, applicants)
}

//helper function for reponse
func respond(w http.ResponseWriter, r *http.Request, status int, data interface{}) {
	var buf bytes.Buffer

	if err := json.NewEncoder(&buf).Encode(data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if status != http.StatusNoContent {
		if _, err := io.Copy(w, &buf); err != nil {
			logx.Error("respond.With.error: ", err)
		}
	}
}
