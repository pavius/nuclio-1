package app

import (
	"k8s.io/client-go/rest"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"github.com/nuclio/nuclio/pkg/logger"
	"github.com/nuclio/nuclio/pkg/kubecr"
	"github.com/nuclio/nuclio/cmd/controller/app/deploy"
	"time"
	"github.com/nuclio/nuclio/pkg/util/common"
	"github.com/nuclio/nuclio/cmd/controller/app/handler"
)


const MaxChannelSize  = 500

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

type Controller struct {
	logger      logger.Logger
	confPath    string
	fchan       chan chanmsg
}

func NewController(configurationPath string) (*Controller, error) {

	newController := Controller{ confPath:configurationPath }
	newController.fchan = make(chan chanmsg, MaxChannelSize)

	return &newController, nil
}

func (c *Controller) Start() error {

	defer close(c.fchan)

	config, err := kubecr.GetClientConfig(c.confPath)
	if err != nil {
		return err
	}
	cs, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	crcl, err := kubecr.CreateFuncCR(config, cs)
	if err != nil {
		return err
	}

	c.watchFunctions(crcl, "")

	stateCache := deploy.NewFuncCache(cs)
	err = stateCache.Init("")
	if err != nil {
		return err
	}

	hand, err := handler.NewHandler(cs , crcl , stateCache)

	go func() {
		for {
			msg, _ := <-c.fchan
			if msg.Mtype == DelMsg {
				err = deploy.DeleteFunc(cs, msg.Namespace, msg.Name)
				if err != nil {
					common.LogDebug("Cant Delete Function resources: %s %s - %s",msg.Namespace, msg.Name,err)
				} else {
					common.LogDebug("Deleted Function: %s %s ",msg.Namespace, msg.Name)
				}
				stateCache.Del(msg.Namespace, msg.Name)
			} else {
				// TODO: handle retry on update conflict
				err = hand.HandleFuncChange(msg.Namespace, msg.Name, msg.Gen)
				if err != nil {
					panic(err)
				}
			}
		}
	}()



	// TODO: shutdown
	select {}

	return nil
}



func (c *Controller) watchFunctions(crcl *rest.RESTClient, namespace string) {

	fif := kubecr.Functions(crcl)

	_, ctrl := cache.NewInformer(
		fif.NewListWatch(namespace),
		&kubecr.Function{},
		time.Minute * 10,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				f := obj.(*kubecr.Function)
				msg := chanmsg{Mtype:AddMsg, Namespace:f.Namespace, Name:f.Name, Gen:f.ResourceVersion}
				c.fchan <- msg

			},
			DeleteFunc: func(obj interface{}) {
				f := obj.(*kubecr.Function)
				msg := chanmsg{Mtype:DelMsg, Namespace:f.Namespace, Name:f.Name, Gen:f.ResourceVersion}
				c.fchan <- msg
			},
			UpdateFunc:func(oldObj, newObj interface{}) {
				f := newObj.(*kubecr.Function)
				msg := chanmsg{Mtype:UpdateMsg, Namespace:f.Namespace, Name:f.Name, Gen:f.ResourceVersion}
				c.fchan <- msg
			},
		},
	)

	stop := make(chan struct{})
	go ctrl.Run(stop)

}

