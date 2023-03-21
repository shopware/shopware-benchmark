package cmd

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var re = regexp.MustCompile(`(?m)^ExecStart=.*`)

var locustCmd = &cobra.Command{
	Use:   "locust",
	Short: "Run Locust",
	RunE: func(cmd *cobra.Command, args []string) error {
		workerAmount, _ := cmd.PersistentFlags().GetInt("worker")
		users, _ := cmd.PersistentFlags().GetString("users")
		spawnRate, _ := cmd.PersistentFlags().GetString("spawn-rate")
		host, _ := cmd.PersistentFlags().GetString("host")
		headless, _ := cmd.PersistentFlags().GetBool("headless")
		runTime, _ := cmd.PersistentFlags().GetString("run-time")
		htmlName, _ := cmd.PersistentFlags().GetString("html-name")
		locustFile, _ := cmd.PersistentFlags().GetString("file")
		runEnv, _ := cmd.PersistentFlags().GetString("run-env")
		useTimeScale, _ := cmd.PersistentFlags().GetBool("time-scale")

		platformHash, err := getPlatformHash(cmd.Context())

		if err != nil {
			return err
		}

		timeScaleArgs := []string{
			"--timescale",
			"--pghost", "grafana-1.sw-bench.de",
			"--pgport", "5432",
			"--pgpassword", "shopware",
			"--pguser", "postgres",
			"--pgdatabase", "postgres",
			"--grafana-url", "https://grafana.sw-bench.de/d/qjIIww4Zz/locust?orgId=1",
			"--override-plan-name", locustFile,
			"--test-env", runEnv,
			"--test-version", platformHash,
		}

		ctx, cancel := context.WithCancel(cmd.Context())

		setupLocustCommandKill(cancel)

		locustWorkers := make([]string, 0)

		for i := 0; i < workerAmount; i++ {
			locustWorkers = append(locustWorkers, fmt.Sprintf("locust-worker@%d", i))
		}

		log.Infof("Stopping all workers")
		stopWorkers := runExternalCommandWithOutputOnLocust(ctx, append([]string{"systemctl", "stop"}, locustWorkers...)...)
		if err := stopWorkers.Run(); err != nil {
			killLocust()
			return err
		}

		log.Infof("Reconfigure the workers")
		workerTpl, err := ioutil.ReadFile("roles/locust/templates/locust-worker-service.j2")
		if err != nil {
			return err
		}

		workerArgs := []string{
			"/usr/bin/python3",
			"/usr/local/bin/locust",
			"-f",
			fmt.Sprintf("/root/platform/src/Core/DevOps/Locust/scenarios/%s.py", locustFile),
			"--worker",
		}

		if useTimeScale {
			workerArgs = append(workerArgs, timeScaleArgs...)
		}

		newWorkerTpl := re.ReplaceAllString(string(workerTpl), fmt.Sprintf("ExecStart=%s", strings.Join(workerArgs, " ")))

		if err := ioutil.WriteFile("locust-worker.service", []byte(newWorkerTpl), os.ModePerm); err != nil {
			return err
		}

		defer os.Remove("locust-worker.service")

		scp := exec.CommandContext(ctx, "scp", "-o", "StrictHostKeyChecking=no", "-o", "UserKnownHostsFile=/dev/null", "-o", "LogLevel=quiet", "locust-worker.service", "root@locust-1.sw-bench.de:/etc/systemd/system/locust-worker@.service")
		if err := scp.Run(); err != nil {
			return err
		}

		reloadSystemd := runExternalCommandWithOutputOnLocust(ctx, "systemctl", "daemon-reload")
		if err := reloadSystemd.Run(); err != nil {
			killLocust()
			return err
		}

		log.Infof("Start all workers")
		startWorkers := runExternalCommandWithOutputOnLocust(ctx, append([]string{"systemctl", "start"}, locustWorkers...)...)
		if err := startWorkers.Run(); err != nil {
			killLocust()
			return err
		}

		log.Infof("Starting locust master")
		locustArgs := []string{
			"locust",
			"-f",
			fmt.Sprintf("/root/platform/src/Core/DevOps/Locust/scenarios/%s.py", locustFile),
			"--master",
			"-H", host,
			"-u", users,
			"-r", spawnRate,
			"--autostart",
			"--exit-code-on-error", "0",
			"--html", "/tmp/locust.html",
			"--csv", "locust",
			"--only-summary",
		}

		if headless {
			locustArgs = append(locustArgs, "--headless")
		}

		if runTime != "" {
			locustArgs = append(locustArgs, "-t", runTime, "--autoquit", "1")
		}

		if useTimeScale {
			locustArgs = append(locustArgs, timeScaleArgs...)
		}

		if len(args) > 0 {
			locustArgs = append(locustArgs, args...)
		}

		start := time.Now().Unix()

		locustMain := runExternalCommandWithOutputOnLocust(ctx, locustArgs...)

		if err := locustMain.Run(); err != nil {
			killLocust()

			log.Infof("Locust exited with error: %v, ", err)
		}

		log.Infof("Locust finished it task")

		filePath := fmt.Sprintf("%s/%s.html", time.Now().Format("2006/01/02"), htmlName)

		minioUpload := runExternalCommandWithOutputOnLocust(ctx, "mc", "cp", "/tmp/locust.html", fmt.Sprintf("storage/locust/%s", filePath))

		if err := minioUpload.Run(); err != nil {
			return err
		}

		resultFile := fmt.Sprintf("https://assets.sw-bench.de/locust/%s", filePath)

		log.Infof("Uploaded locust result at %s", resultFile)

		if headless && locustFile == "integration-benchmark" {
			if err := notifySlack(resultFile, runEnv, locustFile, start, ctx); err != nil {
				return err
			}
		}

		return nil
	},
}

