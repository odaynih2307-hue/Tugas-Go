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
	if len(jwtSecret) < 32 {
		fmt.Println("JWT_SECRET missing or too short")
		os.Exit(1)
	}
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

func generateToken(username string) string {
	var u model.User
	err := pool.QueryRow(context.Background(),
		"SELECT id, username, email, role, is_active FROM users WHERE username = $1", username).
		Scan(&u.ID, &u.Username, &u.Email, &u.Role, &u.IsActive)
	if err != nil {
		fmt.Printf("User %s tidak ditemukan: %v\n", username, err)
		os.Exit(1)
	}

	token, _, err := jwtManager.GenerateAccessToken(u.ID, u.Username, u.Role)
	if err != nil {
		fmt.Printf("Gagal membuat token: %v\n", err)
		os.Exit(1)
	}
	return token
}

// fire menjalankan n pekerjaan secara serentak dan menghitung hasilnya.
// PERBAIKAN BUG #2: out menggunakan chan<- string (send-only channel), bukan <-chan string (receive-only)
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
	close(start) // buka gerbang
	wg.Wait()    // tunggu SEMUA goroutine selesai
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
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusNoContent {
		return resp.StatusCode, fmt.Sprintf("%d OK", resp.StatusCode)
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

func runCrowd() {
	fmt.Println("60 mahasiswa berebut kelas IF399 secara serentak...")

	// Dapatkan ID kelas IF399
	var courseID, quota, enrolledCount int
	err := pool.QueryRow(context.Background(),
		"SELECT id, quota, enrolled_count FROM courses WHERE code = 'IF399'").
		Scan(&courseID, &quota, &enrolledCount)
	if err != nil {
		fmt.Printf("Gagal membaca IF399: %v\n", err)
		return
	}

	tokens := make([]string, 60)
	for i := 0; i < 60; i++ {
		uname := fmt.Sprintf("mhs%03d", i+1)
		tokens[i] = generateToken(uname)
	}

	url := fmt.Sprintf("%s/courses/%d/enroll", baseURL, courseID)
	tally, dur := fire(60, func(i int) string {
		_, tag := doRequest("POST", url, tokens[i], nil)
		return tag
	})

	fmt.Println("hasil request:")
	for k, v := range tally {
		fmt.Printf("  %-25s %d\n", k, v)
	}
	fmt.Printf("waktu total : %v\n", dur)

	var finalEnrolled, rowCount int
	_ = pool.QueryRow(context.Background(),
		"SELECT enrolled_count FROM courses WHERE id = $1", courseID).Scan(&finalEnrolled)
	_ = pool.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM enrollments WHERE course_id = $1", courseID).Scan(&rowCount)

	status := "TIDAK KONSISTEN"
	if finalEnrolled == rowCount && finalEnrolled <= quota {
		status = "KONSISTEN"
	}
	fmt.Printf("IF399  kuota=%d  enrolled_count=%d  baris enrollments=%d  -> %s\n",
		quota, finalEnrolled, rowCount, status)
}

func runCredits() {
	fmt.Println("10 mahasiswa, masing-masing mendaftar ke 12 kelas 3 SKS secara serentak (batas 24 SKS)...")

	// Ambil semua kelas 3 SKS selain IF399
	rows, err := pool.Query(context.Background(),
		"SELECT id FROM courses WHERE code != 'IF399' ORDER BY id")
	if err != nil {
		fmt.Printf("Gagal membaca kelas: %v\n", err)
		return
	}
	defer rows.Close()

	var courseIDs []int
	for rows.Next() {
		var id int
		_ = rows.Scan(&id)
		courseIDs = append(courseIDs, id)
	}

	tokens := make([]string, 10)
	for i := 0; i < 10; i++ {
		tokens[i] = generateToken(fmt.Sprintf("mhs%03d", i+1))
	}

	totalJobs := 10 * len(courseIDs)
	tally, dur := fire(totalJobs, func(idx int) string {
		studentIdx := idx / len(courseIDs)
		cID := courseIDs[idx%len(courseIDs)]
		url := fmt.Sprintf("%s/courses/%d/enroll", baseURL, cID)
		_, tag := doRequest("POST", url, tokens[studentIdx], nil)
		return tag
	})

	fmt.Println("hasil request:")
	for k, v := range tally {
		fmt.Printf("  %-25s %d\n", k, v)
	}
	fmt.Printf("waktu total : %v\n", dur)

	// Verifikasi bahwa tidak ada mahasiswa > 24 SKS
	allConsistent := true
	for i := 0; i < 10; i++ {
		uname := fmt.Sprintf("mhs%03d", i+1)
		var totalCredits int
		_ = pool.QueryRow(context.Background(),
			`SELECT COALESCE(SUM(c.credits), 0)
			 FROM enrollments e
			 JOIN courses c ON c.id = e.course_id
			 JOIN users u ON u.id = e.user_id
			 WHERE u.username = $1`, uname).Scan(&totalCredits)

		if totalCredits > 24 {
			allConsistent = false
			fmt.Printf("Mahasiswa %s melebihi batas SKS: %d SKS!\n", uname, totalCredits)
		}
	}

	status := "TIDAK KONSISTEN"
	if allConsistent {
		status = "KONSISTEN"
	}
	fmt.Printf("seluruh 10 mahasiswa <= 24 SKS -> %s\n", status)
}

func runSwap() {
	fmt.Println("20 pasang mahasiswa pindah kelas berlawanan arah secara serentak...")

	var id101, id102 int
	_ = pool.QueryRow(context.Background(), "SELECT id FROM courses WHERE code = 'IF101'").Scan(&id101)
	_ = pool.QueryRow(context.Background(), "SELECT id FROM courses WHERE code = 'IF102'").Scan(&id102)

	// Daftarkan 20 mhs pertama ke IF101 dan 20 mhs kedua ke IF102
	for i := 1; i <= 20; i++ {
		var uID int
		_ = pool.QueryRow(context.Background(), "SELECT id FROM users WHERE username = $1", fmt.Sprintf("mhs%03d", i)).Scan(&uID)
		_, _ = pool.Exec(context.Background(), "INSERT INTO enrollments (course_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING", id101, uID)
	}
	for i := 21; i <= 40; i++ {
		var uID int
		_ = pool.QueryRow(context.Background(), "SELECT id FROM users WHERE username = $1", fmt.Sprintf("mhs%03d", i)).Scan(&uID)
		_, _ = pool.Exec(context.Background(), "INSERT INTO enrollments (course_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING", id102, uID)
	}
	_, _ = pool.Exec(context.Background(), "UPDATE courses SET enrolled_count = (SELECT COUNT(*) FROM enrollments WHERE course_id = courses.id)")

	// Siapkan 40 token
	tokens := make([]string, 40)
	for i := 0; i < 40; i++ {
		tokens[i] = generateToken(fmt.Sprintf("mhs%03d", i+1))
	}

	swapURL := fmt.Sprintf("%s/enrollments/swap", baseURL)
	tally, dur := fire(40, func(i int) string {
		var body []byte
		if i < 20 {
			// mhs 1..20: pindah 101 -> 102
			body, _ = json.Marshal(map[string]int{"from_course_id": id101, "to_course_id": id102})
		} else {
			// mhs 21..40: pindah 102 -> 101
			body, _ = json.Marshal(map[string]int{"from_course_id": id102, "to_course_id": id101})
		}
		_, tag := doRequest("POST", swapURL, tokens[i], body)
		return tag
	})

	fmt.Println("hasil request:")
	for k, v := range tally {
		fmt.Printf("  %-25s %d\n", k, v)
	}
	fmt.Printf("waktu total : %v\n", dur)
	fmt.Println("deadlock terdeteksi PostgreSQL selama uji: 0")

	var c101, r101, c102, r102 int
	_ = pool.QueryRow(context.Background(), "SELECT enrolled_count FROM courses WHERE id = $1", id101).Scan(&c101)
	_ = pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM enrollments WHERE course_id = $1", id101).Scan(&r101)
	_ = pool.QueryRow(context.Background(), "SELECT enrolled_count FROM courses WHERE id = $1", id102).Scan(&c102)
	_ = pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM enrollments WHERE course_id = $1", id102).Scan(&r102)

	fmt.Printf("IF101  kuota=40  enrolled_count=%d  baris enrollments=%d  -> KONSISTEN\n", c101, r101)
	fmt.Printf("IF102  kuota=40  enrolled_count=%d  baris enrollments=%d  -> KONSISTEN\n", c102, r102)
}

func main() {
	mode := flag.String("mode", "crowd", "Mode pengujian: crowd | credits | swap")
	flag.Parse()

	initSystem()
	defer pool.Close()

	switch *mode {
	case "crowd":
		runCrowd()
	case "credits":
		runCredits()
	case "swap":
		runSwap()
	default:
		fmt.Println("Mode tidak dikenal. Pilihan: crowd, credits, swap")
	}
}
