package handler

import (
	"context"
	"log/slog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/grid-stream-org/theo/internal/config"
	"github.com/grid-stream-org/theo/internal/event"
	"github.com/pkg/errors"
)

type K8sHandler struct {
	cfg    *config.K8s
	client *kubernetes.Clientset
	log    *slog.Logger
}

const (
	Deployment      = "batcher"
	OnStartReplicas = 1
	OnEndReplicas   = 0
)

func NewK8sHandler(cfg *config.K8s, client *kubernetes.Clientset, log *slog.Logger) event.Handler {
	return &K8sHandler{
		cfg:    cfg,
		client: client,
		log:    log.With("component", "k8s_handler"),
	}
}

func (h *K8sHandler) scaleDeployment(ctx context.Context, deployment string, replicas int32) error {
	dc := h.client.AppsV1().Deployments(h.cfg.Namespace)
	dep, err := dc.Get(ctx, deployment, metav1.GetOptions{})
	if err != nil {
		return errors.WithStack(err)
	}

	dep.Spec.Replicas = &replicas

	_, err = dc.Update(ctx, dep, metav1.UpdateOptions{})
	if err != nil {
		return errors.WithStack(err)
	}

	h.log.Info("scaled deployment", "deployment", deployment, "replicas", replicas)
	return nil
}

func (h *K8sHandler) OnStart(ctx context.Context, e event.Event) error {
	h.log.Info("starting DR event", e.LogFields()...)
	return h.scaleDeployment(ctx, Deployment, OnStartReplicas)
}

// OnEnd scales down the "batcher" deployment.
func (h *K8sHandler) OnEnd(ctx context.Context, e event.Event) error {
	h.log.Info("ending DR event", e.LogFields()...)
	return h.scaleDeployment(ctx, Deployment, OnEndReplicas)
}
