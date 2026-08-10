package main

import "fmt"

// swap menukar nilai dua integer melalui pointer.
func swap(a, b *int) {
	*a, *b = *b, *a
}

// updateSlice menambahkan item baru ke slice melalui pointer.
func updateSlice(s *[]string, newItem string) {
	*s = append(*s, newItem)
}

// Pass by value
func passByValue(x int) {
	x = 100
}

// Pass by pointer
func passByPointer(x *int) {
	*x = 100
}

func main() {
	// ========================================
	// 1. Function swap
	// ========================================

	a := 10
	b := 20

	fmt.Println("=== SWAP ===")
	fmt.Println("Sebelum ditukar:")
	fmt.Println("a =", a)
	fmt.Println("b =", b)

	swap(&a, &b)

	fmt.Println("Setelah ditukar:")
	fmt.Println("a =", a)
	fmt.Println("b =", b)

	// ========================================
	// 2. Function updateSlice
	// ========================================

	buah := []string{"Apel", "Jeruk"}

	fmt.Println("\n=== UPDATE SLICE ===")
	fmt.Println("Sebelum update:", buah)

	updateSlice(&buah, "Mangga")

	fmt.Println("Setelah update:", buah)

	// ========================================
	// 3. Pass by value
	// ========================================

	nilaiValue := 50

	fmt.Println("\n=== PASS BY VALUE ===")
	fmt.Println("Sebelum function:", nilaiValue)

	passByValue(nilaiValue)

	fmt.Println("Setelah function:", nilaiValue)

	// ========================================
	// 4. Pass by pointer
	// ========================================

	nilaiPointer := 50

	fmt.Println("\n=== PASS BY POINTER ===")
	fmt.Println("Sebelum function:", nilaiPointer)

	passByPointer(&nilaiPointer)

	fmt.Println("Setelah function:", nilaiPointer)
}