func notifySlack(assetLink string, runEnv string, locustFile string, start int64, ctx context.Context) error {
	slackHook := os.Getenv("SLACK_WEBHOOK_URL")
	if slackHook == "" {
		return nil
	}

	csvContent := exec.CommandContext(ctx, "ssh", "-o", "StrictHostKeyChecking=no", "-o", "UserKnownHostsFile=/dev/null", "-o", "LogLevel=quiet", "root@locust-1.sw-bench.de", "cat", "locust_stats.csv")
	txt, err := csvContent.Output()
	if err != nil {
		return err
	}

	r := csv.NewReader(bytes.NewReader(txt))
	rows, err := r.ReadAll()

	if err != nil {
		return err
	}

	for _, row := range rows {
		if row[1] == "Aggregated" {
			message := fmt.Sprintf("Job: %s (%s), RPS: %s, Total Requests: %s, Failed Requests: %s\n\nHTML Report: %s\nServer Stats: %s\nAll runes: %s",
				locustFile,
				runEnv,
				row[9],
				row[2],
				row[3],
				assetLink,
				fmt.Sprintf("https://grafana.sw-bench.de/d/xfpJB9FGz/server-stats?orgId=1&from=%d000&to=%d000", start, time.Now().Unix()),
				"https://grafana.sw-bench.de/d/rtrgXdxnk/locust-testruns?orgId=1",
			)

			payload, _ := json.Marshal(map[string]string{"text": message})

			slack, _ := http.NewRequestWithContext(ctx, "POST", slackHook, bytes.NewReader(payload))

			_, err := http.DefaultClient.Do(slack)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func setupLocustCommandKill(cancel context.CancelFunc) {
	ch := make(chan os.Signal, 1)

	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)

	go func() {
		signalType := <-ch
		signal.Stop(ch)
		cancel()

		killLocust()
		log.Println("Exit command received. Exiting...")
		log.Println("Signal type : ", signalType)

		os.Exit(0)
	}()
}

func init() {
	rootCmd.AddCommand(locustCmd)
	locustCmd.PersistentFlags().Int("worker", 8, "Amount of workers")
	locustCmd.PersistentFlags().String("host", "http://sw-bench.de", "Run locust against")
	locustCmd.PersistentFlags().String("file", "integration-benchmark", "Specify locustfile")
	locustCmd.PersistentFlags().String("users", "100", "Users")
	locustCmd.PersistentFlags().String("spawn-rate", "10", "Spawn rate")
	locustCmd.PersistentFlags().Bool("headless", false, "Run Locust without UI")
	locustCmd.PersistentFlags().String("run-time", "", "Run Time")
	locustCmd.PersistentFlags().String("run-env", "demo-data", "Show ")
	locustCmd.PersistentFlags().String("html-name", time.Now().Format("15-04-05"), "HTML Filename")
	locustCmd.PersistentFlags().Bool("time-scale", false, "Use timescale db")
}

func killLocust() {
	killAll := runExternalCommandWithOutputOnLocust(context.Background(), "killall", "-9", "/usr/bin/python3")
	err := killAll.Run()
	if err != nil {
		log.Error(err)
	}
}

func runExternalCommandWithOutputOnLocust(ctx context.Context, args ...string) *exec.Cmd {
	defaultArgs := []string{"-o", "StrictHostKeyChecking=no", "-o", "UserKnownHostsFile=/dev/null", "-o", "LogLevel=quiet", "root@locust-1.sw-bench.de"}
	runArgs := append(defaultArgs, args...)

	process := exec.CommandContext(ctx, "ssh", runArgs...)
	process.Stdout = os.Stdout
	process.Stderr = os.Stderr
	process.Stdin = os.Stdin

	return process
}

func getPlatformHash(ctx context.Context) (string, error) {
	defaultArgs := []string{"-o", "StrictHostKeyChecking=no", "-o", "UserKnownHostsFile=/dev/null", "-o", "LogLevel=quiet", "root@app-1.sw-bench.de", "git", "-C", "/var/www/html", "rev-parse", "--short", "HEAD"}

	process := exec.CommandContext(ctx, "ssh", defaultArgs...)

	out, err := process.Output()

	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}
