package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// ValidationWebhook handles the validation logic for pod creation
type ValidationWebhook struct {
	Client  client.Client
	Decoder *admission.Decoder
}

// Pod represents a Kubernetes Pod to be validated
type Pod struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              PodSpec `json:"spec"`
}

type PodSpec struct {
	Containers []Container `json:"containers"`
}

type Container struct {
	Image string `json:"image"`
}

func (v *ValidationWebhook) Handle(ctx context.Context, req admission.Request) admission.Response {
	// Parse the incoming object into a Pod
	pod := &Pod{}
	if err := v.Decoder.Decode(req, pod); err != nil {
		return admission.Errored(http.StatusBadRequest, fmt.Errorf("could not decode object: %w", err))
	}

	// Validate the containers' image field
	for _, container := range pod.Spec.Containers {
		if container.Image == "" {
			return admission.Denied("image name is not defined for container")
		}
	}

	// If everything is good, allow the resource creation
	return admission.Allowed("image name is defined")
}

func main() {
	// Create a new webhook server
	mgr, err := manager.New(cfg, manager.Options{})
	if err != nil {
		log.Fatalf("Unable to set up overall controller manager: %v", err)
	}

	// Create the webhook handler
	webhookHandler := &ValidationWebhook{
		Client: mgr.GetClient(),
	}

	// Create a new admission controller
	admissionHandler := admission.HandlerFunc(webhookHandler.Handle)
	mgr.AddWebhook(&webhook.Webhook{
		// Pod is the resource this validation will target
		Object:  &Pod{},
		Path:    "/validate-image",
		Handler: admissionHandler,
	})

	// Start the webhook server to listen on port 8080 for requests
	log.Fatal(mgr.Start(signals.SetupSignalHandler()))
}
