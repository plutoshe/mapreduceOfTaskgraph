package controller

import (
	"log"
	"os"

	"github.com/coreos/go-etcd/etcd"
	"github.com/go-distributed/meritop/pkg/etcdutil"
)

// This is the controller of a job.
// A job needs controller to setup etcd data layout, request
// cluster containers, etc. to setup framework to run.
type Controller struct {
	name           string
	etcdclient     *etcd.Client
	numOfTasks     uint64
	failDetectStop chan bool
	logger         *log.Logger
}

func New(name string, etcd *etcd.Client, numOfTasks uint64) *Controller {
	return &Controller{
		name:       name,
		etcdclient: etcd,
		numOfTasks: numOfTasks,
		logger:     log.New(os.Stdout, "", log.Lshortfile|log.Ltime|log.Ldate),
	}
}

func (c *Controller) Start() error {
	if err := c.InitEtcdLayout(); err != nil {
		return err
	}
	return c.startFailureDetection()
}

func (c *Controller) Stop() error {
	c.DestroyEtcdLayout()
	c.stopFailureDetection()
	return nil
}

func (c *Controller) InitEtcdLayout() (err error) {
	// Initilize the job epoch to 0
	if _, err = c.etcdclient.Create(etcdutil.EpochPath(c.name), "0", 0); err != nil {
		return err
	}

	// initiate etcd data layout
	// currently it creates as many unassigned tasks as task masters.
	for i := uint64(0); i < c.numOfTasks; i++ {
		key := etcdutil.MakeTaskMasterPath(c.name, i)
		if _, err := c.etcdclient.Create(key, "empty", 0); err != nil {
			return err
		}
	}
	return
}

func (c *Controller) DestroyEtcdLayout() error {
	_, err := c.etcdclient.Delete("/", true)
	return err
}

func (c *Controller) startFailureDetection() error {
	c.failDetectStop = make(chan bool, 1)
	return etcdutil.DetectFailure(c.etcdclient, c.name, c.failDetectStop)
}

func (c *Controller) stopFailureDetection() error {
	c.failDetectStop <- true
	return nil
}
