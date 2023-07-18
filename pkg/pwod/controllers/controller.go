package controllers

import (
	"context"
	"sync"
	"time"

	"github.com/howardjohn/pilot-load/pkg/simulation/model"
	"istio.io/pkg/log"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"github.com/wzshiming/pwod/pkg/app"
)

type Controller struct {
	args    model.Args
	podsMut sync.Mutex
	pods    map[types.UID]*app.Pod
}
type Args = model.Args

func NewController(args Args) *Controller {
	return &Controller{
		args: args,
		pods: make(map[types.UID]*app.Pod),
	}
}

func (c *Controller) Run(ctx context.Context) error {
	pi := c.args.Client.Kubernetes.CoreV1().Pods("")
	ctx, cancel := context.WithCancel(ctx)

	mctx := model.Context{
		Context: ctx,
		Args:    c.args,
		Client:  c.args.Client,
		Cancel:  cancel,
	}

	log.Info("Starting controller")
	_, ctr := cache.NewInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return pi.List(ctx, options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return pi.Watch(ctx, options)
			},
		},
		&v1.Pod{},
		10*time.Second,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				pod := obj.(*v1.Pod)
				p := app.NewPod(pod)
				err := p.Start(mctx)
				if err != nil {
					log.Error(err)
				} else {
					c.podsMut.Lock()
					defer c.podsMut.Unlock()
					c.pods[pod.UID] = p
				}
			},
			DeleteFunc: func(obj interface{}) {
				pod := obj.(*v1.Pod)
				c.podsMut.Lock()
				defer c.podsMut.Unlock()
				p := c.pods[pod.UID]
				if p == nil {
					return
				}
				err := p.Cleanup(mctx)
				if err != nil {
					log.Error(err)
				}
				delete(c.pods, pod.UID)
			},
		},
	)

	ctr.Run(ctx.Done())

	return nil
}
