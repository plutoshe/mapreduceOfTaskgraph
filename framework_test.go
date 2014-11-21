package meritop

// framework_test tests basic workflows of framework impl.
// It uses a scenario with two nodes: 0 as parent, 1 as child.
// The basic idea is that when parent tries to talk to child and vice versa,
// there will be some data transferring and captured by application task.
// Here we have implemented a helper user task to capture those data, test if
// it's passed from framework correctly and unmodified.

import (
	"bytes"
	"fmt"
	"net"
	"testing"
)

func TestFrameworkFlagMetaReady(t *testing.T) {
	m := mustNewMember(t, "framework_test")
	m.Launch()
	defer m.Terminate(t)
	url := fmt.Sprintf("http://%s", m.ClientListeners[0].Addr().String())

	pDataChan := make(chan *tDataBundle, 1)
	cDataChan := make(chan *tDataBundle, 1)
	// simulate two tasks on two nodes -- 0 and 1
	// 0 is parent, 1 is child
	f0 := &framework{
		name:     "framework_test_flagmetaready",
		etcdURLs: []string{url},
		taskID:   0,
		task: &testableTask{
			dataChan: cDataChan,
		},
		topology: NewTreeTopology(2, 1),
		ln:       createListener(t),
	}
	f1 := &framework{
		name:     "framework_test_flagmetaready",
		etcdURLs: []string{url},
		taskID:   1,
		task: &testableTask{
			dataChan: pDataChan,
		},
		topology: NewTreeTopology(2, 1),
		ln:       createListener(t),
	}
	f0.start()
	defer f0.stop()
	f1.start()
	defer f1.stop()

	tests := []struct {
		cMeta string
		pMeta string
	}{
		{"parent", "child"},
		{"ParamReady", "GradientReady"},
	}

	for i, tt := range tests {
		// 0: F#FlagChildMetaReady -> 1: T#ParentMetaReady
		f0.FlagChildMetaReady(tt.cMeta)
		// from child(1)'s view
		data := <-pDataChan
		if data.id != 0 {
			t.Errorf("#%d: parentID want = 0, get = %d", data.id)
		}
		if data.meta != tt.cMeta {
			t.Errorf("#%d: meta want = %s, get = %s", i, tt.cMeta, data.meta)
		}

		// 1: F#FlagParentMetaReady -> 0: T#ChildMetaReady
		f1.FlagParentMetaReady(tt.pMeta)
		// from parent(0)'s view
		data = <-cDataChan
		if data.id != 1 {
			t.Errorf("#%d: parentID want = 1, get = %d", data.id)
		}
		if data.meta != tt.pMeta {
			t.Errorf("#%d: meta want = %s, get = %s", i, tt.pMeta, data.meta)
		}
	}
}

