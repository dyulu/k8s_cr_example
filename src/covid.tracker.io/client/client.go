package client

import (
	"context"
	"os"

	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	covidv1alpha1 "covid.tracker.io/api/v1alpha1"
)

var name string

func init() {
	nodeId := "nodeid"
	if os.Getenv("NODE_ID") != "" {
		nodeId = os.Getenv("NODE_ID")
	}
	name = "coviddata-" + nodeId
}

func Create(c client.Client) error {
	pf := &covidv1alpha1.CovidData{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
	}

	err := c.Create(context.Background(), pf)

	return err
}

func Get(c client.Client) (covidv1alpha1.CovidData, error) {
	pf := &covidv1alpha1.CovidData{}
	err := c.Get(context.TODO(), client.ObjectKey{
		Name:      name,
		Namespace: "default",
	}, pf)

	if err != nil {
		err = Create(c)
		if err != nil {
			return *pf, err
		}

		err = c.Get(context.TODO(), client.ObjectKey{
			Name:      name,
			Namespace: "default",
		}, pf)
	}

	return *pf, err
}

func Connect() (client.Client, error) {
	var config *rest.Config
	var err error
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig != "" {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, err
		}
	} else {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
	}

	var c client.Client
	c, err = client.New(config, client.Options{})
	if err != nil {
		return nil, err
	}

	err = covidv1alpha1.AddToScheme(scheme.Scheme)
	return c, nil
}
