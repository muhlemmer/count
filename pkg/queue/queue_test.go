package queue

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/muhlemmer/count/internal/db"
	"github.com/muhlemmer/count/internal/db/migrations"
	"github.com/muhlemmer/count/internal/service"
	countv1 "github.com/muhlemmer/count/pkg/api/count/v1"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	testCTX    context.Context
	errCTX     context.Context
	testClient countv1.CountServiceClient
	testServer *grpc.Server
)

const dsn = "postgresql://muhlemmer@db:5432/muhlemmer?sslmode=disable"

func testMain(m *testing.M) int {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, NoColor: true}).With().Timestamp().Logger()
	testCTX = logger.WithContext(ctx)
	errCTX, cancel = context.WithCancel(testCTX)
	cancel()

	migrDSN := strings.Replace(dsn, "postgresql", "cockroachdb", 1)

	migrations.Down(migrDSN)
	migrations.Up(migrDSN)

	db, err := db.New(testCTX, dsn)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	testServer = grpc.NewServer(grpc.Creds(insecure.NewCredentials()))
	service.NewCountService(testServer, db)

	lis, err := net.Listen("tcp", "127.0.0.1:9999")
	if err != nil {
		panic(err)
	}

	go func() {
		logger.Err(testServer.Serve(lis)).Msg("grpc server terminated")
	}()
	defer testServer.GracefulStop()

	cc, err := grpc.DialContext(testCTX, "127.0.0.1:9999",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		panic(err)
	}

	testClient = countv1.NewCountServiceClient(cc)

	return m.Run()
}

func TestMain(m *testing.M) {
	os.Exit(testMain(m))
}

func TestCountAddQueue_reconnectCountAddClientStream(t *testing.T) {
	type fields struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name:    "context error",
			fields:  fields{errCTX},
			wantErr: true,
		},
		{
			name:   "success",
			fields: fields{testCTX},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &CountAddQueue{
				ctx:    tt.fields.ctx,
				client: testClient,
			}
			if err := c.reconnectCountAddClientStream(); (err != nil) != tt.wantErr {
				t.Errorf("CountAddQueue.reconnectCountAddClientStream() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

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

func BenchmarkCountAddQueue_processQueue(b *testing.B) {
	stream, err := testClient.Add(testCTX)
	if err != nil {
		b.Fatal(err)
	}

	c := &CountAddQueue{
		ctx:    testCTX,
		client: testClient,
		queue:  make(chan *request),
		stream: stream,
	}

	c.wg.Add(1)
	go func() {
		c.processQueue()
		c.wg.Done()
	}()

	b.Run("bench", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			c.queue <- &request{testCTX, &countv1.AddRequest{
				Method:           countv1.Method_DELETE,
				Path:             "/foo/bar",
				RequestTimestamp: timestamppb.Now(),
			}}
		}
	})

	close(c.queue)
	c.wg.Wait()
}

var testStream = []*countv1.AddRequest{
	{
		Method:           countv1.Method_GET,
		Path:             "/foo/bar",
		RequestTimestamp: timestamppb.New(time.Unix(123, 0)),
	},
	{
		Method:           countv1.Method_POST,
		Path:             "/items/new",
		RequestTimestamp: timestamppb.New(time.Unix(456, 0)),
	},
	{
		Method:           countv1.Method_PUT,
		Path:             "/items/update",
		RequestTimestamp: timestamppb.New(time.Unix(789, 0)),
	},
}

func TestNewCountAddClient(t *testing.T) {
	type args struct {
		ctx    context.Context
		client countv1.CountServiceClient
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "context error",
			args:    args{errCTX, testClient},
			wantErr: true,
		},
		{
			name: "success",
			args: args{testCTX, testClient},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := NewCountAddClient(tt.args.ctx, tt.args.client)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewCountAddClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				for _, req := range testStream {
					q.Queue(testCTX, req)
				}
				q.Close()
			}
		})
	}
}

func TestCountAddQueue_QueueOrDrop(t *testing.T) {
	c := &CountAddQueue{
		queue: make(chan *request, 2),
	}

	for _, req := range testStream {
		c.QueueOrDrop(testCTX, req)
	}
}

func TestCountAddQueue_Middleware(t *testing.T) {
	c := &CountAddQueue{
		queue: make(chan *request),
	}

	r := httptest.NewRequest(http.MethodGet, "/foo/bar", nil)
	w := httptest.NewRecorder()

	const want = "Hello, world!"

	c.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello, world!")
	}))(w, r)

	res := w.Result()
	if res.StatusCode != http.StatusOK {
		t.Errorf("status code = %d, want %d", res.StatusCode, http.StatusOK)
	}
	b, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}

	if got := string(b); got != want {
		t.Errorf("CountAddQueue.Middleware = %s, want %s", got, want)
	}
}

func TestCountAddQueue_UnaryInterceptor(t *testing.T) {
	c := &CountAddQueue{
		queue: make(chan *request),
	}

	var handlerCalled bool
	c.UnaryInterceptor()(testCTX, nil, &grpc.UnaryServerInfo{FullMethod: "foo.bar"}, func(ctx context.Context, req interface{}) (interface{}, error) {
		handlerCalled = true
		return nil, nil
	})

	if !handlerCalled {
		t.Error("CountAddQueue.UnaryInterceptor handler not called")
	}
}
