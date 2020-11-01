package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
)

const tableCreationQuery = `CREATE TABLE IF NOT EXISTS products
(
    id SERIAL,
    name TEXT NOT NULL,
    price NUMERIC(10,2) NOT NULL DEFAULT 0.00,
    CONSTRAINT products_pkey PRIMARY KEY (id)
)`

var a App

func TestMain(m *testing.M) {
	a.Initialize(os.Getenv("APP_DB_USERNAME"), os.Getenv("APP_DB_PASSWORD"), os.Getenv("APP_DB_NAME"))
	//ensureTableExists()
	code := m.Run()
	clearTable()
	os.Exit(code)

}

func ensureTableExists() {
	if _, err := a.DB.Exec(tableCreationQuery); err != nil {
		log.Fatal(err)
	}
}

func clearTable() {
	a.DB.Exec("DELETE FROM products")
	a.DB.Exec("ALTER SEQUENCE products_id_seq RESTART WITH 1")
}

func executeRequest(req *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	a.Router.ServeHTTP(rr, req)

	return rr
}

func checkResponseCode(t *testing.T, expected, actual int) {
	if expected != actual {
		t.Errorf("Expected response code %d, Got %d", expected, actual)
	}
}
func TestEmptyTable(t *testing.T) {
	clearTable()

	req, _ := http.NewRequest("GET", "/products", nil)
	response := executeRequest(req)

	checkResponseCode(t, http.StatusOK, response.Code)
	fmt.Printf("body::%s\n", response.Body.String())
	if body := response.Body.String(); body != "null" {
		log.Println(body)
		t.Errorf("Expected an empty array, Got %s", body)
	}
}

func TestGetNonExistentProduct(t *testing.T) {
	clearTable()

	req, _ := http.NewRequest("GET", "/products/11", nil)
	response := executeRequest(req)

	checkResponseCode(t, http.StatusNotFound, response.Code)

	var m map[string]string
	err := json.Unmarshal(response.Body.Bytes(), &m)
	if err != nil {
		log.Printf("body::%s", response.Body.String())
		t.Errorf("error parsing response %v", response.Body.String())
	}
	if m["error"] != "Product Not Found" {
		t.Errorf("Expected the 'error' set to 'Product Not Found' , Got:%v", m["error"])
	}
}

func TestCreateProduct(t *testing.T) {
	clearTable()

	var jsonStr = []byte(`{"name": "test product", "price": 11.22}`)
	req, _ := http.NewRequest("POST", "/products", bytes.NewBuffer(jsonStr))
	req.Header.Set("Content-Type", "application/json")

	response := executeRequest(req)
	checkResponseCode(t, http.StatusCreated, response.Code)

	var m map[string]interface{}
	json.Unmarshal(response.Body.Bytes(), &m)

	if m["name"] != "test product" {
		t.Errorf("Expected Product Name 'test product', Got:%v", m["name"])

	}

	if m["price"] != 11.22 {
		t.Errorf("Expected price '11.22', Got:%v", m["price"])
	}

	if m["id"] != 1.0 {
		t.Errorf("Expected price '1.0', Got:%v", m["id"])
	}

}

func addProducts(count int) {
	if count < 1 {
		count = 1
	}

	for i := 0; i < count; i++ {
		a.DB.Exec("INSERT INTO products(name, price) VALUES($1, $2)", "Product "+strconv.Itoa(i), (i+1.0)*10)
	}
}

func TestFetchProduct(t *testing.T) {
	clearTable()
	addProducts(1)

	req, _ := http.NewRequest("GET", "/products/1", nil)
	response := executeRequest(req)

	checkResponseCode(t, http.StatusOK, response.Code)

}

func TestUpdateProduct(t *testing.T) {
	clearTable()
	addProducts(1)

	req, _ := http.NewRequest("GET", "/products/1", nil)
	response := executeRequest(req)

	var originalProduct map[string]interface{}
	json.Unmarshal(response.Body.Bytes(), &originalProduct)

	var jsonStr = []byte(`{"name": "test_update_product", "price": 11.22}`)
	req, _ = http.NewRequest("PUT", "/products/1", bytes.NewBuffer(jsonStr))
	response = executeRequest(req)

	checkResponseCode(t, http.StatusOK, response.Code)

	var m map[string]interface{}
	json.Unmarshal(response.Body.Bytes(), &m)

	if m["name"] != "test_update_product" {
		t.Errorf("Expected updated name:'test_updated_product' , Got:%v", m["name"])

	}
	if m["price"] != 11.22 {
		t.Errorf("Expected updated price:'11.22' , Got:%v", m["price"])

	}

	if m["id"] != originalProduct["id"] {
		t.Errorf("Expected Product ID:%v to be same,Got;%v", originalProduct["id"], m["id"])

	}

}

func TestDeleteProduct(t *testing.T) {
	clearTable()
	addProducts(1)

	req, _ := http.NewRequest("GET", "/products/1", nil)
	response := executeRequest(req)
	executeRequest(req)
	checkResponseCode(t, http.StatusOK, response.Code)

	req, _ = http.NewRequest("DELETE", "/products/1", nil)
	response = executeRequest(req)
	checkResponseCode(t, http.StatusOK, response.Code)

	req, _ = http.NewRequest("GET", "/products/1", nil)
	response = executeRequest(req)
	checkResponseCode(t, http.StatusNotFound, response.Code)

}
