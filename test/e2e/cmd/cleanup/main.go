/*
Copyright 2026.

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
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	servicesv1alpha1 "github.com/opendatahub-io/opendatahub-db-operator/api/services/v1alpha1"
	modulemanager "github.com/opendatahub-io/opendatahub-db-operator/pkg/manager"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	e2eOwnerLabel = "shiftleft.opendatahub.io/owner"
	e2eOwnerValue = "rhai-db-operator"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("expected prepare-namespace, database-service, or namespace cleanup mode")
	}

	switch args[0] {
	case "database-service":
		return cleanupDatabaseService()
	case "prepare-namespace":
		name, err := namespaceName(args[1:])
		if err != nil {
			return err
		}
		return prepareNamespace(name)
	case "namespace":
		name, err := namespaceName(args[1:])
		if err != nil {
			return err
		}
		return cleanupNamespace(name)
	default:
		return fmt.Errorf("unknown cleanup mode %q", args[0])
	}
}

func cleanupDatabaseService() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	kubeClient, err := newClient()
	if err != nil {
		return err
	}

	return cleanupDatabaseServiceWithClient(ctx, kubeClient)
}

func cleanupDatabaseServiceWithClient(ctx context.Context, kubeClient client.Client) error {
	key := client.ObjectKey{Name: servicesv1alpha1.DatabaseServiceInstanceName}
	instance := &servicesv1alpha1.DatabaseService{}
	if err := kubeClient.Get(ctx, key, instance); isGone(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("getting DatabaseService singleton: %w", err)
	}
	if instance.Labels[e2eOwnerLabel] != e2eOwnerValue {
		fmt.Fprintln(os.Stderr, "leaving DatabaseService singleton because it is not owned by the shift-left test")
		return nil
	}

	uid := instance.UID
	if err := kubeClient.Delete(ctx, instance, client.Preconditions{UID: &uid}); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("deleting DatabaseService singleton: %w", err)
	}

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		current := &servicesv1alpha1.DatabaseService{}
		err := kubeClient.Get(ctx, key, current)
		if isGone(err) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("waiting for DatabaseService singleton deletion: %w", err)
		}
		if current.UID != uid {
			return fmt.Errorf("DatabaseService singleton was replaced during cleanup; refusing to delete the replacement")
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for DatabaseService singleton deletion: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}

func prepareNamespace(name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	kubeClient, err := newClient()
	if err != nil {
		return err
	}

	namespace := &corev1.Namespace{}
	err = kubeClient.Get(ctx, client.ObjectKey{Name: name}, namespace)
	if apierrors.IsNotFound(err) {
		namespace = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
				Labels: map[string]string{
					e2eOwnerLabel: e2eOwnerValue,
				},
			},
		}
		if err := kubeClient.Create(ctx, namespace); err != nil {
			return fmt.Errorf("creating e2e namespace %q: %w", name, err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("getting namespace %q: %w", name, err)
	}
	if namespace.Labels[e2eOwnerLabel] != e2eOwnerValue {
		return fmt.Errorf("refusing to use existing namespace %q because it is not owned by the shift-left test", name)
	}
	if namespace.DeletionTimestamp != nil {
		return fmt.Errorf("namespace %q is already being deleted", name)
	}
	return nil
}

func cleanupNamespace(name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	kubeClient, err := newClient()
	if err != nil {
		return err
	}

	return cleanupNamespaceWithClient(ctx, kubeClient, name)
}

func cleanupNamespaceWithClient(ctx context.Context, kubeClient client.Client, name string) error {
	namespace := &corev1.Namespace{}
	if err := kubeClient.Get(ctx, client.ObjectKey{Name: name}, namespace); isGone(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("getting namespace %q: %w", name, err)
	}
	if namespace.Labels[e2eOwnerLabel] != e2eOwnerValue {
		return fmt.Errorf("refusing to delete namespace %q because it is not owned by the shift-left test", name)
	}

	uid := namespace.UID
	if err := kubeClient.Delete(
		ctx,
		namespace,
		client.Preconditions{UID: &uid},
	); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("deleting namespace %q: %w", name, err)
	}

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		current := &corev1.Namespace{}
		err := kubeClient.Get(ctx, client.ObjectKey{Name: name}, current)
		if isGone(err) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("waiting for namespace %q deletion: %w", name, err)
		}
		if current.UID != uid {
			return fmt.Errorf("namespace %q was replaced during cleanup; refusing to delete the replacement", name)
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for namespace %q deletion: %w", name, ctx.Err())
		case <-ticker.C:
		}
	}
}

func namespaceName(args []string) (string, error) {
	flags := flag.NewFlagSet("namespace", flag.ContinueOnError)
	name := flags.String("name", "", "namespace name")
	if err := flags.Parse(args); err != nil {
		return "", err
	}
	if *name == "" {
		return "", fmt.Errorf("namespace operation requires --name")
	}
	return *name, nil
}

func newClient() (client.Client, error) {
	restConfig, err := ctrl.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("loading cluster config: %w", err)
	}

	scheme, err := modulemanager.NewScheme()
	if err != nil {
		return nil, fmt.Errorf("creating Kubernetes scheme: %w", err)
	}

	kubeClient, err := client.New(restConfig, client.Options{Scheme: scheme})
	if err != nil {
		return nil, fmt.Errorf("creating Kubernetes client: %w", err)
	}
	return kubeClient, nil
}

func isGone(err error) bool {
	return apierrors.IsNotFound(err) || meta.IsNoMatchError(err)
}
