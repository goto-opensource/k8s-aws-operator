/*

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

package main

import (
	"errors"
	"flag"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	awsv1alpha1 "github.com/logmein/k8s-aws-operator/api/v1alpha1"
	"github.com/logmein/k8s-aws-operator/controllers"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	corev1.AddToScheme(scheme)
	awsv1alpha1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr, region, leaderElectionID, leaderElectionNamespace string
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&region, "region", "", "AWS region")
	flag.StringVar(&leaderElectionID, "leader-election-id", "k8s-aws-operator", "the name of the configmap do use as leader election lock")
	flag.StringVar(&leaderElectionNamespace, "leader-election-namespace", "", "the namespace in which the leader election lock will be held")
	flag.Parse()

	ctrl.SetLogger(zap.Logger(true))

	awsConfig := aws.NewConfig()

	if leaderElectionNamespace == "" {
		setupLog.Error(errors.New("-leader-election-namespace flag is required"), "command line flag validation failed")
		os.Exit(1)
	}

	if region != "" {
		awsConfig = awsConfig.WithRegion(region)
	}
	sess, err := session.NewSession(awsConfig)
	if err != nil {
		setupLog.Error(err, "unable to create AWS session")
		os.Exit(1)
	}

	ec2 := ec2.New(sess)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                  scheme,
		MetricsBindAddress:      metricsAddr,
		LeaderElection:          true,
		LeaderElectionNamespace: leaderElectionNamespace,
		LeaderElectionID:        leaderElectionID,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	err = (&controllers.EIPReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("EIP"),
		EC2:    ec2,
	}).SetupWithManager(mgr)
	if err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "EIP")
		os.Exit(1)
	}
	err = (&controllers.ENIReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("ENI"),
		EC2:    ec2,
	}).SetupWithManager(mgr)
	if err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ENI")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
