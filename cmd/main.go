package main

import (
	"github.com/Cuppojoe/agent/pkg/commander"
	"github.com/Cuppojoe/agent/pkg/web"

	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/prometheus/client_golang/prometheus"
)

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
		err := flagSet.Parse(os.Args[2:])
		if err != nil {
			return
		}
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

func runServer() {
	hostName := os.Getenv("HOSTNAME")
	if hostName == "" {
		hostName = "agent"
	}
	cv := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "request_count",
	}, []string{"path", "org", "location"})

	go web.ExposeGrpc(hostName, cv)
	web.ExposeRest(hostName, cv)
}
