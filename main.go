package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

var (
	key []byte
	t   *jwt.Token
	s   string
)

type FormType int

const (
	Inbound FormType = iota
	Outbound
)

type FormShort struct {
	ID      string    `json:"id"`
	Company string    `json:"company"`
	Type    FormType  `json:"type"`
	Date    time.Time `json:"date"`
}
type FormDTO struct {
	ShortData FormShort `json:"shortData"`
	Data      string    `json:"youAreGayData"`
	Price     float64   `json:"price"`
}

var forms []FormDTO

func main() {
	key = []byte("key")
	forms = []FormDTO{
		{
			ShortData: FormShort{
				ID:      "tohle bude ičo + formnumber",
				Company: "SUS America, inc.",
				Type:    Inbound,
				Date:    time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC),
			},
			Data:  "Why are you gay",
			Price: 169.42,
		},
		{
			ShortData: FormShort{
				ID:      "350+69",
				Company: "Mateřská škola Čar a Kouzel v Pardubicích DELTA",
				Type:    Outbound,
				Date:    time.Date(1527, 2, 20, 0, 0, 0, 0, time.UTC),
			},
			Data:  "Skibidi Danda Švanda",
			Price: 999999,
		},
	}

	router := mux.NewRouter()
	corsMiddleware := handlers.CORS(
		handlers.AllowedOrigins([]string{"*"}),
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
		handlers.AllowedHeaders([]string{"Content-Type", "Authorization"}),
	)
	router.HandleFunc("/userToken", getUserToken).Methods("GET")
	router.Handle("/forms", jwtMiddleware(http.HandlerFunc(getForms))).Methods("GET")
	router.Handle("/form/{id}", jwtMiddleware(http.HandlerFunc(getFormById))).Methods("GET")
	router.Handle("/form/{id}/dto", jwtMiddleware(http.HandlerFunc(getFormDTOById))).Methods("GET")
	router.Handle("/form/{id}/pdf", jwtMiddleware(http.HandlerFunc(getFormPDFById))).Methods("GET")

	loggedRouter := handlers.LoggingHandler(os.Stdout, router)

	log.Fatal(http.ListenAndServe(":8080", corsMiddleware(loggedRouter)))
}

func jwtMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return key, nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func getUserToken(w http.ResponseWriter, r *http.Request) {
	t = jwt.NewWithClaims(jwt.SigningMethodHS256,
		jwt.MapClaims{
			"gay": true,
		},
	)

	s, _ = t.SignedString(key)
	fmt.Fprint(w, s)
}
func getForms(w http.ResponseWriter, r *http.Request) {

	resp := []string{}
	for _, form := range forms {
		resp = append(resp, form.ShortData.ID)
	}
	resp_j, _ := json.Marshal(resp)
	fmt.Fprint(w, string(resp_j))
}

func getFormById(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	for _, form := range forms {
		if form.ShortData.ID == id {
			resp_j, _ := json.Marshal(form.ShortData)
			fmt.Fprint(w, string(resp_j))
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprint(w, "No such form\n")
}

func getFormDTOById(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	for _, form := range forms {
		if form.ShortData.ID == id {
			resp_j, _ := json.Marshal(form)
			fmt.Fprint(w, string(resp_j))
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprint(w, "No such form\n")
}

func getFormPDFById(w http.ResponseWriter, r *http.Request) {
	filePath := "./files/report.pdf"

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "inline; filename=\"report.pdf\"") // use "attachment" to force download

	http.ServeFile(w, r, filePath)
}
