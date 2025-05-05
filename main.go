package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	_ "github.com/lib/pq"
)

type PostgresConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	DBName   string `json:"dbname"`
}

type BackupConfig struct {
	Prefix    string `json:"prefix"`    // Префикс для таблиц бэкапа (по умолчанию "autobackup")
	Retention int    `json:"retention"` // Количество дней хранения бэкапов (по умолчанию 14)
}

// Config структура для хранения параметров конфигурации
type Config struct {
	Postgres PostgresConfig `json:"postgres"`
	Backup   BackupConfig   `json:"backup"`
}

func main() {
	run := flag.Bool("run", false, "Normal run instead of test run?")

	// Загрузка конфигурации
	config, err := loadConfig("config.ini")
	if err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	// Подключение к PostgreSQL
	db, err := connectToPostgres(&config.Postgres)
	if err != nil {
		log.Fatalf("Ошибка подключения к PostgreSQL: %v", err)
	}
	defer db.Close()

	// Выполнение задачи бэкапа
	err = performBackup(db, config.Backup.Prefix, config.Backup.Retention, *run)
	if err != nil {
		log.Fatalf("Ошибка выполнения бэкапа: %v", err)
	}

	log.Println("backup done")
}

// loadConfig загружает конфигурацию из файла
func loadConfig(filename string) (*Config, error) {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения файла конфигурации: %v", err)
	}

	var config Config
	err = json.Unmarshal(file, &config)
	if err != nil {
		return nil, fmt.Errorf("ошибка парсинга конфигурации: %v", err)
	}

	// Установка значений по умолчанию
	if config.Backup.Prefix == "" {
		config.Backup.Prefix = "autobackup"
	}
	if config.Backup.Retention == 0 {
		config.Backup.Retention = 14
	}
	return &config, nil
}

// connectToPostgres устанавливает соединение с PostgreSQL
func connectToPostgres(cfg *PostgresConfig) (*sql.DB, error) {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	return db, nil
}

// performBackup выполняет основную логику бэкапа
func performBackup(db *sql.DB, prefix string, retentionDays int, realRun bool) error {
	// Удаление старых бэкапов
	err := deleteOldBackups(db, prefix, retentionDays, realRun)
	if err != nil {
		return fmt.Errorf("ошибка удаления старых бэкапов: %v", err)
	}

	// Получение списка таблиц для бэкапа
	tables, err := getTablesToBackup(db, prefix)
	if err != nil {
		return fmt.Errorf("ошибка получения списка таблиц: %v", err)
	}

	// Создание бэкапов для каждой таблицы
	currentDate := time.Now().Format("20060102")
	for _, table := range tables {
		backupTableName := fmt.Sprintf("%s_%s_%s", prefix, table, currentDate)
		if realRun {
			err := createBackupTable(db, table, backupTableName)
			if err != nil {
				log.Printf("Ошибка создания бэкапа таблицы %s: %v", table, err)
				continue
			}
		}
		log.Printf("Создан бэкап таблицы %s как %s", table, backupTableName)
	}

	return nil
}

// deleteOldBackups удаляет бэкапы старше указанного количества дней
func deleteOldBackups(db *sql.DB, prefix string, retentionDays int, realRun bool) error {
	thresholdDate := time.Now().AddDate(0, 0, -retentionDays)
	threshold := thresholdDate.Format("20060102")

	// Получение списка всех таблиц с префиксом бэкапа
	rows, err := db.Query(`
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = 'public' 
		AND table_name LIKE $1 || '%'`, prefix)
	if err != nil {
		return err
	}
	defer rows.Close()

	var tablesToDelete []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return err
		}

		// Извлечение даты из имени таблицы (последние 8 символов)
		if len(tableName) >= 8 {
			datePart := tableName[len(tableName)-8:]
			if datePart < threshold {
				tablesToDelete = append(tablesToDelete, tableName)
			}
		}
	}

	// Удаление старых таблиц
	for _, table := range tablesToDelete {
		if realRun {
			_, err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", table))
			if err != nil {
				log.Printf("Ошибка удаления таблицы %s: %v", table, err)
				continue
			}
		}
		log.Printf("Удалена старая таблица бэкапа: %s", table)
	}

	return nil
}

// getTablesToBackup возвращает список таблиц, которые нужно бэкапировать
func getTablesToBackup(db *sql.DB, prefix string) ([]string, error) {
	rows, err := db.Query(`
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = 'public' 
		AND table_name NOT LIKE $1 || '%'`, prefix)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, err
		}
		tables = append(tables, tableName)
	}

	return tables, nil
}

// createBackupTable создает копию таблицы
func createBackupTable(db *sql.DB, originalTable, backupTable string) error {
	_, err := db.Exec(fmt.Sprintf("CREATE TABLE %s AS SELECT * FROM %s", backupTable, originalTable))
	return err
}
