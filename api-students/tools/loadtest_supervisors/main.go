package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"api-students/app/model"
	"api-students/config"
	"api-students/database"
	"api-students/helper"
)

var (
	baseURL    = "http://localhost:3000/api/v1"
	jwtManager *helper.JWTManager
	pool       *pgxpool.Pool
)

func initSystem() {
	config.LoadEnv()
	jwtSecret := config.GetEnv("JWT_SECRET", "")
	jwtManager = helper.NewJWTManager(
		jwtSecret,
		config.GetEnv("JWT_ISSUER", "praktikum-backend"),
		time.Duration(config.GetEnvInt("JWT_ACCESS_TTL_MINUTES", 15))*time.Minute,
	)

	var err error
	pool, err = database.NewPool(context.Background())
	if err != nil {
		fmt.Printf("Gagal terhubung database: %v\n", err)
		os.Exit(1)
	}
}

func getAdminToken() string {
	var u model.User
	err := pool.QueryRow(context.Background(),
		"SELECT id, username, email, role, is_active FROM users WHERE username = 'admin'").
		Scan(&u.ID, &u.Username, &u.Email, &u.Role, &u.IsActive)
	if err != nil {
		fmt.Printf("Admin user tidak ditemukan: %v\n", err)
		os.Exit(1)
	}

	token, _, err := jwtManager.GenerateAccessToken(u.ID, u.Username, u.Role)
	if err != nil {
		fmt.Printf("Gagal membuat token admin: %v\n", err)
		os.Exit(1)
	}
	return token
}

func fire(n int, job func(i int) string) (map[string]int, time.Duration) {
	var wg sync.WaitGroup
	start := make(chan struct{})
	results := make(chan string, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int, out chan<- string) {
			defer wg.Done()
			<-start
			out <- job(i)
		}(i, results)
	}

	t0 := time.Now()
	close(start)
	wg.Wait()
	dur := time.Since(t0)
	close(results)

	tally := map[string]int{}
	for r := range results {
		tally[r]++
	}
	return tally, dur
}

func doRequest(method, url, token string, body []byte) (int, string) {
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return 0, err.Error()
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err.Error()
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
		return resp.StatusCode, fmt.Sprintf("%d OK/Created", resp.StatusCode)
	}

	var res struct {
		Code string `json:"code"`
	}
	_ = json.Unmarshal(respBody, &res)
	if res.Code != "" {
		return resp.StatusCode, fmt.Sprintf("%d %s", resp.StatusCode, res.Code)
	}
	return resp.StatusCode, fmt.Sprintf("%d Error", resp.StatusCode)
}

func resetSupervisions() {
	_, _ = pool.Exec(context.Background(), "DELETE FROM supervisions")
	_, _ = pool.Exec(context.Background(),
		"UPDATE supervisors SET supervised_count = (SELECT COUNT(*) FROM supervisions WHERE supervisor_id = supervisors.id)")
}

func runSupervisorsLoadTest(supervisorID int, studentCount int) {
	var sup model.Supervisor
	err := pool.QueryRow(context.Background(),
		"SELECT id, name, quota, supervised_count FROM supervisors WHERE id = $1", supervisorID).
		Scan(&sup.ID, &sup.Name, &sup.Quota, &sup.SupervisedCount)
	if err != nil {
		fmt.Printf("Gagal membaca supervisor: %v\n", err)
		return
	}

	// Ambil student_id yang valid
	rows, err := pool.Query(context.Background(), "SELECT id FROM students ORDER BY id LIMIT $1", studentCount)
	if err != nil {
		fmt.Printf("Gagal membaca data students: %v\n", err)
		return
	}
	defer rows.Close()

	var studentIDs []int
	for rows.Next() {
		var sID int
		_ = rows.Scan(&sID)
		studentIDs = append(studentIDs, sID)
	}

	if len(studentIDs) < studentCount {
		fmt.Printf("Data students kurang! Dibutuhkan %d, ditemukan %d\n", studentCount, len(studentIDs))
		return
	}

	token := getAdminToken()
	url := fmt.Sprintf("%s/supervisors/%d/assign", baseURL, supervisorID)

	fmt.Printf("%d mahasiswa serentak ditetapkan ke pembimbing '%s' (kuota %d)...\n",
		studentCount, sup.Name, sup.Quota)

	tally, dur := fire(studentCount, func(i int) string {
		body, _ := json.Marshal(map[string]int{"student_id": studentIDs[i]})
		_, tag := doRequest("POST", url, token, body)
		return tag
	})

	fmt.Println("hasil request:")
	for k, v := range tally {
		fmt.Printf("  %-28s %d\n", k, v)
	}
	fmt.Printf("waktu total : %v\n", dur)

	var finalSupervised, rowCount int
	_ = pool.QueryRow(context.Background(),
		"SELECT supervised_count FROM supervisors WHERE id = $1", supervisorID).Scan(&finalSupervised)
	_ = pool.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM supervisions WHERE supervisor_id = $1", supervisorID).Scan(&rowCount)

	status := "TIDAK KONSISTEN"
	if finalSupervised == rowCount && finalSupervised <= sup.Quota {
		status = "KONSISTEN"
	}
	fmt.Printf("Supervisor ID %d  kuota=%d  supervised_count=%d  baris supervisions=%d  -> %s\n",
		supervisorID, sup.Quota, finalSupervised, rowCount, status)
}

func main() {
	runs := flag.Int("runs", 3, "Jumlah iterasi pengujian")
	flag.Parse()

	initSystem()
	defer pool.Close()

	for r := 1; r <= *runs; r++ {
		fmt.Printf("\n========== UJI BEBAN SUPERVISORS - PERCOBAAN %d ==========\n", r)
		resetSupervisions()
		runSupervisorsLoadTest(1, 30)
	}
}
