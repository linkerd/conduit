package destination

import (
	"context"

	common "github.com/runconduit/conduit/controller/gen/common"
)

// implements the updateListener interface
type collectUpdateListener struct {
	added             []common.TcpAddress
	removed           []common.TcpAddress
	noEndpointsCalled bool
	noEndpointsExists bool
	context           context.Context
	stopCh            chan struct{}
}

func (c *collectUpdateListener) Update(add []common.TcpAddress, remove []common.TcpAddress) {
	c.added = append(c.added, add...)
	c.removed = append(c.removed, remove...)
}

func (c *collectUpdateListener) ClientClose() <-chan struct{} {
	return c.context.Done()
}

func (c *collectUpdateListener) ServerClose() <-chan struct{} {
	return c.stopCh
}

func (c *collectUpdateListener) Stop() {
	close(c.stopCh)
}

func (c *collectUpdateListener) NoEndpoints(exists bool) {
	c.noEndpointsCalled = true
	c.noEndpointsExists = exists
}

func (c *collectUpdateListener) SetServiceId(id *serviceId) {}

func newCollectUpdateListener() (*collectUpdateListener, context.CancelFunc) {
	ctx, cancelFn := context.WithCancel(context.Background())
	return &collectUpdateListener{context: ctx}, cancelFn
}
