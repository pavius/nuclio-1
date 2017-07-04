package server

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"time"
	"github.com/nuclio/nuclio/pkg/kubecr"
	"github.com/nuclio/nuclio/pkg/util/common"
	"github.com/nuclio/nuclio/cmd/controller/app/deploy"
)

const MaxChannelSize  = 100

type MsgType int8

const (
	AddMsg    MsgType = 0
	UpdateMsg MsgType = 1
	DelMsg    MsgType = 2
)

type chanmsg struct {
	Mtype MsgType
	Namespace, Name string
	Gen string
}


func NewServer(cs *kubernetes.Clientset, crcl  *rest.RESTClient) (*Server, error) {
	newServer := Server{cs:cs, crcl:crcl}
	newServer.fchan = make(chan chanmsg, 100)

	return &newServer, nil
}

type Server struct {
	cs    *kubernetes.Clientset
	crcl  *rest.RESTClient
	fchan chan chanmsg
	stateCache  *deploy.FuncCache
}

func (srv *Server) Start() error {

	defer close(srv.fchan)

	srv.watchFunctions("")

	stateCache := deploy.NewFuncCache()
	err := deploy.InitFlist(srv.cs,stateCache)
	if err != nil {
		return err
	}

	go func() {
		for {
			msg, _ := <-srv.fchan
			if msg.Mtype == DelMsg {
				err = deploy.DeleteFunc(srv.cs, msg.Namespace, msg.Name)
				if err != nil {
					common.LogDebug("Cant Delete Function resources: %s %s - %s",msg.Namespace, msg.Name,err)
				}
				stateCache.Del(msg.Namespace, msg.Name)
			} else {
				// TODO: handle retry on update conflict
				err = HandleFuncChange(srv.crcl, srv.cs, msg.Namespace, msg.Name, msg.Gen)
				if err != nil {
					panic(err)
				}
			}
		}
	}()

	// TODO: few sec after start clean orphan deployments/dervices/HPA
	//   (e.g. in case function was deleted and controller failed in the middle)

}


func (srv *Server) watchFunctions(namespace string) {

	fif := kubecr.Functions(srv.crcl)

	_, controller := cache.NewInformer(
		fif.NewListWatch(namespace),
		&kubecr.Function{},
		time.Minute * 10,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				f := obj.(*kubecr.Function)
				msg := chanmsg{Mtype:AddMsg, Namespace:f.Namespace, Name:f.Name, Gen:f.ResourceVersion}
				srv.fchan <- msg

			},
			DeleteFunc: func(obj interface{}) {
				f := obj.(*kubecr.Function)
				msg := chanmsg{Mtype:DelMsg, Namespace:f.Namespace, Name:f.Name, Gen:f.ResourceVersion}
				srv.fchan <- msg
			},
			UpdateFunc:func(oldObj, newObj interface{}) {
				f := newObj.(*kubecr.Function)
				msg := chanmsg{Mtype:UpdateMsg, Namespace:f.Namespace, Name:f.Name, Gen:f.ResourceVersion}
				srv.fchan <- msg
			},
		},
	)

	stop := make(chan struct{})
	go controller.Run(stop)

}

