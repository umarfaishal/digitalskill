package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

var db *sql.DB
var currentUserID int
var reader = bufio.NewReader(os.Stdin)

func connectDB() {
	var err error
	db, err = sql.Open("mysql", "root:@tcp(127.0.0.1:3306)/bank")
	if err != nil {
		log.Fatal(err)
	}
}

func input(prompt string) string {
	fmt.Print(prompt)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func register() {
	username := input("username: ")
	pin := input("PIN: ")

	_, err := db.Exec("INSERT INTO akun(username, pin) VALUES (?, ?)", username, pin)
	if err != nil {
		fmt.Println("Gagal Registrasi:", err)
		return
	}
	fmt.Println("Registrasi Berhasil")
}

func login() bool {
	username := input("username: ")
	pin := input("pin: ")

	row := db.QueryRow("SELECT id FROM akun WHERE username=? AND pin=?", username, pin)
	err := row.Scan(&currentUserID)
	if err != nil {
		fmt.Println("Login Gagal")
		return false
	} else {
		fmt.Println("Login Berhasil")
		return true
	}
}

func cekSaldo() {
	var saldo float64
	err := db.QueryRow("SELECT saldo FROM akun WHERE id=?", currentUserID).Scan(&saldo)
	if err != nil {
		fmt.Println("Gagal melihat saldo", err)
		return
	}
	fmt.Printf("Saldo anda saat ini: %.2f\n", saldo)
}

func tambahSaldo() {
	amountStr := input("Jumlah tambah saldo: ")
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil || amount <= 0 {
		fmt.Println("Jumlah tidak valid")
		return
	}

	tx, err := db.Begin()
	if err != nil {
		fmt.Println("Gagal memulai transaksi", err)
		return
	}

	_, err = tx.Exec("UPDATE akun SET saldo = saldo + ? WHERE id=?", amount, currentUserID)
	if err != nil {
		fmt.Println("Gagal menambah saldo", err)
		return
	}

	_, err = tx.Exec("INSERT INTO transactions(account_id, type, amount) VALUES (?, 'deposit', ?)", currentUserID, amount)
	if err != nil {
		tx.Rollback()
		fmt.Println("Gagal mencatat transaksi", err)
		return
	}

	err = tx.Commit()
	if err != nil {
		fmt.Println("Gagal menyimpan perubahan", err)
		return
	}

	fmt.Println("Saldo berhasil ditambahkan")
}

func kurangiSaldo() {
	amountStr := input("Jumlah pengambilan: ")
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil || amount <= 0 {
		fmt.Println("Jumlah tidak valid")
		return
	}

	var saldo float64
	err = db.QueryRow("SELECT saldo FROM akun WHERE id=?", currentUserID).Scan(&saldo)
	if err != nil {
		fmt.Println("Gagal mengambil saldo", err)
		return
	}

	if saldo < amount {
		fmt.Println("Saldo tidak cukup")
		return
	}

	_, err = db.Exec("UPDATE akun SET saldo = saldo - ? WHERE id=?", amount, currentUserID)
	if err != nil {
		fmt.Println("Gagal mengurangi saldo")
		return
	}

	_, err = db.Exec("INSERT INTO transactions(account_id, type, amount) VALUES (?, 'withdraw', ?)", currentUserID, amount)
	if err != nil {
		fmt.Println("Gagal mencatat transaksi")
	}
	fmt.Println("Saldo berhasil diambil")
}

func transfer() {
	target := input("nama akun tujuan: ")
	amountStr := input("Jumlah transfer: ")
	amount, _ := strconv.ParseFloat(amountStr, 64)

	var targetID int
	err := db.QueryRow("SELECT id FROM akun WHERE username=?", target).Scan(&targetID)
	if err != nil {
		fmt.Println("Akun tujuan tidak ditemukan")
		return
	}

	tx, _ := db.Begin()
	_, err1 := tx.Exec("UPDATE akun SET saldo = saldo - ? WHERE id=? AND saldo >= ?", amount, currentUserID, amount)
	_, err2 := tx.Exec("UPDATE akun SET saldo = saldo + ? WHERE id=?", amount, targetID)
	if err1 != nil || err2 != nil {
		tx.Rollback()
		fmt.Println("Transfer gagal")
		return
	}
	tx.Exec("INSERT INTO transactions(account_id, type, amount, target_id) VALUES (?, 'transfer_in', ?, ?)", currentUserID, amount, target)
	tx.Commit()
	fmt.Println("Transfer berhasil")
}

func lihatRiwayat() {
	rows, err := db.Query("SELECT type, amount, target_id, created_at FROM transactions WHERE account_id = ? ORDER BY created_at DESC", currentUserID)
	if err != nil {
		fmt.Println("Gagal mengambil riwayat transaksi")
		return
	}
	defer rows.Close()

	fmt.Println("\n=== Riwayat Transaksi ===")
	for rows.Next() {
		var tipe, targetID string
		var amount float64
		var date string

		err := rows.Scan(&tipe, &amount, &targetID, &date)
		if err != nil {
			fmt.Println("Gagal membaca data transaksi", err)
			return
		}
		if tipe == "transfer_out" || tipe == "transfer_in" {
			fmt.Printf("[%s] %s sebesar %.2f ke/dari %s pada %s\n", tipe, tipe, amount, targetID, date)
		} else {
			fmt.Printf("[%s] sebesar %2.f pada %s\n", tipe, amount, date)
		}
	}
}

func main() {
	connectDB()
	defer db.Close()

	for {
		fmt.Println("\n--- SELAMAT DATANG ---")
		fmt.Println("1. Register")
		fmt.Println("2. Login")
		fmt.Println("3. Keluar")

		option := input("Pilih menu: ")

		switch option {
		case "1":
			register()
		case "2":
			if login() {
				goto menuUtama
			}
		case "3":
			return
		default:
			fmt.Println("Pilihan tidak valid")
		}
	}

menuUtama:
	for {
		fmt.Println("\n--- MENU UTAMA ---")
		fmt.Println("1. Cek Saldo")
		fmt.Println("2. Tambah Saldo")
		fmt.Println("3. Ambil Saldo")
		fmt.Println("4. Transfer")
		fmt.Println("5. Riwayat Transaksi")
		fmt.Println("6. Keluar")

		option := input("Pilih menu: ")
		switch option {
		case "1":
			cekSaldo()
		case "2":
			tambahSaldo()
		case "3":
			kurangiSaldo()
		case "4":
			transfer()
		case "5":
			lihatRiwayat()
		case "6":
			currentUserID = 0
			fmt.Println("Logout berhasil")
			main()
			return
		default:
			fmt.Println("Pilihan tidak valid")
		}
	}
}
