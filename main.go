package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"gopkg.in/ini.v1"
)

var (
	key      []byte
	t        *jwt.Token
	s        string
	dataFile = "forms.json"
	forms    []InvoiceType
	mu       sync.Mutex
	cfg      *ini.File
)

type IntrastatData struct {
	TariffCode      string `json:"tariffCode"`
	CountryOfOrigin string `json:"countryOfOrigin"`
}

type Item struct {
	Name        string        `json:"name"`
	Amount      float64       `json:"amount"`
	UnitPrice   float64       `json:"unitPrice"`
	TotalPrice  float64       `json:"totalPrice"`
	OrderNumber string        `json:"orderNumber"`
	Intrastat   IntrastatData `json:"intrastatData"`
}

type PairData struct {
	Ico  string `json:"IC"`
	Dico string `json:"DIC"`
}

type AccountingData struct {
	SupplierAccountNumber     string  `json:"supplierAccountNumber"`
	Currency                  string  `json:"currency"`
	IBAN                      string  `json:"iban"`
	Swift                     string  `json:"swift"`
	TotalAmount               float64 `json:"totalAmount"`
	TotalAmountPayingCurrency float64 `json:"totalAmountInPayingCurrency"`
	DPHPercent                float64 `json:"dphPercent"`
	DPHPayingCurrency         float64 `json:"dphPayingCurrency"`
	DPHCzk                    float64 `json:"dphCzk"`
	DPHBase                   float64 `json:"dphBaseCzk"`
	PaymentCircumstances      *string `json:"paymentCircumstances"`
	PaymentInstructions       *string `json:"paymentInstructions"`
	DueDate                   string  `json:"dueDate"`
	DUZPDate                  string  `json:"duzpDate"`
}
type FormDTO struct {
	InvoiceNum string         `json:"invoiceNum"`
	Storage    []Item         `json:"storage"`
	PairDatas  PairData       `json:"pairData"`
	Accounting AccountingData `json:"accountingData"`
}

type InvoiceTypeJson struct {
	Form FormDTO `json:"invoice"`
}
type InvoiceType struct {
	ID      string          `json:"id"`
	Invoice InvoiceTypeJson `json:"invoiceTypeJson"`
}

func txtToJsonInvoice(data string) (InvoiceType, error) {
	requestURL := fmt.Sprintf("http://%s:%s", cfg.Section("Microservices").Key("txtToJsonIp").String(), cfg.Section("Microservices").Key("txtToJsonPort").String())
	res, err := http.Post(requestURL, "application/text", bytes.NewBuffer([]byte(data)))
	if err != nil {
		return InvoiceType{}, err
	}
	defer res.Body.Close()

	form := &InvoiceTypeJson{}
	fmt.Print(res.Body)
	body_text, err := io.ReadAll(res.Body)
	if err != nil {
		return InvoiceType{}, err
	}
	derr := json.Unmarshal(body_text, form)
	if derr != nil {
		return InvoiceType{}, derr
	}
	return InvoiceType{ID: form.Form.PairDatas.Ico + form.Form.InvoiceNum, Invoice: *form}, nil
}
func pdfToTxtInvoice(filePath string) (string, error) {
	url := fmt.Sprintf("http://%s:%s", cfg.Section("Microservices").Key("pdfToTxtIp").String(), cfg.Section("Microservices").Key("pdfToTxtPort").String())
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return "", err
	}
	_, err = io.Copy(part, file)

	err = writer.Close()
	if err != nil {
		return "", err
	}

	request, err := http.NewRequest("POST", url, body)
	request.Header.Add("Content-Type", writer.FormDataContentType())
	if err != nil {
		return "", err
	}

	client := &http.Client{}
	response, err := client.Do(request)

	if err != nil {
		return "", err
	}
	if response.StatusCode != 200 {
		return "", errors.New("failed to convert PDF to text, status code: " + response.Status)
	}
	defer response.Body.Close()

	body_text, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	return string(body_text), nil
}
func loadData() error {
	file, err := os.Open(dataFile)
	if os.IsNotExist(err) {
		// No file yet, start with empty slice
		forms = []InvoiceType{}
		return nil
	} else if err != nil {
		return err
	}
	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	if len(bytes) == 0 {
		forms = []InvoiceType{}
		return nil
	}

	err = json.Unmarshal(bytes, &forms)
	if err != nil {
		return err
	}

	log.Printf("Loaded %d forms from file.\n", len(forms))
	return nil
}

