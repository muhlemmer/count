//go:build !race

package queue

import (
	"testing"
	"time"

	countv1 "github.com/muhlemmer/count/pkg/api/count/v1"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestCountAddQueue_processQueue(t *testing.T) {
	stream, err := testClient.Add(testCTX)
	if err != nil {
		t.Fatal(err)
	}

	c := &CountAddQueue{
		ctx:    testCTX,
		client: testClient,
		queue:  make(chan *request, 500),
		stream: stream,
	}

	c.wg.Add(2)
	go func() {
		c.processQueue()
		c.wg.Done()
	}()
	go func() {
		time.Sleep(time.Second)
		resp, err := c.stream.CloseAndRecv()
		zerolog.Ctx(testCTX).Err(err).Stringer("response", resp).Msg("stream closed")
		c.wg.Done()
	}()

	for i := 0; i < 15; i++ {
		time.Sleep(time.Second / 10)
		c.queue <- &request{testCTX, &countv1.AddRequest{
			Method:           countv1.Method_DELETE,
			Path:             "/foo/bar",
			RequestTimestamp: timestamppb.Now(),
		}}
	}
	close(c.queue)

	c.wg.Wait()
}
