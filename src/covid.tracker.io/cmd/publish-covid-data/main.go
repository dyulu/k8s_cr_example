package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	covidv1alpha1 "covid.tracker.io/api/v1alpha1"
	coviddataclient "covid.tracker.io/client"
)

func main() {
	logLevel := flag.String("logLevel", "info", "Optional, logging level: trace, debug, info, warn, error, fatal, panic")
	state := flag.String("state", "", "Required, name of the US State")
	covidCases := flag.Int("covidCases", 0, "Optional, number of covid cases")
	flag.Parse()

	if *state == "" {
		fmt.Println("Use -help option to get help.\nOption state required")
		os.Exit(1)
	}

	log := logrus.Logger{
		Out:   os.Stderr,
		Level: logrus.InfoLevel,
		Formatter: &logrus.TextFormatter{
			TimestampFormat: "2021 Jan 01 01:01:01",
			FullTimestamp:   true,
		},
	}

	parsedLevel, err := logrus.ParseLevel(*logLevel)
	if err != nil {
		fmt.Printf("Invalid log level: %s", *logLevel)
		os.Exit(1)
	}
	log.SetLevel(parsedLevel)

	log.Info("Connecting to k8s client ...")
	c, err := coviddataclient.Connect()
	if err != nil {
		log.Error("Could not connect: ", err.Error())
		os.Exit(1)
	}

	log.Info("Get the CovidData object ...")
	covidData, err := coviddataclient.Get(c)
	if err != nil {
		log.Error("Cannot get the CovidData object: ", err.Error())
		os.Exit(1)
	}

	// if covidData.CovidDataEntries == nil {
	// 	covidData.CovidDataEntries = make([]covidv1alpha1.CovidDataEntry, 30)
	// }
	covidDataEntry := covidv1alpha1.CovidDataEntry{
		State:      *state,
		CovidCases: *covidCases,
	}
	covidDataEntry.ReportTime = metav1.Now()
	log.Info("CovidDataEntry", covidDataEntry)
	covidData.CovidDataEntries = append(covidData.CovidDataEntries, covidDataEntry)

	log.Info("Update the CovidData object ...")
	err = c.Update(context.TODO(), &covidData)
	if err != nil {
		log.Error("Cannot update the CovidData object: ", err.Error())
		os.Exit(1)
	}
}
