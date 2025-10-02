package web

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	pb "github.com/Cuppojoe/agent/pkg/protobuf"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"crypto/rand"
)

type AgentServer struct {
	pb.UnimplementedAgentServer
	HostName string
}

func (s *AgentServer) HelloWorld(context.Context, *pb.HelloWorldRequest) (*pb.HelloWorldResponse, error) {
	return &pb.HelloWorldResponse{
		Message: fmt.Sprintf("Hello, World!\n\n\t-from %s", s.HostName),
	}, nil
}

type SayHelloMessage struct {
	Url string `json:"url"`
}

type SendBytes struct {
	Size      uint64 `json:"size"`
	TargetUrl string `json:"targetUrl"`
}

// ExposeRest starts REST endpoints and metrics
func ExposeRest(hostName string, cv *prometheus.CounterVec) {
	r := http.NewServeMux()
	exposeHelloWorldHandler(r, hostName, cv)
	exposeHealthCheck(r, cv)
	exposeSayHelloGrpc(r)
	exposeSendBytes(r)
	exposeReceiveBytes(r)
	exposeSleep(r, cv)
	go exposeMetrics(cv)
	err := http.ListenAndServe(fmt.Sprintf(":%d", getPortFromEnv("REST_PORT", 8080)), r)
	if err != nil {
		log.Fatal(err.Error())
	}
}

func exposeSendBytes(r *http.ServeMux) {
	r.HandleFunc("/send_bytes", func(rw http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			rw.WriteHeader(500)
			return
		}
		s := SendBytes{}
		err = json.Unmarshal(body, &s)
		if err != nil {
			rw.WriteHeader(400)
			_, _ = rw.Write([]byte("Invalid request body. Details: " + err.Error()))
			return
		}

		b := make([]byte, s.Size)
		if _, err := rand.Read(b); err != nil {
			rw.WriteHeader(500)
			_, _ = rw.Write([]byte("Failed to generate bytes: " + err.Error()))
			return
		}

		if s.TargetUrl == "" {
			_, _ = rw.Write(b)
			return
		}

		response, err := http.Post(s.TargetUrl, "text/plain", bytes.NewBuffer(b))
		if err != nil {
			rw.WriteHeader(400)
			return
		}
		b, err = io.ReadAll(response.Body)
		if err != nil {
			rw.WriteHeader(500)
			return
		}
		_, _ = rw.Write(b)
	})
}

func exposeReceiveBytes(r *http.ServeMux) {
	r.HandleFunc("/receive_bytes", func(rw http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			rw.WriteHeader(500)
			return
		}
		_, _ = rw.Write([]byte(strconv.Itoa(len(body))))
	})
}

func exposeSayHelloGrpc(r *http.ServeMux) {
	r.HandleFunc("/grpc/say-hello", func(rw http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			rw.WriteHeader(405)
			return
		}
		b, err := io.ReadAll(r.Body)
		if err != nil {
			rw.WriteHeader(500)
			return
		}
		m := &SayHelloMessage{}
		err = json.Unmarshal(b, m)
		if err != nil {
			rw.WriteHeader(500)
			_, _ = rw.Write([]byte("Invalid request body"))
			return
		}
		conn, err := grpc.NewClient(m.Url, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			rw.WriteHeader(500)
			_, _ = rw.Write([]byte(fmt.Sprintf("Error while dialing %s: %v", m.Url, err)))
			return
		}
		defer func(conn *grpc.ClientConn) {
			_ = conn.Close()
		}(conn)
		client := pb.NewAgentClient(conn)
		resp, err := client.HelloWorld(context.Background(), &pb.HelloWorldRequest{})
		if err != nil {
			rw.WriteHeader(500)
			_, _ = rw.Write([]byte(fmt.Sprintf("Error while saying hello to \"%s\" Details: %v", m.Url, err)))
			return
		}
		_, err = rw.Write([]byte(fmt.Sprintf("Response from %s: %s", m.Url, resp.Message)))
		if err != nil {
			return
		}
	})
}

func getCounter(req *http.Request, cv *prometheus.CounterVec) prometheus.Counter {
	labelMap := prometheus.Labels{
		"path":     "/",
		"org":      "",
		"location": "",
	}
	err := req.ParseForm()
	if err != nil {
		return nil
	}
	if req.Form.Get("labels") != "" {
		for _, v := range req.Form["labels"] {
			var labels map[string]string
			err := json.Unmarshal([]byte(v), &labels)
			if err != nil {
				fmt.Println(err)
			}
			for k, l := range labels {
				labelMap[k] = l
			}
		}
	}
	m, err := cv.GetMetricWith(labelMap)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	return m
}

func exposeHelloWorldHandler(r *http.ServeMux, hostName string, cv *prometheus.CounterVec) {
	r.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		_, err := fmt.Fprintf(w, "Hello, world!\n\n\t-from %s", hostName)
		if err != nil {
			return
		}
		m := getCounter(req, cv)
		if m != nil {
			m.Inc()
		}
	})
}

func exposeMetrics(cv *prometheus.CounterVec) {
	r := prometheus.NewRegistry()
	r.MustRegister(cv)
	prometheusServer := http.Server{
		Addr: fmt.Sprintf(":%d", getPortFromEnv("METRICS_PORT", 8082)),
		Handler: promhttp.HandlerFor(
			r,
			promhttp.HandlerOpts{},
		),
	}
	err := prometheusServer.ListenAndServe()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func exposeHealthCheck(r *http.ServeMux, cv *prometheus.CounterVec) {
	r.HandleFunc("/health", func(w http.ResponseWriter, req *http.Request) {
		_, err := w.Write([]byte("I am quite healthy. Thanks for asking!"))
		if err != nil {
			return
		}
		m := getCounter(req, cv)
		if m != nil {
			m.Inc()
		}
	})
}

func exposeSleep(r *http.ServeMux, cv *prometheus.CounterVec) {
	r.HandleFunc("/sleep", func(w http.ResponseWriter, req *http.Request) {
		duration, err := time.ParseDuration(req.FormValue("duration"))
		if err != nil {
			w.WriteHeader(400)
			_, _ = w.Write([]byte(fmt.Sprintf("Invalid duration: %s", req.FormValue("duration"))))
			return
		}
		time.Sleep(duration)
		_, _ = w.Write([]byte("OK"))
		m := getCounter(req, cv)
		if m != nil {
			m.Inc()
		}
	})
}

// ExposeGrpc starts gRPC server
func ExposeGrpc(hostName string, cv *prometheus.CounterVec) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", getPortFromEnv("GRPC_PORT", 8081)))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	pb.RegisterAgentServer(grpcServer, newAgentServer(hostName))
	err = grpcServer.Serve(lis)
	if err != nil {
		log.Fatalf("failed to serve grpc. Details: %v", err)
	}
}
func newAgentServer(hostName string) *AgentServer {
	return &AgentServer{
		HostName: hostName,
	}
}

// Helpers moved from main
func getPortFromEnv(envName string, defaultPort int) int {
	portEnv := os.Getenv(envName)
	port, err := strconv.Atoi(portEnv)
	if err != nil {
		port = defaultPort
	}
	return port
}