func TestFrameworkDataRequest(t *testing.T) {
	tests := []struct {
		req  string
		resp []byte
	}{
		{"request", []byte("response")},
		{"parameters", []byte{1, 2, 3}},
		{"gradient", []byte{4, 5, 6}},
	}

	dataMap := make(map[string][]byte)
	for _, tt := range tests {
		dataMap[tt.req] = tt.resp
	}

	m := mustNewMember(t, "framework_test")
	m.Launch()
	defer m.Terminate(t)
	url := fmt.Sprintf("http://%s", m.ClientListeners[0].Addr().String())
	l0 := createListener(t)
	l1 := createListener(t)
	addressMap := map[uint64]string{
		0: l0.Addr().String(),
		1: l1.Addr().String(),
	}

	pDataChan := make(chan *tDataBundle, 1)
	cDataChan := make(chan *tDataBundle, 1)
	// simulate two tasks on two nodes -- 0 and 1
	// 0 is parent, 1 is child
	f0 := &framework{
		name:     "framework_test_datarequest",
		etcdURLs: []string{url},
		taskID:   0,
		task: &testableTask{
			dataMap:  dataMap,
			dataChan: cDataChan,
		},
		topology:   NewTreeTopology(2, 1),
		ln:         l0,
		addressMap: addressMap,
	}
	f1 := &framework{
		name:     "framework_test_datarequest",
		etcdURLs: []string{url},
		taskID:   1,
		task: &testableTask{
			dataMap:  dataMap,
			dataChan: pDataChan,
		},
		topology:   NewTreeTopology(2, 1),
		ln:         l1,
		addressMap: addressMap,
	}
	f0.start()
	defer f0.stop()
	f1.start()
	defer f1.stop()

	for i, tt := range tests {
		// 0: F#DataRequest -> 1: T#ServeAsChild -> 0: T#ChildDataReady
		f0.DataRequest(1, tt.req)
		// from child(1)'s view at 1: T#ServeAsChild
		data := <-pDataChan
		if data.id != 0 {
			t.Errorf("#%d: fromID want = 0, get = %d", i, data.id)
		}
		if data.req != tt.req {
			t.Errorf("#%d: req want = %s, get = %s", i, tt.req, data.req)
		}
		// from parent(0)'s view at 0: T#ChildDataReady
		data = <-cDataChan
		if data.id != 1 {
			t.Errorf("#%d: fromID want = 1, get = %d", i, data.id)
		}
		if data.req != tt.req {
			t.Errorf("#%d: req want = %s, get = %s", i, tt.req, data.req)
		}
		if bytes.Compare(data.resp, tt.resp) != 0 {
			t.Errorf("#%d: resp want = %v, get = %v", i, tt.resp, data.resp)
		}

		// 1: F#DataRequest -> 0: T#ServeAsParent -> 1: T#ParentDataReady
		f1.DataRequest(0, tt.req)
		// from parent(0)'s view at 0: T#ServeAsParent
		data = <-cDataChan
		if data.id != 1 {
			t.Errorf("#%d: fromID want = 1, get = %d", i, data.id)
		}
		if data.req != tt.req {
			t.Errorf("#%d: req want = %s, get = %s", i, tt.req, data.req)
		}
		// from child(1)'s view at 1: T#ParentDataReady
		data = <-pDataChan
		if data.id != 0 {
			t.Errorf("#%d: fromID want = 1, get = %d", i, data.id)
		}
		if data.req != tt.req {
			t.Errorf("#%d: req want = %s, get = %s", i, tt.req, data.req)
		}
		if bytes.Compare(data.resp, tt.resp) != 0 {
			t.Errorf("#%d: resp want = %v, get = %v", i, tt.resp, data.resp)
		}
	}
}

type tDataBundle struct {
	id   uint64
	meta string
	req  string
	resp []byte
}

type testableTask struct {
	id        uint64
	framework Framework
	// dataMap will be used to serve data according to request
	dataMap map[string][]byte

	// This channel is used to convey data passed from framework back to the main
	// thread, for checking. Thus it's initialized and passed in from outside.
	//
	// The basic idea is that there are only two nodes -- one parent and one child.
	// When this channel is for parent, it passes information from child.
	dataChan chan *tDataBundle
}

func (t *testableTask) Init(taskID uint64, framework Framework, config Config) {
	t.id = taskID
	t.framework = framework
}
func (t *testableTask) Exit()                 {}
func (t *testableTask) SetEpoch(epoch uint64) {}

func (t *testableTask) ParentMetaReady(fromID uint64, meta string) {
	if t.dataChan != nil {
		t.dataChan <- &tDataBundle{fromID, meta, "", nil}
	}
}

func (t *testableTask) ChildMetaReady(fromID uint64, meta string) {
	t.ParentMetaReady(fromID, meta)
}

func (t *testableTask) ServeAsParent(fromID uint64, req string) []byte {
	if t.dataChan != nil {
		t.dataChan <- &tDataBundle{fromID, "", req, nil}
	}
	return t.dataMap[req]
}
func (t *testableTask) ServeAsChild(fromID uint64, req string) []byte {
	return t.ServeAsParent(fromID, req)
}
func (t *testableTask) ParentDataReady(fromID uint64, req string, resp []byte) {
	if t.dataChan != nil {
		t.dataChan <- &tDataBundle{fromID, "", req, resp}
	}
}

func (t *testableTask) ChildDataReady(fromID uint64, req string, resp []byte) {
	t.ParentDataReady(fromID, req, resp)
}

func createListener(t *testing.T) net.Listener {
	l, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen(\"tcp4\", \"\") failed: %v", err)
	}
	return l
}
