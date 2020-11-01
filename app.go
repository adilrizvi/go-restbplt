package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"log"
	"net/http"
	"strconv"
)

type App struct {
	Router *mux.Router
	DB     *sql.DB
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJson(w, code, map[string]string{"error": message})

}

func respondWithJson(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "internal error")
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)

}

func (a *App) getProduct(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid Product ID")
		return
	}

	p := product{ID: id}
	if err := p.getProduct(a.DB); err != nil {
		switch err {

		case sql.ErrNoRows:
			respondWithError(w, http.StatusNotFound, "Product Not Found")
		default:
			respondWithError(w, http.StatusInternalServerError, err.Error())

		}

		return
	}

	respondWithJson(w, http.StatusOK, p)
}

func (a *App) getProducts(w http.ResponseWriter, r *http.Request) {
	var err error
	var count, start int

	count, _ = strconv.Atoi(r.FormValue("count"))
	start, _ = strconv.Atoi(r.FormValue("start"))

	if count > 10 || count < 1 {
		count = 10
	}

	if start < 0 {
		start = 0
	}

	var products []product
	products, err = getProducts(a.DB, start, count)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJson(w, http.StatusOK, products)
}

func (a *App) createProduct(w http.ResponseWriter, r *http.Request) {
	var p product

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&p); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid payload")
		return
	}

	defer r.Body.Close()

	if err := p.createProduct(a.DB); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJson(w, http.StatusCreated, p)
}

func (a *App) updateProduct(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid product id")
		return
	}

	log.Println(r.Body)

	var p product
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&p); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	log.Println(p)

	defer r.Body.Close()

	p.ID = id

	if err := p.updateProduct(a.DB); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJson(w, http.StatusOK, p)
}

func (a *App) deleteProduct(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid product id")
		return
	}

	p := product{ID:id}


	//TODO: deleteProduct should return 404 for non existent products
	if err := p.deleteProduct(a.DB); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJson(w, http.StatusOK, map[string]string{"result": "success"})

}

func (a *App) Initialize(user, password, dbname string) {
	connectionString := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", user, password, dbname)


	var err error
	a.DB, err = sql.Open("postgres", connectionString)
	if err != nil {
		log.Fatalf("error in creating db connection %s", err)
	}
	log.Println("DB Connection Created")

	a.Router = mux.NewRouter()
	a.initializeRoutes()

}

func (a *App) initializeRoutes() {
	a.Router.HandleFunc("/products", a.getProducts).Methods("GET")
	a.Router.HandleFunc("/products", a.createProduct).Methods("POST")
	a.Router.HandleFunc("/products/{id:[0-9]+}", a.getProduct).Methods("GET")
	a.Router.HandleFunc("/products/{id:[0-9]+}", a.updateProduct).Methods("PUT")
	a.Router.HandleFunc("/products/{id:[0-9]+}", a.deleteProduct).Methods("DELETE")
	a.Router.HandleFunc("/hc", healthCheck).Methods("GET")

}


func healthCheck( w http.ResponseWriter, r *http.Request) {
	respondWithJson(w, http.StatusOK, "OK")
}

func (a *App) Run(addr string) {
	log.Printf("starting server at :%s", addr)
	log.Fatal(http.ListenAndServe(addr, a.Router))

}