// Save data to file (thread-safe)
func saveData() error {
	mu.Lock()
	defer mu.Unlock()

	bytes, err := json.MarshalIndent(forms, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(dataFile, bytes, 0644)
}

type ShortFormDto struct {
	Id         string `json:"id"`
	InvoiceNum string `json:"invoiceNum"`
	Amount     string `json:"totalAmount"`
}

func addForm(p InvoiceType) error {
	mu.Lock()
	forms = append(forms, p)
	mu.Unlock()

	log.Printf("Added: %+v\n", p)
	return saveData()
}
func loadConfig() error {
	con, err := ini.Load("configMain.ini")
	if err != nil {
		log.Printf("Generating default config file...")
		con = ini.Empty()
		con.NewSection("Microservices")
		con.Section("Microservices").NewKey("pdfToTxtIp", "pdfToTxt")

		con.Section("Microservices").NewKey("txtToJsonIp", "txtToJson")
		con.Section("Microservices").NewKey("txtToJsonPort", "5000")
		con.Section("Microservices").NewKey("pdfToTxtPort", "5001")
		err = con.SaveTo("configMain.ini")
		if err != nil {
			return errors.New("Failed to create new config " + err.Error())
		}
	}
	con, err = ini.Load("configMain.ini")
	if err != nil {
		return err
	}
	log.Printf("Config file loaded.")
	cfg = con
	return nil
}
func main() {
	key = []byte("key")
	err := loadConfig()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}
	err = loadData()
	if err != nil {
		log.Fatalf("Error loading data: %v", err)
	}

	router := mux.NewRouter()
	corsMiddleware := handlers.CORS(
		handlers.AllowedOrigins([]string{"*"}),
		handlers.AllowedMethods([]string{"GET", "POST"}), //, "PUT", "DELETE", "OPTIONS"}),
		handlers.AllowedHeaders([]string{"Content-Type", "Authorization"}),
	)
	router.HandleFunc("/userToken", getUserToken).Methods("GET")
	router.Handle("/forms", jwtMiddleware(http.HandlerFunc(getForms))).Methods("GET")
	router.Handle("/form/{id}", jwtMiddleware(http.HandlerFunc(getFormById))).Methods("GET")
	router.Handle("/form/{id}/dto", jwtMiddleware(http.HandlerFunc(getFormDTOById))).Methods("GET")
	router.Handle("/form/{id}/pdf", jwtMiddleware(http.HandlerFunc(getFormPDFById))).Methods("GET")
	router.Handle("/form/upload", jwtMiddleware(http.HandlerFunc(postUploadForm))).Methods("POST")

	loggedRouter := handlers.LoggingHandler(os.Stdout, router)

	log.Fatal(http.ListenAndServe(":8080", corsMiddleware(loggedRouter)))
}
func postUploadForm(w http.ResponseWriter, r *http.Request) {
	var buf bytes.Buffer
	// in your case file would be fileupload
	file, header, err := r.FormFile("file")
	if err != nil {
		panic(err)
	}
	defer file.Close()
	name := strings.Split(header.Filename, ".")
	log.Printf("File name %s\n", name[0])
	// Copy the file data to my buffer
	io.Copy(&buf, file)
	_, err = os.Create("files/" + header.Filename)
	if err != nil {
		log.Fatalf("Error creating file: %v", err)
	}
	pdfFile := os.WriteFile("files/"+header.Filename, buf.Bytes(), 0644)
	if pdfFile != nil {
		log.Fatalf("Error saving uploaded PDF file: %v", pdfFile)
	}
	tttxt, err := pdfToTxtInvoice("files/" + header.Filename)
	if err != nil {
		log.Printf("Error converting PDF to text: %v", err)
		return
	}
	fmt.Print(tttxt)
	retval, err := txtToJsonInvoice(tttxt)
	if err != nil {
		log.Printf("Error converting text to JSON invoice: %v", err)
		return
	}
	w.WriteHeader(http.StatusOK)
	os.Rename("files/"+header.Filename, "files/"+retval.ID+".pdf")
	addForm(retval)
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
		resp = append(resp, form.ID)
	}
	resp_j, _ := json.Marshal(resp)
	fmt.Fprint(w, string(resp_j))

}

func getFormById(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	for _, form := range forms {
		if form.ID == id {
			resp_j, _ := json.Marshal(form)
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
		if form.ID == id {
			resp_j, _ := json.Marshal(form)
			fmt.Fprint(w, string(resp_j))
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprint(w, "No such form\n")
}

func getFormPDFById(w http.ResponseWriter, r *http.Request) {
	for _, form := range forms {
		filePath := "files/" + form.ID + ".pdf"
		formFile, err := os.Open(filePath)
		if err != nil {
			continue
		}
		defer formFile.Close()
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", "inline; filename=\"report.pdf\"") // use "attachment" to force download

		http.ServeFile(w, r, filePath)
		return
	}
	filePath := "files/report.pdf"

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "inline; filename=\"report.pdf\"") // use "attachment" to force download

	http.ServeFile(w, r, filePath)
}
