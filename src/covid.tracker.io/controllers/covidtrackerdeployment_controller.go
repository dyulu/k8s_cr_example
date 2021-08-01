/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	covidv1alpha1 "covid.tracker.io/api/v1alpha1"
)

var (
	DaemonSetNamePrefix = "covid"
	CovidContainerName  = "covid-data-api"
)

// CovidTrackerDeploymentReconciler reconciles a CovidTrackerDeployment object
type CovidTrackerDeploymentReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=covid.covid.tracker.io,resources=covidtrackerdeployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=covid.covid.tracker.io,resources=covidtrackerdeployments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=covid.covid.tracker.io,resources=covidtrackerdeployments/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the CovidTrackerDeployment object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *CovidTrackerDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logging := r.Log.WithValues("covidtrackerdeployment", req.NamespacedName)

	logging.Info("CovidTrackerDeploymentReconciler starts ...")

	deploymentCR := &covidv1alpha1.CovidTrackerDeployment{}
	err := r.Get(ctx, req.NamespacedName, deploymentCR)
	if err != nil {
		if errors.IsNotFound(err) {
			logging.Info("CovidTrackerDeployment CR not found")
			return ctrl.Result{}, nil
		}

		logging.Error(err, "Failed to get CovidTrackerDeployment CR")
		return ctrl.Result{}, err
	}

	err = r.reconcileCovidDaemonSet(ctx, deploymentCR)
	if err != nil {
		logging.Error(err, "Failed to update deploy covid daemonset")
		return ctrl.Result{}, err
	}

	deploymentCR.Status.CurrentControlPlaneVersion = deploymentCR.Spec.CurrentControlPlaneVersion
	err = r.Status().Update(ctx, deploymentCR)
	if err != nil {
		logging.Error(err, "Failed to update CovidTrackerDeployment CR status")
		return ctrl.Result{}, err
	}

	logging.Info("CovidTrackerDeploymentReconciler done")

	return ctrl.Result{}, nil
}

func (r *CovidTrackerDeploymentReconciler) reconcileCovidDaemonSet(ctx context.Context, deploymentCR *covidv1alpha1.CovidTrackerDeployment) error {
	var err error
	log := r.Log.WithValues(
		"covidtrackerdeployment", deploymentCR.Name,
		"error", err,
	)

	daemonSetName := DaemonSetNamePrefix + deploymentCR.Name

	labelSet := map[string]string{
		"app": daemonSetName,
	}
	privileged := false
	vols := []corev1.Volume{}
	hpType := corev1.HostPathType(corev1.HostPathDirectoryOrCreate)
	vols = append(vols, corev1.Volume{
		Name: "covid-data",
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: "/covid-data",
				Type: &hpType,
			},
		},
	})

	dsObject := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      daemonSetName,
			Namespace: deploymentCR.Namespace,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labelSet,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      daemonSetName,
					Namespace: deploymentCR.Namespace,
					Labels:    labelSet,
				},

				Spec: corev1.PodSpec{
					ServiceAccountName: "default",
					HostNetwork:        true,

					Containers: []corev1.Container{
						{
							Name:  CovidContainerName,
							Image: deploymentCR.Spec.Images.CovidDataAPI,
							// ImagePullPolicy: "IfNotPresent", // One of Always, Never, IfNotPresent
							// export IMAGE_PULL_POLICY=IfNotPresent
							ImagePullPolicy: corev1.PullPolicy(os.Getenv("IMAGE_PULL_POLICY")),
							SecurityContext: &corev1.SecurityContext{
								Privileged: &privileged,
							},
							Env: []corev1.EnvVar{
								{
									Name: "COUNTRY",
									// Value: "USA",
									// kubectl create configmap covid-data-country --from-literal=country=USA
									// OR
									// kubectl create configmap covid-data-country --from-env-file covid-data-country.properties
									// File covid-data-country.properties contents: country=USA
									ValueFrom: &corev1.EnvVarSource{
										ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "covid-data-country",
											},
											Key:      "country",
											Optional: new(bool), // Default to false
										},
									},
									// ValueFrom: &corev1.EnvVarSource{
									//    FieldRef: &corev1.ObjectFieldSelector{
									//        FieldPath: "spec.nodeName",
									//    },
									//},
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "covid-data",
									MountPath: "/covid-data",
								},
							},
						},
					},
					Volumes: vols,
				},
			},
		},
	}

	// Check if the daemonset already exists in the specified namespace
	found := &appsv1.DaemonSet{}
	err = r.Get(ctx, types.NamespacedName{Name: daemonSetName, Namespace: deploymentCR.Namespace}, found)
	if err != nil {
		if errors.IsNotFound(err) {
			// Daemonset does not exist -- create it
			log.Info("Creating covid daemonset")

			// Set deployment CR instance as the owner and controller
			ctrl.SetControllerReference(deploymentCR, dsObject, r.Scheme)

			err = r.Create(ctx, dsObject)
			if err != nil {
				return fmt.Errorf("Unexpected error creating covid daemonset: %v", err)
			}

			log.Info("Created covid daemonset")
		} else {
			return fmt.Errorf("Unexpected error reterving covid daemonset: %v", err)
		}
	} else {
		log.Info("Updating covid daemonset")
		// Daemonset exists -- update it
		// err = r.Update(ctx, dsObject)
		err = r.Update(ctx, found)
		if err != nil {
			return fmt.Errorf("Unexpected error updating covid daemonset: %v", err)
		}

		log.Info("Updated covid daemonset")
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CovidTrackerDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&covidv1alpha1.CovidTrackerDeployment{}).
		Complete(r)
}
