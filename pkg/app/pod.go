package app

import (
	"github.com/howardjohn/pilot-load/pkg/simulation/model"
	"github.com/howardjohn/pilot-load/pkg/simulation/xds"
	v1 "k8s.io/api/core/v1"
)

func podToPodSpec(pod *v1.Pod) *podSpec {
	at := model.SidecarType
	if len(pod.Spec.Containers) == 1 {
		at = model.GatewayType
	} else if len(pod.Spec.InitContainers) == 0 {
		at = model.AmbientType
	}
	return &podSpec{
		ServiceAccount: pod.Spec.ServiceAccountName,
		Node:           pod.Spec.NodeName,
		App:            pod.Name,
		Namespace:      pod.Namespace,
		Labels:         pod.Labels,
		UID:            string(pod.UID),
		IP:             pod.Status.PodIP,
		AppType:        at,
	}
}

type podSpec struct {
	ServiceAccount string
	Node           string
	App            string
	Namespace      string
	Labels         map[string]string
	UID            string
	IP             string
	AppType        model.AppType
}

type Pod struct {
	spec *podSpec
	xds  *xds.Simulation
}

func NewPod(pod *v1.Pod) *Pod {
	return &Pod{
		spec: podToPodSpec(pod),
	}
}

func (p *Pod) Start(ctx model.Context) (err error) {
	if !p.spec.AppType.HasProxy() {
		return nil
	}

	p.xds = &xds.Simulation{
		Labels:    p.spec.Labels,
		Namespace: p.spec.Namespace,
		Name:      p.spec.App,
		IP:        p.spec.IP,
		AppType:   p.spec.AppType,
		// TODO: multicluster
		Cluster:  "Kubernetes",
		GrpcOpts: ctx.Args.Auth.GrpcOptions(p.spec.ServiceAccount, p.spec.Namespace),
		Delta:    ctx.Args.DeltaXDS,
	}
	return p.xds.Run(ctx)
}

func (p *Pod) Cleanup(ctx model.Context) error {
	if p.xds != nil {
		return p.xds.Cleanup(ctx)
	}
	return nil
}
