package handler

import (
	"fmt"
	"net/http"
)

type Order struct{}

func (o *Order) Create(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Created an order")
}

func (o *Order) List(w http.ResponseWriter, r *http.Request) {
	fmt.Println("List of all errors")
}

func (o *Order) GetById(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Getting by Id")
}

func (o *Order) UpdateById(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Updating by Id")
}

func (o *Order) DeleteById(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Deleting by Id")
}
