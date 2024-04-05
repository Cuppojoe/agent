package main

import (
	"agent/commander"
	pb "agent/protobuf"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
)

type SayHelloMessage struct {
	Url string `json:"url"`
}

func main() {
	if len(os.Args) < 2 {
		displayUsageAndExit()
	}
	mode := os.Args[1]
	var args []string
	switch mode {
	case "client":
		flagSet := flag.NewFlagSet("\"agent client\"", flag.ExitOnError)
		userCount := flagSet.Int("c", 10, "The number of concurrent users.")
		targetUrl := flagSet.String("u", "", "The target URL to load test. (REQUIRED)")
		attackUnitRate := flagSet.String("r", "", "The approximate attack unit rate. This only works well if the response time of the target is far lower than the value of this parameter.")
		timeSpan := flagSet.String("t", "", "The time span for which the load test should be run. If this option is omitted, the agent will run indefinitely.")
		flagSet.Parse(os.Args[2:])
		args = make([]string, 0)
		args = append(args, "-c")
		args = append(args, strconv.Itoa(*userCount))
		if *attackUnitRate != "" {
			args = append(args, "-t")
			args = append(args, *attackUnitRate)
		}
		if *targetUrl == "" {
			fmt.Println("Missing required flag: url")
			os.Exit(1)
		}
		args = append(args, *targetUrl)
		runClient(*userCount, *targetUrl, *attackUnitRate, *timeSpan)
		break
	case "server":
		fmt.Println("Starting the server")
		go runServer()
		break
	default:
		displayUsageAndExit()
	}
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM)
	<-sigChan
	fmt.Println("Shutting down")
	os.Exit(0)
}

func displayUsageAndExit() {
	fmt.Println("Please enter a valid sub-command from the list below:\n - client:       In client mode, the app will barrage a targeted URL with requests.\n - server:       In server mode, the app will serve HTTP traffic on port 8080.")
	os.Exit(1)
}

func runClient(soldierCount int, targetUrl string, attackUnitRate string, timeSpan string) {
	c := commander.NewCommander(soldierCount)
	c.Assault(targetUrl, attackUnitRate, timeSpan)
}

type Labels map[string]string

func runServer() {
	hostName := os.Getenv("HOSTNAME")
	if hostName == "" {
		hostName = "agent"
	}
	cv := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "request_count",
	}, []string{"path", "org", "location"})

	go exposeGrpc(hostName, cv)
	exposeRest(hostName, cv)
}

func exposeRest(hostName string, cv *prometheus.CounterVec) {
	r := mux.NewRouter()
	exposeHelloWorldHandler(r, hostName, cv)
	exposeHealthCheck(r, cv)
	exposeSayHelloGrpc(r)
	go exposeMetrics(cv)
	err := http.ListenAndServe(fmt.Sprintf(":%d", getPortFromEnv("REST_PORT", 8080)), r)
	if err != nil {
		log.Fatal(err.Error())
	}
}

func exposeSayHelloGrpc(r *mux.Router) {
	r.HandleFunc("/grpc/say-hello", func(rw http.ResponseWriter, r *http.Request) {
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
		conn, err := grpc.Dial(m.Url, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			rw.WriteHeader(500)
			_, _ = rw.Write([]byte(fmt.Sprintf("Error while dialing %s: %v", m.Url, err)))
			return
		}
		defer conn.Close()
		client := pb.NewAgentClient(conn)
		resp, err := client.HelloWorld(context.Background(), &pb.HelloWorldRequest{})
		if err != nil {
			rw.WriteHeader(500)
			_, _ = rw.Write([]byte(fmt.Sprintf("Error while saying hello to \"%s\" Details: %v", m.Url, err)))
			return
		}
		rw.Write([]byte(fmt.Sprintf("Response from %s: %s", m.Url, resp.Message)))

	}).Methods("POST")
}

func getCounter(req *http.Request, cv *prometheus.CounterVec) prometheus.Counter {
	labelMap := prometheus.Labels{
		"path":     "/",
		"org":      "",
		"location": "",
	}
	req.ParseForm()
	if req.Form.Get("labels") != "" {
		for _, v := range req.Form["labels"] {
			var labels Labels
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

func exposeHelloWorldHandler(r *mux.Router, hostName string, cv *prometheus.CounterVec) {
	r.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(w, "Hello, world!\n\n\t-from %s", hostName)
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

func exposeHealthCheck(r *mux.Router, cv *prometheus.CounterVec) {
	r.HandleFunc("/health", func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("I am quite healthy. Thanks for asking!"))
		m := getCounter(req, cv)
		if m != nil {
			m.Inc()
		}
	})
}

//Grpc stuff

func newAgentServer(hostName string) *AgentServer {
	return &AgentServer{
		HostName: hostName,
	}
}

func exposeGrpc(hostName string, cv *prometheus.CounterVec) {
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

func getPortFromEnv(envName string, defaultPort int) int {
	portEnv := os.Getenv(envName)
	port, err := strconv.Atoi(portEnv)
	if err != nil {
		port = defaultPort
	}
	return port
}
