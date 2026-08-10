package main

import "fmt"

func main() {
	// 1. Lima variabel dengan tipe berbeda
	var nama string = "Ody"
	var umur int = 20
	var ipk float64 = 3.75
	var mahasiswaAktif bool = true
	var mataKuliah []string = []string{
		"Pemrograman Backend",
		"Basis Data",
		"UI/UX",
	}

	fmt.Println("=== VARIABEL ===")
	fmt.Println("Nama:", nama)
	fmt.Println("Umur:", umur)
	fmt.Println("IPK:", ipk)
	fmt.Println("Mahasiswa Aktif:", mahasiswaAktif)
	fmt.Println("Mata Kuliah:", mataKuliah)

	// 2. Map untuk menyimpan data mahasiswa
	mahasiswa := make(map[string]int)

	// Menambahkan data
	mahasiswa["Ody"] = 90
	mahasiswa["Budi"] = 85
	mahasiswa["Sari"] = 95

	fmt.Println("\n=== DATA MAHASISWA ===")
	fmt.Println(mahasiswa)

	// 3. Membaca data dengan pengecekan keberadaan
	nilai, ada := mahasiswa["Ody"]

	if ada {
		fmt.Println("\nNilai Ody:", nilai)
	} else {
		fmt.Println("\nData Ody tidak ditemukan")
	}

	// Mengecek data mahasiswa yang tidak ada
	nilaiAndi, adaAndi := mahasiswa["Andi"]

	if adaAndi {
		fmt.Println("Nilai Andi:", nilaiAndi)
	} else {
		fmt.Println("Data Andi tidak ditemukan")
	}

	// 4. Menghapus data
	delete(mahasiswa, "Budi")

	fmt.Println("\n=== SETELAH BUDI DIHAPUS ===")
	fmt.Println(mahasiswa)

	// 5. Menelusuri seluruh isi map
	fmt.Println("\n=== SELURUH DATA MAHASISWA ===")

	for nama, nilai := range mahasiswa {
		fmt.Println(nama, ":", nilai)
	}
}