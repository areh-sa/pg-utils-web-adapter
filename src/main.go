package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"
)

// External utils
const (
	pgDump     = "pg_dump"
	pgRestore  = "pg_restore"
	pgCreateDb = "createdb"
)

// Message
const (
	messageEnvNotSet = "environment variables not set"
)

// Status values
const (
	statusOk      = "ok"
	statusError   = "error"
	statusSkipped = "skipped"
)

type actionResponse struct {
	Status  string `json:"status"`
	Action  string `json:"action,omitempty"`
	Message string `json:"message,omitempty"`
	File    string `json:"file,omitempty"`
}

type pgConnection struct {
	Host string
	Port string
	Db   string
	User string
	Pass string
}

var pgEnvSet *pgConnection

func main() {
	if !checkPgUtils() {
		log.Fatal("[ERR] PostgreSQL utils not found")
		return
	}

	pgEnvSet = loadEnvSettings()

	log.Printf("[INFO] PostgreSQL connection settings set in environment variables: %v\n", pgEnvSet != nil)

	http.HandleFunc("/status", statusHandler)
	http.HandleFunc("/backup", backupHandler)
	http.HandleFunc("/backup-db", backupFullHandler)
	http.HandleFunc("/restore", restoreHandler)
	http.HandleFunc("/restore-db", restoreFullHandler)

	log.Println("[INFO] Listening port 80")
	err := http.ListenAndServe(":80", nil)
	if err != nil {
		log.Fatal("[ERR] ListenAndServe: ", err)
	}
}

func loadEnvSettings() *pgConnection {
	result := pgConnection{
		getEnvVariableWithDefault("PG_HOST", ""),
		getEnvVariableWithDefault("PG_PORT", "5432"),
		getEnvVariableWithDefault("PG_DB", ""),
		getEnvVariableWithDefault("PG_USER", ""),
		getEnvVariableWithDefault("PG_PASS", "")}

	return checkSettings(&result)
}

func checkSettings(pgSettings *pgConnection) *pgConnection {
	if pgSettings.Host == "" || pgSettings.Db == "" || pgSettings.User == "" {
		return nil
	}
	return pgSettings
}

func getEnvVariableWithDefault(envVariable, defaultValue string) string {
	value := os.Getenv(envVariable)
	if value == "" {
		return defaultValue
	}
	return value
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("[INFO] Request URI: %s, handler: %s", r.RequestURI, "statusHandler")
	writeResponse(w, http.StatusOK, actionResponse{Action: "status", Status: statusOk})
}

func backupHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("[INFO] Request URI: %s, handler: %s", r.RequestURI, "backupHandler")

	if pgEnvSet == nil {
		writeResponse(w, http.StatusNotImplemented,
			actionResponse{Action: "backup", Status: statusError, Message: messageEnvNotSet})
		return
	}

	// do default backup
	fileName := fmt.Sprintf("/backups/%s_%s_%s.dump", pgEnvSet.Host, pgEnvSet.Db, time.Now().Format("20060102_150405"))

	args := []string{
		"-h", pgEnvSet.Host,
		"-p", pgEnvSet.Port,
		"-U", pgEnvSet.User,
		"-Fc",
		pgEnvSet.Db,
		"-f", fileName,
	}

	returnExecutionResult(w, "backup", pgDump, args, fileName)
}

func backupFullHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("[INFO] Request URI: %s, handler: %s", r.RequestURI, "backupFullHandler")

	// do backup

	args := []string{
		"ya.ru",
		"-c", "3",
	}

	returnExecutionResult(w, "backupFull", "pingTest", args, "")
}

func restoreHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("[INFO] Request URI: %s, handler: %s", r.RequestURI, "restoreHandler")

	if pgEnvSet == nil {
		writeResponse(w, http.StatusNotImplemented,
			actionResponse{Action: "restore", Status: statusError, Message: messageEnvNotSet})
		return
	}

	// do default restore

	writeResponse(w, http.StatusOK, actionResponse{Action: "restore", Status: statusSkipped})
}

func restoreFullHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("[INFO] Request URI: %s, handler: %s", r.RequestURI, "restoreFullHandler")

	// do restore

	args := []string{
		"ya.ru",
		"-c", "3",
	}

	returnExecutionResult(w, "restoreFull", "ping", args, "")
}

func returnExecutionResult(w http.ResponseWriter, actionName, app string, args []string, fileName string) {
	status := statusError
	httpStatus := http.StatusInternalServerError
	resultFile := ""
	res, out := executeWithOutput(app, args, true)
	if res {
		status = statusOk
		httpStatus = http.StatusOK
		resultFile = fileName
	}

	writeResponse(w, httpStatus, actionResponse{Action: actionName, Status: status, Message: out, File: resultFile})
}

func writeResponse(w http.ResponseWriter, responseStatus int, responseData actionResponse) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(responseStatus)

	err := json.NewEncoder(w).Encode(responseData)
	if err != nil {
		log.Printf("[WARN] failed to send response, %v", err)
	}
}

func checkPgUtils() bool {
	args := []string{"--help"}
	return execute(pgDump, args) && execute(pgRestore, args) && execute(pgCreateDb, args)
}

func execute(app string, args []string) bool {
	res, _ := executeWithOutput(app, args, false)
	return res
}

func executeWithOutput(app string, args []string, printOutput bool) (bool, string) {
	cmd := exec.Command(app, args...)

	if pgEnvSet.Pass != "" {
		cmd.Env = []string{"PGPASSWORD=" + pgEnvSet.Pass}
	}

	out, err := cmd.Output()
	if err != nil {
		fmt.Printf("[ERR] Can't execute app %v, error:\n%v", app, err.Error())
		return false, err.Error()
	}

	if printOutput {
		fmt.Printf("[INFO] External app output\n%v\n", string(out))
	}

	return true, string(out)
}
