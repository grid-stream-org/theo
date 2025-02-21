package handler

import (
	"context"
	"log/slog"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/grid-stream-org/theo/internal/config"
	"github.com/grid-stream-org/theo/internal/event"
	"github.com/pkg/errors"
)

type k8sHandler struct {
	cfg    *config.K8s
	client *kubernetes.Clientset
	log    *slog.Logger
}

const (
	Deployment             = "batcher"
	defaultOnStartReplicas = 1
	defaultOnEndReplicas   = 0
)

func NewK8sHandler(cfg *config.K8s, client *kubernetes.Clientset, log *slog.Logger) event.Handler {
	return &k8sHandler{
		cfg:    cfg,
		client: client,
		log:    log.With("component", "k8s_handler"),
	}
}

func (h *k8sHandler) scaleDeployment(ctx context.Context, deployment string, replicas int32, e event.Event) error {
	dc := h.client.AppsV1().Deployments(h.cfg.Namespace)
	dep, err := dc.Get(ctx, deployment, metav1.GetOptions{})
	if err != nil {
		return errors.WithStack(err)
	}

	dep.Spec.Replicas = &replicas

	if replicas > 0 {
		dep.Spec.Template.Spec.Containers[0].Env = append(
			dep.Spec.Template.Spec.Containers[0].Env,
			corev1.EnvVar{
				Name:  "BUFFER_START_TIME",
				Value: e.StartTime.Format(time.RFC3339),
			},
		)
	}

	_, err = dc.Update(ctx, dep, metav1.UpdateOptions{})
	if err != nil {
		return errors.WithStack(err)
	}

	h.log.Info("scaled deployment", "deployment", deployment, "replicas", replicas)
	return nil
}

func (h *k8sHandler) OnStart(ctx context.Context, e event.Event) error {
	h.log.Info("starting event", e.LogFields()...)
	return h.scaleDeployment(ctx, Deployment, defaultOnStartReplicas, e)
}

func (h *k8sHandler) OnEnd(ctx context.Context, e event.Event) error {
	h.log.Info("ending event", e.LogFields()...)
	return h.scaleDeployment(ctx, Deployment, defaultOnEndReplicas, e)
}
