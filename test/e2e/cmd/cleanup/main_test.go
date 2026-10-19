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
	"testing"

	. "github.com/onsi/gomega"
	servicesv1alpha1 "github.com/opendatahub-io/opendatahub-db-operator/api/services/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCleanupDatabaseServiceLeavesUnownedSingleton(t *testing.T) {
	g := NewWithT(t)
	scheme := runtime.NewScheme()
	g.Expect(servicesv1alpha1.AddToScheme(scheme)).To(Succeed())

	singleton := &servicesv1alpha1.DatabaseService{
		ObjectMeta: metav1.ObjectMeta{Name: servicesv1alpha1.DatabaseServiceInstanceName},
	}
	kubeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(singleton).Build()

	g.Expect(cleanupDatabaseServiceWithClient(context.Background(), kubeClient)).To(Succeed())

	actual := &servicesv1alpha1.DatabaseService{}
	g.Expect(kubeClient.Get(context.Background(), client.ObjectKeyFromObject(singleton), actual)).To(Succeed())
}

func TestCleanupNamespaceLeavesUnownedNamespace(t *testing.T) {
	g := NewWithT(t)
	scheme := runtime.NewScheme()
	g.Expect(corev1.AddToScheme(scheme)).To(Succeed())

	namespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "keep-me"}}
	kubeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(namespace).Build()

	g.Expect(cleanupNamespaceWithClient(context.Background(), kubeClient, namespace.Name)).To(HaveOccurred())

	actual := &corev1.Namespace{}
	g.Expect(kubeClient.Get(context.Background(), client.ObjectKeyFromObject(namespace), actual)).To(Succeed())
}
