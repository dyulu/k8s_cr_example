package main

import (
	"context"
	"fmt"
	"os"

	"log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	covidv1alpha1 "covid.tracker.io/api/v1alpha1"
	coviddataclient "covid.tracker.io/client"

	"encoding/json"
	"github.com/gorilla/mux"
	"io/ioutil"
	"net/http"
)

func homePage(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Endpoint Hit: homePage")

	fmt.Fprintf(w, "Welcome to the page for covid data!\n")
	fmt.Fprintf(w, "COUNTRY: "+os.Getenv("COUNTRY"))
}

func publishCovidData(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Endpoint Hit: publishCovidData")

	log.Print("Connecting to k8s client ...")
	c, err := coviddataclient.Connect()
	if err != nil {
		log.Print("Could not connect: ", err.Error())
		os.Exit(1)
	}

	log.Print("Get the CovidData object ...")
	covidData, err := coviddataclient.Get(c)
	if err != nil {
		log.Print("Cannot get the CovidData object: ", err.Error())
		os.Exit(1)
	}

	reqBody, _ := ioutil.ReadAll(r.Body)
	var covidDataEntry covidv1alpha1.CovidDataEntry
	json.Unmarshal(reqBody, &covidDataEntry)
	covidDataEntry.ReportTime = metav1.Now()
	covidData.CovidDataEntries = append(covidData.CovidDataEntries, covidDataEntry)

	log.Print("Update the CovidData object ...")
	err = c.Update(context.TODO(), &covidData)
	if err != nil {
		log.Print("Cannot update the CovidData object: ", err.Error())
		os.Exit(1)
	}

	json.NewEncoder(w).Encode(covidDataEntry)
}

func listAllCovidData(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Endpoint Hit: listAllCovidData")

	log.Print("Connecting to k8s client ...")
	c, err := coviddataclient.Connect()
	if err != nil {
		log.Print("Could not connect: ", err.Error())
		os.Exit(1)
	}

	log.Print("Get the CovidData object ...")
	covidData, err := coviddataclient.Get(c)
	if err != nil {
		log.Print("Cannot get the CovidData object: ", err.Error())
		os.Exit(1)
	}

	json.NewEncoder(w).Encode(covidData.CovidDataEntries)
}

func clearAllCovidData(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Endpoint Hit: clearAllCovidData")

	log.Print("Connecting to k8s client ...")
	c, err := coviddataclient.Connect()
	if err != nil {
		log.Print("Could not connect: ", err.Error())
		os.Exit(1)
	}

	log.Print("Get the CovidData object ...")
	covidData, err := coviddataclient.Get(c)
	if err != nil {
		log.Print("Cannot get the CovidData object: ", err.Error())
		os.Exit(1)
	}

	covidData.CovidDataEntries = nil

	log.Print("Update the CovidData object ...")
	err = c.Update(context.TODO(), &covidData)
	if err != nil {
		log.Print("Cannot update the CovidData object: ", err.Error())
		os.Exit(1)
	}
}

func handleRequests() {
	covidRouter := mux.NewRouter().StrictSlash(true)
	covidRouter.HandleFunc("/", homePage)
	covidRouter.HandleFunc("/covid/data", publishCovidData).Methods("POST")
	covidRouter.HandleFunc("/covid/data/clear", clearAllCovidData).Methods("DELETE")
	covidRouter.HandleFunc("/covid/data/list", listAllCovidData)
	log.Fatal(http.ListenAndServe(":10000", covidRouter))
}

func main() {
	handleRequests()
}
